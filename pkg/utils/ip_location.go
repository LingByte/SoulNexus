package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// IPLocationService IP地理位置服务
type IPLocationService struct {
	apiURL      string
	timeout     time.Duration
	logger      *zap.Logger
	usePconline bool // 是否使用pconline API（国内IP更准确）
}

// IPLocationResponse IP地理位置查询响应结构（pconline格式）
type IPLocationResponse struct {
	Pro  string `json:"pro"`  // 省份
	City string `json:"city"` // 城市
}

// IPGeolocationResponse IP地理位置API响应（ip-api格式）
type IPGeolocationResponse struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"`
	Status      string  `json:"status"`
	Message     string  `json:"message"`
}

const (
	// PCONLINE_IP_URL IP地址查询API（国内）
	PCONLINE_IP_URL = "http://whois.pconline.com.cn/ipJson.jsp"
	// IP_API_URL IP地址查询API（国际）
	IP_API_URL = "http://ip-api.com/json/"
	// UNKNOWN 未知地址
	UNKNOWN = "Unknown"
	// INTERNAL_IP 内网IP标识
	INTERNAL_IP = "内网IP"
	// LOCAL_NETWORK 本地网络
	LOCAL_NETWORK = "Local Network"
)

// NewIPLocationService 创建IP地理位置服务
func NewIPLocationService(logger *zap.Logger) *IPLocationService {
	return &IPLocationService{
		apiURL:      IP_API_URL,
		timeout:     5 * time.Second,
		logger:      logger,
		usePconline: false, // 默认使用国际API
	}
}

// NewIPLocationServiceWithPconline 创建使用pconline API的IP地理位置服务（国内IP更准确）
func NewIPLocationServiceWithPconline(logger *zap.Logger) *IPLocationService {
	return &IPLocationService{
		apiURL:      PCONLINE_IP_URL,
		timeout:     5 * time.Second,
		logger:      logger,
		usePconline: true,
	}
}

// IsInternalIP 判断是否为内网IP
func IsInternalIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	if parsedIP.IsLoopback() {
		return true
	}
	if parsedIP.IsPrivate() {
		return true
	}
	if parsedIP.IsLinkLocalUnicast() || parsedIP.IsLinkLocalMulticast() {
		return true
	}
	return false
}

// GetLocation 获取IP地理位置（返回国家、城市、完整位置字符串）
func (ils *IPLocationService) GetLocation(ip string) (country, city, location string, err error) {
	// 跳过私有IP
	if IsInternalIP(ip) || ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return "Local", "Local", LOCAL_NETWORK, nil
	}

	if ils.usePconline {
		return ils.getLocationFromPconline(ip)
	}
	return ils.getLocationFromIPAPI(ip)
}

// getLocationFromPconline 从pconline API获取地理位置（国内IP更准确）
func (ils *IPLocationService) getLocationFromPconline(ip string) (country, city, location string, err error) {
	client := &http.Client{
		Timeout: ils.timeout,
	}

	url := fmt.Sprintf("%s?ip=%s&json=true", ils.apiURL, ip)
	resp, err := client.Get(url)
	if err != nil {
		if ils.logger != nil {
			ils.logger.Warn("Failed to get IP location from pconline", zap.String("ip", ip), zap.Error(err))
		}
		return UNKNOWN, UNKNOWN, UNKNOWN, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if ils.logger != nil {
			ils.logger.Warn("Pconline API returned non-200 status", zap.String("ip", ip), zap.Int("status", resp.StatusCode))
		}
		return UNKNOWN, UNKNOWN, UNKNOWN, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if ils.logger != nil {
			ils.logger.Warn("Failed to read pconline response", zap.String("ip", ip), zap.Error(err))
		}
		return UNKNOWN, UNKNOWN, UNKNOWN, nil
	}

	var locationResp IPLocationResponse
	err = json.Unmarshal(body, &locationResp)
	if err != nil {
		if ils.logger != nil {
			ils.logger.Warn("Failed to parse pconline response", zap.String("ip", ip), zap.Error(err))
		}
		return UNKNOWN, UNKNOWN, UNKNOWN, nil
	}

	region := strings.TrimSpace(locationResp.Pro)
	city = strings.TrimSpace(locationResp.City)

	if region == "" {
		region = "未知"
	}
	if city == "" {
		city = "未知"
	}

	// pconline返回的是省份和城市，国家默认为中国
	country = "中国"
	location = fmt.Sprintf("%s %s", region, city)

	return country, city, location, nil
}

// getLocationFromIPAPI 从ip-api.com获取地理位置（国际IP更准确）
func (ils *IPLocationService) getLocationFromIPAPI(ip string) (country, city, location string, err error) {
	url := fmt.Sprintf("%s%s?fields=status,message,country,countryCode,regionName,city,lat,lon,timezone,isp,org,as,query", ils.apiURL, ip)

	client := &http.Client{
		Timeout: ils.timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		if ils.logger != nil {
			ils.logger.Warn("Failed to get IP geolocation", zap.String("ip", ip), zap.Error(err))
		}
		return UNKNOWN, UNKNOWN, UNKNOWN, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if ils.logger != nil {
			ils.logger.Warn("IP geolocation API returned non-200 status", zap.String("ip", ip), zap.Int("status", resp.StatusCode))
		}
		return UNKNOWN, UNKNOWN, UNKNOWN, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if ils.logger != nil {
			ils.logger.Warn("Failed to read IP geolocation response", zap.String("ip", ip), zap.Error(err))
		}
		return UNKNOWN, UNKNOWN, UNKNOWN, nil
	}

	var geoResp IPGeolocationResponse
	if err := json.Unmarshal(body, &geoResp); err != nil {
		if ils.logger != nil {
			ils.logger.Warn("Failed to parse IP geolocation response", zap.String("ip", ip), zap.Error(err))
		}
		return UNKNOWN, UNKNOWN, UNKNOWN, nil
	}

	// 检查API返回状态
	if geoResp.Status == "fail" {
		if ils.logger != nil {
			ils.logger.Warn("IP geolocation API returned fail status", zap.String("ip", ip), zap.String("message", geoResp.Message))
		}
		return UNKNOWN, UNKNOWN, UNKNOWN, nil
	}

	country = geoResp.Country
	if country == "" {
		country = UNKNOWN
	}

	city = geoResp.City
	if city == "" {
		city = UNKNOWN
	}

	location = fmt.Sprintf("%s, %s", city, country)

	return country, city, location, nil
}

// GetRealAddressByIP 根据IP获取真实地址（兼容旧接口，返回完整地址字符串）
func GetRealAddressByIP(ip string) string {
	// 内网不查询
	if IsInternalIP(ip) {
		return INTERNAL_IP
	}

	// 创建临时服务实例（使用默认配置和空logger）
	service := &IPLocationService{
		apiURL:      IP_API_URL,
		timeout:     5 * time.Second,
		logger:      nil, // 在测试中可能为nil
		usePconline: false,
	}
	_, _, location, _ := service.GetLocation(ip)

	if location == UNKNOWN || location == "" {
		return UNKNOWN
	}

	return location
}

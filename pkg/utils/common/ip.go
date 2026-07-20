package common

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func IsIP(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil
}

func ParseIP(s string) net.IP {
	return net.ParseIP(s)
}

func IsPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	private := []net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
	}

	for _, privateNet := range private {
		if privateNet.Contains(ip) {
			return true
		}
	}
	return false
}

func IsIpInCIDRList(ip net.IP, cidrList []string) bool {
	for _, cidr := range cidrList {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			// 尝试作为单个IP处理
			if whitelistIP := net.ParseIP(cidr); whitelistIP != nil {
				if ip.Equal(whitelistIP) {
					return true
				}
			}
			continue
		}

		if network.Contains(ip) {
			return true
		}
	}
	return false
}

const (
	PCONLINE_IP_URL = "http://whois.pconline.com.cn/ipJson.jsp"
	IP_API_URL      = "http://ip-api.com/json/"
	UNKNOWN         = "Unknown"
	INTERNAL_IP     = "内网IP"
	LOCAL_NETWORK   = "Local Network"
)

// LocalNetwork is the display label for internal/private client IPs.
const LocalNetwork = LOCAL_NETWORK

// IPLocationResponse IP 地理位置查询响应（pconline 格式）
type IPLocationResponse struct {
	Pro  string `json:"pro"`
	City string `json:"city"`
}

// IPGeolocationResponse IP 地理位置 API 响应（ip-api 格式）
type IPGeolocationResponse struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	RegionName  string  `json:"regionName"`
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

// IsInternalIP 判断是否为内网IP（纯函数）
func IsInternalIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	if parsedIP.IsLoopback() || parsedIP.IsPrivate() {
		return true
	}

	if parsedIP.IsLinkLocalUnicast() || parsedIP.IsLinkLocalMulticast() {
		return true
	}

	return false
}

// GetIPLocation 获取 IP 地理位置（自动选择国内/国外接口）
// 返回：国家、城市、完整地址、错误
func GetIPLocation(ip string) (string, string, string, error) {
	if IsInternalIP(ip) || ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return "Local", "Local", LOCAL_NETWORK, nil
	}

	if isChinaIP(ip) {
		country, city, loc, err := getIPLocationFromPconline(ip)
		if err == nil && country != UNKNOWN {
			return country, city, loc, nil
		}
		return getIPLocationFromIPAPI(ip)
	}

	return getIPLocationFromIPAPI(ip)
}

// GetIPLocationCN 强制使用国内接口查询 IP 地址（更准）
func GetIPLocationCN(ip string) (string, string, string, error) {
	if IsInternalIP(ip) {
		return "Local", "Local", LOCAL_NETWORK, nil
	}
	country, city, loc, err := getIPLocationFromPconline(ip)
	if err == nil && country != UNKNOWN {
		return country, city, loc, nil
	}
	return getIPLocationFromIPAPI(ip)
}

// GetIPLocationGlobal 强制使用国际接口查询 IP 地址
func GetIPLocationGlobal(ip string) (string, string, string, error) {
	if IsInternalIP(ip) {
		return "Local", "Local", LOCAL_NETWORK, nil
	}
	return getIPLocationFromIPAPI(ip)
}

// GetRealAddressByIP 兼容旧接口：只返回地址字符串
func GetRealAddressByIP(ip string) string {
	if IsInternalIP(ip) {
		return INTERNAL_IP
	}

	_, _, location, _ := GetIPLocation(ip)
	if location == "" || location == UNKNOWN {
		return UNKNOWN
	}
	return location
}

// isChinaIP 简单判断是否国内IP
func isChinaIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || !parsedIP.IsGlobalUnicast() {
		return false
	}
	return strings.HasPrefix(ip, "112.") ||
		strings.HasPrefix(ip, "113.") ||
		strings.HasPrefix(ip, "115.") ||
		strings.HasPrefix(ip, "116.") ||
		strings.HasPrefix(ip, "117.") ||
		strings.HasPrefix(ip, "118.") ||
		strings.HasPrefix(ip, "119.") ||
		strings.HasPrefix(ip, "120.") ||
		strings.HasPrefix(ip, "183.") ||
		strings.HasPrefix(ip, "223.")
}

// getIPLocationFromPconline 从太平洋电脑网获取地址
func getIPLocationFromPconline(ip string) (string, string, string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("%s?ip=%s&json=true", PCONLINE_IP_URL, ip)

	resp, err := client.Get(url)
	if err != nil {
		return UNKNOWN, UNKNOWN, UNKNOWN, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UNKNOWN, UNKNOWN, UNKNOWN, fmt.Errorf("http status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var loc IPLocationResponse
	if err := json.Unmarshal(body, &loc); err != nil {
		return UNKNOWN, UNKNOWN, UNKNOWN, err
	}

	pro := strings.TrimSpace(loc.Pro)
	city := strings.TrimSpace(loc.City)
	if pro == "" {
		pro = UNKNOWN
	}
	if city == "" {
		city = UNKNOWN
	}

	return "中国", city, fmt.Sprintf("%s %s", pro, city), nil
}

// getIPLocationFromIPAPI 从国际 IP-API 获取地址
func getIPLocationFromIPAPI(ip string) (string, string, string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("%s%s?fields=status,message,country,countryCode,regionName,city,lat,lon,timezone,isp,org,as,query", IP_API_URL, ip)

	resp, err := client.Get(url)
	if err != nil {
		return UNKNOWN, UNKNOWN, UNKNOWN, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var geo IPGeolocationResponse
	if err := json.Unmarshal(body, &geo); err != nil {
		return UNKNOWN, UNKNOWN, UNKNOWN, err
	}

	if geo.Status == "fail" {
		return UNKNOWN, UNKNOWN, UNKNOWN, fmt.Errorf("%s", geo.Message)
	}

	country := geo.Country
	city := geo.City
	if country == "" {
		country = UNKNOWN
	}
	if city == "" {
		city = UNKNOWN
	}

	return country, city, fmt.Sprintf("%s, %s", city, country), nil
}

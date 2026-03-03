package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SojsonWeatherResponse sojson 天气 API 响应
type SojsonWeatherResponse struct {
	Message  string `json:"message"`
	Status   int    `json:"status"`
	Date     string `json:"date"`
	Time     string `json:"time"`
	CityInfo struct {
		City       string `json:"city"`
		CityKey    string `json:"citykey"`
		Parent     string `json:"parent"`
		UpdateTime string `json:"updateTime"`
	} `json:"cityInfo"`
	Data struct {
		Shidu   string  `json:"shidu"`   // 湿度
		PM25    float64 `json:"pm25"`    // PM2.5
		PM10    float64 `json:"pm10"`    // PM10
		Quality string  `json:"quality"` // 空气质量
		Wendu   string  `json:"wendu"`   // 温度
		Ganmao  string  `json:"ganmao"`  // 感冒提示
	} `json:"data"`
}

// 城市代码映射（常用城市）
var cityCodes = map[string]string{
	"北京":   "101010100",
	"上海":   "101020100",
	"广州":   "101280101",
	"深圳":   "101280601",
	"成都":   "101270101",
	"杭州":   "101210101",
	"武汉":   "101200101",
	"西安":   "101110101",
	"重庆":   "101040100",
	"天津":   "101030100",
	"南京":   "101190101",
	"苏州":   "101190401",
	"长沙":   "101250101",
	"郑州":   "101180101",
	"沈阳":   "101070101",
	"青岛":   "101120201",
	"宁波":   "101210401",
	"厦门":   "101230201",
	"济南":   "101120101",
	"哈尔滨":  "101050101",
	"福州":   "101230101",
	"昆明":   "101290101",
	"兰州":   "101160101",
	"石家庄":  "101090101",
	"合肥":   "101220101",
	"南昌":   "101240101",
	"太原":   "101100101",
	"贵阳":   "101260101",
	"南宁":   "101300101",
	"海口":   "101310101",
	"乌鲁木齐": "101130101",
	"拉萨":   "101140101",
	"银川":   "101170101",
	"西宁":   "101150101",
	"呼和浩特": "101080101",
}

// GetWeather 获取天气信息（使用 sojson.com 免费 API）
// 完全免费，无需 API Key，由又拍云提供 CDN 加速
func GetWeather(city string) (string, error) {
	// 获取城市代码
	cityCode, ok := cityCodes[city]
	if !ok {
		return "", fmt.Errorf("暂不支持查询 %s 的天气，目前支持的城市：北京、上海、广州、深圳、成都、杭州、武汉、西安、重庆等主要城市", city)
	}

	// 构建请求 URL
	apiURL := fmt.Sprintf("http://t.weather.sojson.com/api/weather/city/%s", cityCode)

	client := &http.Client{
		Timeout: 3 * time.Second, // 3秒超时，这个 API 很快
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("请求天气 API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("天气 API 返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	var weatherResp SojsonWeatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		return "", fmt.Errorf("解析天气数据失败: %w", err)
	}

	// 检查 API 返回状态
	if weatherResp.Status != 200 {
		return "", fmt.Errorf("天气 API 返回错误: %s", weatherResp.Message)
	}

	// 格式化返回结果
	result := formatSojsonWeatherResult(&weatherResp)
	return result, nil
}

// formatSojsonWeatherResult 格式化 sojson 天气结果
func formatSojsonWeatherResult(weather *SojsonWeatherResponse) string {
	return fmt.Sprintf(
		"%s·%s当前天气：温度%s℃，湿度%s，空气质量%s（PM2.5: %.0f），%s",
		weather.CityInfo.Parent,
		weather.CityInfo.City,
		weather.Data.Wendu,
		weather.Data.Shidu,
		weather.Data.Quality,
		weather.Data.PM25,
		weather.Data.Ganmao,
	)
}

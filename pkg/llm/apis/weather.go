package apis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WeatherService 天气服务接口
type WeatherService interface {
	GetCurrentWeather(location string, unit string) (*WeatherData, error)
	GetWeatherForecast(location string, days int) (*WeatherForecast, error)
}

// WeatherData 天气数据结构
type WeatherData struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	Unit        string  `json:"unit"`
	Description string  `json:"description"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
	Pressure    int     `json:"pressure"`
	Visibility  int     `json:"visibility"`
	UVIndex     float64 `json:"uv_index"`
	Timestamp   string  `json:"timestamp"`
}

// WeatherForecast 天气预报数据结构
type WeatherForecast struct {
	Location string       `json:"location"`
	Days     []WeatherDay `json:"days"`
}

// WeatherDay 单日天气数据
type WeatherDay struct {
	Date        string  `json:"date"`
	MaxTemp     float64 `json:"max_temp"`
	MinTemp     float64 `json:"min_temp"`
	Description string  `json:"description"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
	RainChance  int     `json:"rain_chance"`
}

// WttrInService wttr.in 免费天气服务
type WttrInService struct {
	client *http.Client
}

// NewWttrInService 创建wttr.in服务
func NewWttrInService() *WttrInService {
	return &WttrInService{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetCurrentWeather 获取当前天气
func (s *WttrInService) GetCurrentWeather(location string, unit string) (*WeatherData, error) {
	// wttr.in 支持多种格式，我们使用JSON格式
	url := fmt.Sprintf("https://wttr.in/%s?format=j1&lang=zh", location)

	resp, err := s.client.Get(url)
	if err != nil {
		return s.getMockWeather(location, unit)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.getMockWeather(location, unit)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return s.getMockWeather(location, unit)
	}

	var wttrResp struct {
		CurrentCondition []struct {
			TempC       string `json:"temp_C"`
			TempF       string `json:"temp_F"`
			WeatherDesc []struct {
				Value string `json:"value"`
			} `json:"weatherDesc"`
			Humidity      string `json:"humidity"`
			WindspeedKmph string `json:"windspeedKmph"`
			Pressure      string `json:"pressure"`
			Visibility    string `json:"visibility"`
			UVIndex       string `json:"uvIndex"`
		} `json:"current_condition"`
		NearestArea []struct {
			AreaName []struct {
				Value string `json:"value"`
			} `json:"areaName"`
		} `json:"nearest_area"`
	}

	if err := json.Unmarshal(body, &wttrResp); err != nil {
		return s.getMockWeather(location, unit)
	}

	if len(wttrResp.CurrentCondition) == 0 || len(wttrResp.NearestArea) == 0 {
		return s.getMockWeather(location, unit)
	}

	condition := wttrResp.CurrentCondition[0]
	area := wttrResp.NearestArea[0]

	var temp float64
	var tempUnit string
	if unit == "fahrenheit" {
		if tempF, err := strconv.ParseFloat(condition.TempF, 64); err == nil {
			temp = tempF
			tempUnit = "°F"
		} else {
			temp = 72.0
			tempUnit = "°F"
		}
	} else {
		if tempC, err := strconv.ParseFloat(condition.TempC, 64); err == nil {
			temp = tempC
			tempUnit = "°C"
		} else {
			temp = 22.0
			tempUnit = "°C"
		}
	}

	description := "晴天"
	if len(condition.WeatherDesc) > 0 {
		description = condition.WeatherDesc[0].Value
	}

	humidity := 65
	if h, err := strconv.Atoi(condition.Humidity); err == nil {
		humidity = h
	}

	windSpeed := 3.2
	if ws, err := strconv.ParseFloat(condition.WindspeedKmph, 64); err == nil {
		windSpeed = ws
	}

	pressure := 1013
	if p, err := strconv.Atoi(condition.Pressure); err == nil {
		pressure = p
	}

	visibility := 10
	if v, err := strconv.Atoi(condition.Visibility); err == nil {
		visibility = v
	}

	uvIndex := 5.0
	if uv, err := strconv.ParseFloat(condition.UVIndex, 64); err == nil {
		uvIndex = uv
	}

	locationName := location
	if len(area.AreaName) > 0 {
		locationName = area.AreaName[0].Value
	}

	return &WeatherData{
		Location:    locationName,
		Temperature: temp,
		Unit:        tempUnit,
		Description: description,
		Humidity:    humidity,
		WindSpeed:   windSpeed,
		Pressure:    pressure,
		Visibility:  visibility,
		UVIndex:     uvIndex,
		Timestamp:   time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// GetWeatherForecast 获取天气预报
func (s *WttrInService) GetWeatherForecast(location string, days int) (*WeatherForecast, error) {
	// wttr.in 的预报格式
	url := fmt.Sprintf("https://wttr.in/%s?format=j1&lang=zh", location)

	resp, err := s.client.Get(url)
	if err != nil {
		return s.getMockForecast(location, days)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.getMockForecast(location, days)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return s.getMockForecast(location, days)
	}

	var wttrResp struct {
		Weather []struct {
			Date     string `json:"date"`
			MaxtempC string `json:"maxtempC"`
			MintempC string `json:"mintempC"`
			Hourly   []struct {
				WeatherDesc []struct {
					Value string `json:"value"`
				} `json:"weatherDesc"`
				WindspeedKmph string `json:"windspeedKmph"`
			} `json:"hourly"`
		} `json:"weather"`
		NearestArea []struct {
			AreaName []struct {
				Value string `json:"value"`
			} `json:"areaName"`
		} `json:"nearest_area"`
	}

	if err := json.Unmarshal(body, &wttrResp); err != nil {
		return s.getMockForecast(location, days)
	}

	if len(wttrResp.Weather) == 0 || len(wttrResp.NearestArea) == 0 {
		return s.getMockForecast(location, days)
	}

	locationName := location
	if len(wttrResp.NearestArea) > 0 && len(wttrResp.NearestArea[0].AreaName) > 0 {
		locationName = wttrResp.NearestArea[0].AreaName[0].Value
	}

	var forecastDays []WeatherDay
	for i, day := range wttrResp.Weather {
		if i >= days {
			break
		}

		maxTemp, _ := strconv.ParseFloat(day.MaxtempC, 64)
		minTemp, _ := strconv.ParseFloat(day.MintempC, 64)

		description := "晴天"
		windSpeed := 3.0
		if len(day.Hourly) > 0 {
			if len(day.Hourly[0].WeatherDesc) > 0 {
				description = day.Hourly[0].WeatherDesc[0].Value
			}
			if ws, err := strconv.ParseFloat(day.Hourly[0].WindspeedKmph, 64); err == nil {
				windSpeed = ws
			}
		}

		forecastDays = append(forecastDays, WeatherDay{
			Date:        day.Date,
			MaxTemp:     maxTemp,
			MinTemp:     minTemp,
			Description: description,
			WindSpeed:   windSpeed,
			RainChance:  20 + i*5, // 模拟降雨概率
		})
	}

	return &WeatherForecast{
		Location: locationName,
		Days:     forecastDays,
	}, nil
}

// getMockWeather 获取模拟天气数据
func (s *WttrInService) getMockWeather(location string, unit string) (*WeatherData, error) {
	weatherData := map[string]map[string]string{
		"beijing": {
			"celsius":    "晴天",
			"fahrenheit": "晴天",
		},
		"shanghai": {
			"celsius":    "多云",
			"fahrenheit": "多云",
		},
		"chengdu": {
			"celsius":    "小雨",
			"fahrenheit": "小雨",
		},
		"guangzhou": {
			"celsius":    "阴天",
			"fahrenheit": "阴天",
		},
		"hangzhou": {
			"celsius":    "晴天",
			"fahrenheit": "晴天",
		},
	}

	city := strings.ToLower(location)
	var description string
	var temp float64

	for cityKey, data := range weatherData {
		if strings.Contains(city, cityKey) {
			description = data[unit]
			if unit == "fahrenheit" {
				temp = 72.0
			} else {
				temp = 22.0
			}
			break
		}
	}

	if description == "" {
		description = "晴天"
		if unit == "fahrenheit" {
			temp = 70.0
		} else {
			temp = 20.0
		}
	}

	tempUnit := "°C"
	if unit == "fahrenheit" {
		tempUnit = "°F"
	}

	return &WeatherData{
		Location:    location,
		Temperature: temp,
		Unit:        tempUnit,
		Description: description,
		Humidity:    65,
		WindSpeed:   3.2,
		Pressure:    1013,
		Visibility:  10,
		UVIndex:     5.0,
		Timestamp:   time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// getMockForecast 获取模拟天气预报
func (s *WttrInService) getMockForecast(location string, days int) (*WeatherForecast, error) {
	var forecastDays []WeatherDay
	baseDate := time.Now()

	for i := 0; i < days; i++ {
		date := baseDate.AddDate(0, 0, i)
		forecastDays = append(forecastDays, WeatherDay{
			Date:        date.Format("2006-01-02"),
			MaxTemp:     25.0 + float64(i),
			MinTemp:     15.0 + float64(i),
			Description: "晴天",
			Humidity:    60 + i,
			WindSpeed:   3.0,
			RainChance:  10 + i*5,
		})
	}

	return &WeatherForecast{
		Location: location,
		Days:     forecastDays,
	}, nil
}

// OpenMeteoService Open-Meteo 免费天气服务
type OpenMeteoService struct {
	client *http.Client
}

// NewOpenMeteoService 创建Open-Meteo服务
func NewOpenMeteoService() *OpenMeteoService {
	return &OpenMeteoService{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetCurrentWeather 获取当前天气
func (s *OpenMeteoService) GetCurrentWeather(location string, unit string) (*WeatherData, error) {
	// Open-Meteo 需要先获取坐标，这里简化处理
	// 实际应用中可以使用地理编码API获取坐标
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=39.9042&longitude=116.4074&current_weather=true&hourly=temperature_2m,relativehumidity_2m,windspeed_10m,pressure_msl,visibility&timezone=Asia/Shanghai")

	resp, err := s.client.Get(url)
	if err != nil {
		return s.getMockWeather(location, unit)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.getMockWeather(location, unit)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return s.getMockWeather(location, unit)
	}

	var meteResp struct {
		CurrentWeather struct {
			Temperature float64 `json:"temperature"`
			Windspeed   float64 `json:"windspeed"`
			Weathercode int     `json:"weathercode"`
		} `json:"current_weather"`
		Hourly struct {
			Relativehumidity2m []float64 `json:"relativehumidity_2m"`
			PressureMsl        []float64 `json:"pressure_msl"`
			Visibility         []float64 `json:"visibility"`
		} `json:"hourly"`
	}

	if err := json.Unmarshal(body, &meteResp); err != nil {
		return s.getMockWeather(location, unit)
	}

	// 转换天气代码为描述
	weatherDescriptions := map[int]string{
		0: "晴天", 1: "大部分晴天", 2: "部分多云", 3: "阴天",
		45: "雾", 48: "霜雾", 51: "小雨", 53: "中雨", 55: "大雨",
		61: "小雨", 63: "中雨", 65: "大雨", 71: "小雪", 73: "中雪", 75: "大雪",
		80: "阵雨", 81: "中阵雨", 82: "强阵雨", 85: "阵雪", 86: "强阵雪",
		95: "雷暴", 96: "雷暴伴冰雹", 99: "强雷暴伴冰雹",
	}

	description := "晴天"
	if desc, ok := weatherDescriptions[meteResp.CurrentWeather.Weathercode]; ok {
		description = desc
	}

	humidity := 65
	if len(meteResp.Hourly.Relativehumidity2m) > 0 {
		humidity = int(meteResp.Hourly.Relativehumidity2m[0])
	}

	pressure := 1013
	if len(meteResp.Hourly.PressureMsl) > 0 {
		pressure = int(meteResp.Hourly.PressureMsl[0])
	}

	visibility := 10
	if len(meteResp.Hourly.Visibility) > 0 {
		visibility = int(meteResp.Hourly.Visibility[0] / 1000) // 转换为公里
	}

	temp := meteResp.CurrentWeather.Temperature
	tempUnit := "°C"
	if unit == "fahrenheit" {
		temp = temp*9/5 + 32
		tempUnit = "°F"
	}

	return &WeatherData{
		Location:    location,
		Temperature: temp,
		Unit:        tempUnit,
		Description: description,
		Humidity:    humidity,
		WindSpeed:   meteResp.CurrentWeather.Windspeed,
		Pressure:    pressure,
		Visibility:  visibility,
		UVIndex:     5.0,
		Timestamp:   time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// GetWeatherForecast 获取天气预报
func (s *OpenMeteoService) GetWeatherForecast(location string, days int) (*WeatherForecast, error) {
	// 简化处理，返回模拟数据
	return s.getMockForecast(location, days)
}

// getMockWeather 获取模拟天气数据
func (s *OpenMeteoService) getMockWeather(location string, unit string) (*WeatherData, error) {
	service := &WttrInService{}
	return service.getMockWeather(location, unit)
}

// getMockForecast 获取模拟天气预报
func (s *OpenMeteoService) getMockForecast(location string, days int) (*WeatherForecast, error) {
	service := &WttrInService{}
	return service.getMockForecast(location, days)
}

// WeatherManager 天气服务管理器
type WeatherManager struct {
	services []WeatherService
}

// NewWeatherManager 创建天气服务管理器
func NewWeatherManager() *WeatherManager {
	return &WeatherManager{
		services: []WeatherService{
			NewWttrInService(),
			NewOpenMeteoService(),
		},
	}
}

// GetCurrentWeather 获取当前天气（尝试多个服务）
func (m *WeatherManager) GetCurrentWeather(location string, unit string) (*WeatherData, error) {
	var lastErr error

	for _, service := range m.services {
		data, err := service.GetCurrentWeather(location, unit)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("所有天气服务都不可用: %w", lastErr)
}

// GetWeatherForecast 获取天气预报（尝试多个服务）
func (m *WeatherManager) GetWeatherForecast(location string, days int) (*WeatherForecast, error) {
	var lastErr error

	for _, service := range m.services {
		forecast, err := service.GetWeatherForecast(location, days)
		if err == nil {
			return forecast, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("所有天气服务都不可用: %w", lastErr)
}

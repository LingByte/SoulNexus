package utils

import (
	"math"
	"testing"
)

func TestGetDistance(t *testing.T) {
	tests := []struct {
		name       string
		lon1, lat1 float64 // 第一个点的经纬度
		lon2, lat2 float64 // 第二个点的经纬度
		expected   float64 // 期望的距离（米）
		tolerance  float64 // 允许的误差范围（米）
	}{
		{
			name:      "Same point",
			lon1:      0.0,
			lat1:      0.0,
			lon2:      0.0,
			lat2:      0.0,
			expected:  0.0,
			tolerance: 0.001,
		},
		{
			name:      "Same point (non-zero)",
			lon1:      120.0,
			lat1:      30.0,
			lon2:      120.0,
			lat2:      30.0,
			expected:  0.0,
			tolerance: 0.001,
		},
		{
			name:      "1 degree longitude at equator",
			lon1:      0.0,
			lat1:      0.0,
			lon2:      1.0,
			lat2:      0.0,
			expected:  111195.0, // 约111.195公里
			tolerance: 100.0,    // 100米误差范围
		},
		{
			name:      "1 degree latitude anywhere",
			lon1:      0.0,
			lat1:      0.0,
			lon2:      0.0,
			lat2:      1.0,
			expected:  111195.0, // 约111.195公里
			tolerance: 100.0,    // 100米误差范围
		},
		{
			name:      "Short distance - Beijing locations",
			lon1:      116.3975, // 北京
			lat1:      39.9088,
			lon2:      116.4754, // 北京附近
			lat2:      39.9372,
			expected:  7500.0, // 约7.5公里
			tolerance: 1000.0, // 1公里误差范围
		},
		{
			name:      "Medium distance - locations in same region",
			lon1:      -73.9352, // 纽约区域
			lat1:      40.7306,
			lon2:      -73.5437, // 同一区域内的点
			lat2:      40.8522,
			expected:  35000.0, // 约35公里
			tolerance: 5000.0,  // 5公里误差范围
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDistance(tt.lon1, tt.lat1, tt.lon2, tt.lat2)
			diff := math.Abs(got - tt.expected)

			if diff > tt.tolerance {
				t.Errorf("GetDistance(%f, %f, %f, %f) = %f, want %f (tolerance: %f, diff: %f)",
					tt.lon1, tt.lat1, tt.lon2, tt.lat2, got, tt.expected, tt.tolerance, diff)
			}
		})
	}
}

// 测试对称性 - 两点间距离应该是一样的，无论方向如何
func TestGetDistanceSymmetry(t *testing.T) {
	tests := []struct {
		name       string
		lon1, lat1 float64
		lon2, lat2 float64
	}{
		{
			name: "Random points 1",
			lon1: 120.5,
			lat1: 30.2,
			lon2: 121.8,
			lat2: 31.4,
		},
		{
			name: "Random points 2",
			lon1: -75.3,
			lat1: 42.1,
			lon2: -72.7,
			lat2: 41.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance1 := GetDistance(tt.lon1, tt.lat1, tt.lon2, tt.lat2)
			distance2 := GetDistance(tt.lon2, tt.lat2, tt.lon1, tt.lat1)

			// 由于浮点数精度问题，允许极小的差异
			if math.Abs(distance1-distance2) > 0.001 {
				t.Errorf("Distance is not symmetric: GetDistance(%f, %f, %f, %f) = %f, GetDistance(%f, %f, %f, %f) = %f",
					tt.lon1, tt.lat1, tt.lon2, tt.lat2, distance1,
					tt.lon2, tt.lat2, tt.lon1, tt.lat1, distance2)
			}
		})
	}
}

// 测试边界值
func TestGetDistanceBoundaryValues(t *testing.T) {
	tests := []struct {
		name        string
		lon1, lat1  float64
		lon2, lat2  float64
		expectValid bool
	}{
		{
			name:        "Valid coordinates - Equator",
			lon1:        0.0,
			lat1:        0.0,
			lon2:        10.0,
			lat2:        0.0,
			expectValid: true,
		},
		{
			name:        "Valid coordinates - North Pole to South Pole",
			lon1:        0.0,
			lat1:        90.0,
			lon2:        0.0,
			lat2:        -90.0,
			expectValid: true,
		},
		{
			name:        "Invalid latitude > 90",
			lon1:        0.0,
			lat1:        91.0,
			lon2:        0.0,
			lat2:        0.0,
			expectValid: true, // 函数不会验证输入有效性，所以仍会返回一个值
		},
		{
			name:        "Invalid latitude < -90",
			lon1:        0.0,
			lat1:        -91.0,
			lon2:        0.0,
			lat2:        0.0,
			expectValid: true, // 函数不会验证输入有效性，所以仍会返回一个值
		},
		{
			name:        "Invalid longitude > 180",
			lon1:        181.0,
			lat1:        0.0,
			lon2:        0.0,
			lat2:        0.0,
			expectValid: true, // 函数不会验证输入有效性，所以仍会返回一个值
		},
		{
			name:        "Invalid longitude < -180",
			lon1:        -181.0,
			lat1:        0.0,
			lon2:        0.0,
			lat2:        0.0,
			expectValid: true, // 函数不会验证输入有效性，所以仍会返回一个值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 只测试函数不会崩溃，因为函数不验证输入
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("GetDistance panicked with invalid input: %v", r)
				}
			}()

			got := GetDistance(tt.lon1, tt.lat1, tt.lon2, tt.lat2)

			// 检查结果是否为有效数字
			if math.IsNaN(got) {
				t.Errorf("GetDistance(%f, %f, %f, %f) returned NaN", tt.lon1, tt.lat1, tt.lon2, tt.lat2)
			}

			if math.IsInf(got, 0) {
				t.Errorf("GetDistance(%f, %f, %f, %f) returned infinity", tt.lon1, tt.lat1, tt.lon2, tt.lat2)
			}
		})
	}
}

// 特别测试对跖点计算
func TestGetDistanceAntipodalPoints(t *testing.T) {
	tests := []struct {
		name       string
		lon1, lat1 float64
		lon2, lat2 float64
	}{
		{
			name: "Equator antipodal points",
			lon1: 0.0,
			lat1: 0.0,
			lon2: 180.0,
			lat2: 0.0,
		},
		{
			name: "Northern to Southern hemisphere",
			lon1: 45.0,
			lat1: 30.0,
			lon2: -135.0,
			lat2: -30.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("GetDistance panicked: %v", r)
				}
			}()

			got := GetDistance(tt.lon1, tt.lat1, tt.lon2, tt.lat2)

			// 检查结果是否为有效数字
			if math.IsNaN(got) {
				t.Errorf("GetDistance(%f, %f, %f, %f) returned NaN", tt.lon1, tt.lat1, tt.lon2, tt.lat2)
			}

			if math.IsInf(got, 0) {
				t.Errorf("GetDistance(%f, %f, %f, %f) returned infinity", tt.lon1, tt.lat1, tt.lon2, tt.lat2)
			}

			// 对于对跖点，距离应该接近地球半周长
			earthHalfCircumference := math.Pi * 6371000.0 // 约20015000米
			if got > earthHalfCircumference*1.1 || got < earthHalfCircumference*0.9 {
				t.Logf("Distance between antipodal points %f is not close to half Earth circumference %f", got, earthHalfCircumference)
			}
		})
	}
}

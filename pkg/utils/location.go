package utils

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net"
	"strings"

	"go.uber.org/zap"
)

// cityEnToZh 常用城市英文名 → 中文名映射（用于 ip-api 降级时转换）
var cityEnToZh = map[string]string{
	"shanghai": "上海", "beijing": "北京", "guangzhou": "广州", "shenzhen": "深圳",
	"chengdu": "成都", "chongqing": "重庆", "hangzhou": "杭州", "nanjing": "南京",
	"wuhan": "武汉", "tianjin": "天津", "xi'an": "西安", "xian": "西安",
	"suzhou": "苏州", "ningbo": "宁波", "qingdao": "青岛", "jinan": "济南",
	"zhengzhou": "郑州", "changsha": "长沙", "kunming": "昆明", "harbin": "哈尔滨",
	"shenyang": "沈阳", "dalian": "大连", "fuzhou": "福州", "xiamen": "厦门",
	"hefei": "合肥", "nanchang": "南昌", "guiyang": "贵阳", "nanning": "南宁",
	"lanzhou": "兰州", "yinchuan": "银川", "xining": "西宁", "urumqi": "乌鲁木齐",
	"lhasa": "拉萨", "hohhot": "呼和浩特", "shijiazhuang": "石家庄", "taiyuan": "太原",
	"changchun": "长春", "jilin": "吉林", "haikou": "海口", "sanya": "三亚",
	"wenzhou": "温州", "dongguan": "东莞", "foshan": "佛山", "zhuhai": "珠海",
}

// translateCityToZh 将英文城市名转换为中文（不区分大小写，包内使用）
func translateCityToZh(city string) string {
	key := strings.ToLower(strings.TrimSpace(city))
	if zh, ok := cityEnToZh[key]; ok {
		return zh
	}
	return city
}

// TranslateCityToZh 将英文城市名转换为中文（导出版本，供其他包使用）
func TranslateCityToZh(city string) string {
	return translateCityToZh(city)
}

// GetLocationFromAddr 从 RemoteAddr（host:port 格式）提取 IP 并查询地理位置，返回中文位置字符串。
// 优先使用 pconline（国内省市准确），失败时降级到 ip-api 并将城市名转为中文。
// 内网 IP 返回空字符串。
func GetLocationFromAddr(remoteAddr string, logger *zap.Logger) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = strings.TrimSpace(remoteAddr)
	}
	if host == "" || IsInternalIP(host) {
		return ""
	}

	// 优先 pconline（返回格式：省份 城市，全中文）
	svc := NewIPLocationServiceWithPconline(logger)
	_, _, location, _ := svc.GetLocation(host)
	if location != "" && location != UNKNOWN && location != LOCAL_NETWORK {
		return location
	}

	// 降级到 ip-api，返回格式：City, Country（英文）
	svc2 := NewIPLocationService(logger)
	_, city, location2, _ := svc2.GetLocation(host)
	if location2 == "" || location2 == UNKNOWN {
		return ""
	}

	// 尝试把城市名翻译成中文
	zhCity := translateCityToZh(city)
	if zhCity != city {
		// 翻译成功，只返回中文城市名（不带英文国家）
		return zhCity
	}
	return location2
}

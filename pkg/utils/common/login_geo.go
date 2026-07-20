package common

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
)

// LoginGeo holds parsed IP geolocation for login auditing.
type LoginGeo struct {
	Country  string
	City     string
	Location string
}

// LoginGeoFromIP resolves country, city and display location for a client IP.
func LoginGeoFromIP(ip string) LoginGeo {
	country, city, location, err := GetIPLocation(strings.TrimSpace(ip))
	if err != nil {
		return LoginGeo{}
	}
	country = normalizeLoginGeoPart(country)
	city = normalizeLoginGeoPart(city)
	location = strings.TrimSpace(location)
	if location == "" || location == "Unknown" || location == LocalNetwork {
		location = ""
	}
	return LoginGeo{Country: country, City: city, Location: location}
}

// LoginCityFromIP returns the city component for a client IP.
func LoginCityFromIP(ip string) string {
	return LoginGeoFromIP(ip).City
}

func normalizeLoginGeoPart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "Unknown" || s == "Local" {
		return ""
	}
	return s
}

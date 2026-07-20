package common

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"strings"
)

const MaxKnownLoginCities = 5

func ParseKnownLoginCities(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	seen := make(map[string]struct{}, len(out))
	uniq := make([]string, 0, len(out))
	for _, c := range out {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		uniq = append(uniq, c)
	}
	return uniq
}

func IsKnownLoginCity(cities []string, city string) bool {
	city = strings.TrimSpace(city)
	if city == "" {
		return true
	}
	for _, c := range cities {
		if c == city {
			return true
		}
	}
	return false
}

// NeedsRemoteLoginVerify reports whether login from currentCity should require extra verification.
func NeedsRemoteLoginVerify(require bool, knownCities []string, currentCity string) bool {
	if !require || currentCity == "" {
		return false
	}
	if len(knownCities) == 0 {
		return false
	}
	return !IsKnownLoginCity(knownCities, currentCity)
}

func AddKnownLoginCity(raw, city string) string {
	city = strings.TrimSpace(city)
	if city == "" || city == "Local" {
		return raw
	}
	cities := ParseKnownLoginCities(raw)
	if IsKnownLoginCity(cities, city) {
		return raw
	}
	cities = append([]string{city}, cities...)
	if len(cities) > MaxKnownLoginCities {
		cities = cities[:MaxKnownLoginCities]
	}
	b, err := json.Marshal(cities)
	if err != nil {
		return raw
	}
	return string(b)
}

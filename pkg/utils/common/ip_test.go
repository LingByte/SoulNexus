package common

import (
	"net"
	"testing"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

func TestIsIP(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"192.168.1.1", true},
		{"::1", true},
		{"invalid", false},
		{"", false},
		{"256.0.0.1", false},
	}
	for _, tt := range tests {
		got := IsIP(tt.input)
		if got != tt.want {
			t.Errorf("IsIP(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseIP(t *testing.T) {
	got := ParseIP("192.168.1.1")
	if got == nil {
		t.Fatal("ParseIP should return non-nil for valid IP")
	}
	got = ParseIP("invalid")
	if got != nil {
		t.Fatal("ParseIP should return nil for invalid IP")
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"127.0.0.1", true},
		{"8.8.8.8", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.input)
		got := IsPrivateIP(ip)
		if got != tt.want {
			t.Errorf("IsPrivateIP(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsInternalIP(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"8.8.8.8", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		got := IsInternalIP(tt.input)
		if got != tt.want {
			t.Errorf("IsInternalIP(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsIpInCIDRList(t *testing.T) {
	ip := net.ParseIP("192.168.1.100")
	cidrList := []string{"192.168.1.0/24", "10.0.0.0/8"}
	if !IsIpInCIDRList(ip, cidrList) {
		t.Error("should be in CIDR list")
	}

	ip = net.ParseIP("8.8.8.8")
	if IsIpInCIDRList(ip, cidrList) {
		t.Error("should not be in CIDR list")
	}
}

func TestIsIpInCIDRList_InvalidCIDR(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	cidrList := []string{"invalid", "192.168.1.0/24"}
	if !IsIpInCIDRList(ip, cidrList) {
		t.Error("should be in CIDR list despite invalid entry")
	}
}

func TestGetRealAddressByIP_Internal(t *testing.T) {
	got := GetRealAddressByIP("127.0.0.1")
	if got != INTERNAL_IP {
		t.Errorf("got %q, want %q", got, INTERNAL_IP)
	}
}

func TestGetIPLocation_Internal(t *testing.T) {
	country, city, location, err := GetIPLocation("127.0.0.1")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if country != "Local" {
		t.Errorf("country = %q", country)
	}
	if city != "Local" {
		t.Errorf("city = %q", city)
	}
	if location != LocalNetwork {
		t.Errorf("location = %q", location)
	}
}

func TestGetIPLocationCN_Internal(t *testing.T) {
	country, city, location, err := GetIPLocationCN("10.0.0.1")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if country != "Local" {
		t.Errorf("country = %q", country)
	}
	if city != "Local" {
		t.Errorf("city = %q", city)
	}
	if location != LocalNetwork {
		t.Errorf("location = %q", location)
	}
}

func TestGetIPLocationGlobal_Internal(t *testing.T) {
	country, city, location, err := GetIPLocationGlobal("192.168.1.1")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if country != "Local" {
		t.Errorf("country = %q", country)
	}
	if city != "Local" {
		t.Errorf("city = %q", city)
	}
	if location != LocalNetwork {
		t.Errorf("location = %q", location)
	}
}

func TestIsChinaIP(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"112.0.0.1", true},
		{"183.0.0.1", true},
		{"8.8.8.8", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		got := isChinaIP(tt.input)
		if got != tt.want {
			t.Errorf("isChinaIP(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

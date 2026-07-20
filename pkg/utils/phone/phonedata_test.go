package phone

import (
	"fmt"
	"testing"
)

func TestFindInvalidPhone(t *testing.T) {
	cases := []string{
		"13580198235123123213213",
		"1300",
		"10074872323",
		"afsd32323",
	}
	for _, phoneNum := range cases {
		if _, err := Find(phoneNum); err == nil {
			t.Fatalf("expected error for %q", phoneNum)
		}
	}
}

func TestFindKnownSegments(t *testing.T) {
	cases := []struct {
		phone    string
		expected string
	}{
		{"1952947", "1952947,广西,玉林,中国移动"},
		{"1669981", "1669981,新疆,乌鲁木齐,中国联通"},
		{"1921306", "1921306,河北,保定,中国广电"},
		{"1936137", "1936137,广东,广州,中国电信"},
		{"1903845", "1903845,贵州,遵义,中国电信"},
	}
	for _, tc := range cases {
		info, err := Find(tc.phone)
		if err != nil {
			t.Fatalf("Find(%q): %v", tc.phone, err)
		}
		got := fmt.Sprintf("%s,%s,%s,%s", info.PhoneNum, info.Province, info.City, info.CardType)
		if got != tc.expected {
			t.Fatalf("Find(%q) = %q, want %q", tc.phone, got, tc.expected)
		}
	}
}

func TestFindPhone1703576(t *testing.T) {
	if _, err := Find("1703576"); err != nil {
		t.Fatal(err)
	}
}

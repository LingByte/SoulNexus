package phone

import (
	"strings"
	"testing"
)

func TestFormatPhoneLocation(t *testing.T) {
	got := FormatPhoneLocation("四川", "成都", "中国移动")
	if got != "四川成都(中国移动)" {
		t.Fatalf("got %q", got)
	}
	got = FormatPhoneLocation("上海", "上海", "中国电信")
	if got != "上海(中国电信)" {
		t.Fatalf("duplicate province/city: got %q", got)
	}
	// 新增：广电测试用例1：省市不同
	got = FormatPhoneLocation("河南", "郑州", "中国广电")
	if got != "河南郑州(中国广电)" {
		t.Fatalf("china radio diff city got %q", got)
	}
	// 新增：广电测试用例2：省市同名（直辖市场景）
	got = FormatPhoneLocation("重庆", "重庆", "中国广电")
	if got != "重庆(中国广电)" {
		t.Fatalf("china radio same city got %q", got)
	}

	if FormatPhoneLocation("", "", "") != "" {
		t.Fatal("empty")
	}
}

func TestLookupPhoneLocation(t *testing.T) {
	got := LookupPhoneLocation("19511899044")
	if got == "" {
		t.Fatal("expected lookup result")
	}
	if !strings.Contains(got, "成都") {
		t.Fatalf("got %q", got)
	}

	// 新增广电192号段真实号码测试
	radioPhone := "19208101234" // 192号段，归属地四川成都
	got = LookupPhoneLocation(radioPhone)
	if got == "" {
		t.Fatalf("广电号码没有解析出结果")
	}
	if !strings.Contains(got, "成都") || !strings.Contains(got, "中国广电") {
		t.Fatalf("广电解析错误 got = %s", got)
	}

	if LookupPhoneLocation("123") != "" {
		t.Fatal("short number")
	}
	// 额外加一个非法短号码兜底
	if LookupPhoneLocation("192") != "" {
		t.Fatal("short radio number should empty")
	}
}

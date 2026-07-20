package common

import "testing"

func TestParseKnownLoginCities(t *testing.T) {
	if got := ParseKnownLoginCities(`["上海","北京"]`); len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if IsKnownLoginCity(ParseKnownLoginCities(`["上海"]`), "上海") != true {
		t.Fatal("expected known city")
	}
	if IsKnownLoginCity(ParseKnownLoginCities(`["上海"]`), "广州") != false {
		t.Fatal("expected unknown city")
	}
}

func TestAddKnownLoginCity(t *testing.T) {
	raw := AddKnownLoginCity("", "上海")
	if !IsKnownLoginCity(ParseKnownLoginCities(raw), "上海") {
		t.Fatal("city not added")
	}
	raw = AddKnownLoginCity(raw, "上海")
	if len(ParseKnownLoginCities(raw)) != 1 {
		t.Fatal("duplicate city added")
	}
}

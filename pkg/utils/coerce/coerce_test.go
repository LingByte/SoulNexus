package coerce

import (
	"encoding/json"
	"testing"
)

func TestFloat64FromAny(t *testing.T) {
	f, ok := Float64FromAny(float32(1.5))
	if !ok || f != 1.5 {
		t.Fatalf("float32: ok=%v f=%v", ok, f)
	}
	f, ok = Float64FromAny(json.Number("2.25"))
	if !ok || f != 2.25 {
		t.Fatalf("json.Number: ok=%v f=%v", ok, f)
	}
	_, ok = Float64FromAny(true)
	if ok {
		t.Fatal("expected false for bool")
	}
}

func TestIntFromAny(t *testing.T) {
	n, ok := IntFromAny(json.Number("7"))
	if !ok || n != 7 {
		t.Fatalf("json.Number: ok=%v n=%v", ok, n)
	}
	n, ok = IntFromAny("42")
	if !ok || n != 42 {
		t.Fatalf("string: ok=%v n=%v", ok, n)
	}
}

func TestIntDefault(t *testing.T) {
	if got := IntDefault(0, 3); got != 3 {
		t.Fatalf("zero -> def: got %d", got)
	}
	if got := IntDefault(5, 3); got != 5 {
		t.Fatalf("positive: got %d", got)
	}
}

func TestBoolFromAny(t *testing.T) {
	if !BoolFromAny("true") || !BoolFromAny("1") {
		t.Fatal("expected true for true/1 strings")
	}
	if BoolFromAny("false") {
		t.Fatal("expected false")
	}
}

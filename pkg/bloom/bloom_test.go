package bloom

import (
	"context"
	"testing"
)

func TestMemoryFilterAdd(t *testing.T) {
	filter := NewMemoryFilter(1000, 0.01)
	ctx := context.Background()

	err := filter.Add(ctx, "test")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	exists, err := filter.Contains(ctx, "test")
	if err != nil {
		t.Fatalf("Contains failed: %v", err)
	}
	if !exists {
		t.Error("Expected element to exist")
	}
}

func TestMemoryFilterContains(t *testing.T) {
	filter := NewMemoryFilter(1000, 0.01)
	ctx := context.Background()

	filter.Add(ctx, "apple")
	filter.Add(ctx, "banana")

	tests := []struct {
		name     string
		data     string
		expected bool
	}{
		{"apple exists", "apple", true},
		{"banana exists", "banana", true},
		{"orange not exists", "orange", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := filter.Contains(ctx, tt.data)
			if err != nil {
				t.Fatalf("Contains failed: %v", err)
			}
			if exists != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, exists)
			}
		})
	}
}

func TestMemoryFilterAddMulti(t *testing.T) {
	filter := NewMemoryFilter(1000, 0.01)
	ctx := context.Background()

	data := []string{"a", "b", "c", "d", "e"}
	err := filter.AddMulti(ctx, data)
	if err != nil {
		t.Fatalf("AddMulti failed: %v", err)
	}

	for _, item := range data {
		exists, err := filter.Contains(ctx, item)
		if err != nil {
			t.Fatalf("Contains failed: %v", err)
		}
		if !exists {
			t.Errorf("Expected %s to exist", item)
		}
	}
}

func TestMemoryFilterContainsMulti(t *testing.T) {
	filter := NewMemoryFilter(1000, 0.01)
	ctx := context.Background()

	filter.AddMulti(ctx, []string{"x", "y", "z"})

	results, err := filter.ContainsMulti(ctx, []string{"x", "y", "z", "w"})
	if err != nil {
		t.Fatalf("ContainsMulti failed: %v", err)
	}

	expected := []bool{true, true, true, false}
	for i, result := range results {
		if result != expected[i] {
			t.Errorf("Index %d: expected %v, got %v", i, expected[i], result)
		}
	}
}

func TestMemoryFilterClear(t *testing.T) {
	filter := NewMemoryFilter(1000, 0.01)
	ctx := context.Background()

	filter.Add(ctx, "test")
	exists, _ := filter.Contains(ctx, "test")
	if !exists {
		t.Error("Element should exist before clear")
	}

	filter.Clear(ctx)
	exists, _ = filter.Contains(ctx, "test")
	if exists {
		t.Error("Element should not exist after clear")
	}
}

func TestMemoryFilterStats(t *testing.T) {
	filter := NewMemoryFilter(1000, 0.01)

	stats := filter.Stats()
	if stats["type"] != "memory" {
		t.Errorf("Expected type 'memory', got %v", stats["type"])
	}

	if _, ok := stats["size_bits"]; !ok {
		t.Error("Missing size_bits in stats")
	}
	if _, ok := stats["hash_count"]; !ok {
		t.Error("Missing hash_count in stats")
	}
}

func TestMemoryFilterConcurrency(t *testing.T) {
	filter := NewMemoryFilter(10000, 0.01)
	ctx := context.Background()

	// 并发写入
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := "key_" + string(rune('0'+id)) + "_" + string(rune('0'+j%10))
				filter.Add(ctx, key)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证数据
	exists, _ := filter.Contains(ctx, "key_0_0")
	if !exists {
		t.Error("Expected element to exist after concurrent writes")
	}
}

func TestMemoryFilterEdgeCases(t *testing.T) {
	filter := NewMemoryFilter(1000, 0.01)
	ctx := context.Background()

	// 空字符串
	filter.Add(ctx, "")
	exists, _ := filter.Contains(ctx, "")
	if !exists {
		t.Error("Empty string should be added")
	}

	// 长字符串
	longStr := ""
	for i := 0; i < 1000; i++ {
		longStr += "a"
	}
	filter.Add(ctx, longStr)
	exists, _ = filter.Contains(ctx, longStr)
	if !exists {
		t.Error("Long string should be added")
	}

	// 特殊字符
	filter.Add(ctx, "!@#$%^&*()")
	exists, _ = filter.Contains(ctx, "!@#$%^&*()")
	if !exists {
		t.Error("Special characters should be added")
	}
}

func TestNewFilter(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "memory filter",
			config: Config{
				Type:              "memory",
				ExpectedElements:  1000,
				FalsePositiveRate: 0.01,
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			config: Config{
				Type: "invalid",
			},
			wantErr: true,
		},
		{
			name: "redis without address",
			config: Config{
				Type: "redis",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFilter(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewMemoryFilterWithDefaults(t *testing.T) {
	filter := NewMemoryFilterWithDefaults()
	if filter == nil {
		t.Error("NewMemoryFilterWithDefaults returned nil")
	}

	ctx := context.Background()
	err := filter.Add(ctx, "test")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
}

func BenchmarkMemoryFilterAdd(b *testing.B) {
	filter := NewMemoryFilter(100000, 0.01)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Add(ctx, "benchmark_test")
	}
}

func BenchmarkMemoryFilterContains(b *testing.B) {
	filter := NewMemoryFilter(100000, 0.01)
	ctx := context.Background()
	filter.Add(ctx, "benchmark_test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Contains(ctx, "benchmark_test")
	}
}

func BenchmarkMemoryFilterAddMulti(b *testing.B) {
	filter := NewMemoryFilter(100000, 0.01)
	ctx := context.Background()
	data := []string{"a", "b", "c", "d", "e"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.AddMulti(ctx, data)
	}
}

package utils

import (
	"testing"
	"time"
)

func TestNowUnix(t *testing.T) {
	result := NowUnix()
	if result <= 0 {
		t.Errorf("NowUnix() = %d; want > 0", result)
	}
}

func TestMillisFromTime(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want int64
	}{
		{
			name: "Unix epoch",
			t:    time.Unix(0, 0),
			want: 0,
		},
		{
			name: "Unix epoch + 1 second",
			t:    time.Unix(1, 0),
			want: 1000,
		},
		{
			name: "Unix epoch + 1 millisecond",
			t:    time.Unix(0, 1*int64(time.Millisecond)),
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MillisFromTime(tt.t); got != tt.want {
				t.Errorf("MillisFromTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeFromMillis(t *testing.T) {
	tests := []struct {
		name   string
		millis int64
		want   time.Time
	}{
		{
			name:   "Zero milliseconds",
			millis: 0,
			want:   time.Unix(0, 0),
		},
		{
			name:   "1000 milliseconds (1 second)",
			millis: 1000,
			want:   time.Unix(1, 0),
		},
		{
			name:   "1500 milliseconds (1.5 seconds)",
			millis: 1500,
			want:   time.Unix(1, 500*int64(time.Millisecond)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeFromMillis(tt.millis)
			if !got.Equal(tt.want) {
				t.Errorf("TimeFromMillis() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStartOfDay(t *testing.T) {
	// Create a specific time with date and time components
	testTime := time.Date(2023, 5, 15, 14, 30, 45, 123456789, time.UTC)
	expected := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)

	got := StartOfDay(testTime)
	if !got.Equal(expected) {
		t.Errorf("StartOfDay() = %v, want %v", got, expected)
	}
}

func TestEndOfDay(t *testing.T) {
	// Create a specific time with date and time components
	testTime := time.Date(2023, 5, 15, 14, 30, 45, 123456789, time.UTC)
	expected := time.Date(2023, 5, 15, 23, 59, 59, 999999999, time.UTC)

	got := EndOfDay(testTime)
	if !got.Equal(expected) {
		t.Errorf("EndOfDay() = %v, want %v", got, expected)
	}
}

func TestYesterday(t *testing.T) {
	now := time.Now()
	yesterday := Yesterday()

	// Check that yesterday is exactly one day before today
	expected := now.AddDate(0, 0, -1)

	// Compare dates (year, month, day) since time might differ slightly
	yesterdayDate := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	expectedDate := time.Date(expected.Year(), expected.Month(), expected.Day(), 0, 0, 0, 0, expected.Location())

	if !yesterdayDate.Equal(expectedDate) {
		t.Errorf("Yesterday() = %v, want %v", yesterdayDate, expectedDate)
	}
}

func TestCalculateAge(t *testing.T) {
	now := time.Now()
	currentYear := now.Year()
	tests := []struct {
		name     string
		birthday time.Time
		want     int
	}{
		{
			name:     "Birthday this year but not yet",
			birthday: time.Date(currentYear-25, now.Month()+1, now.Day(), 0, 0, 0, 0, now.Location()),
			want:     24,
		},
		{
			name:     "Birthday this year but already passed",
			birthday: time.Date(currentYear-25, 1, 1, 0, 0, 0, 0, now.Location()),
			want:     25,
		},
		{
			name:     "Birthday yesterday",
			birthday: time.Date(currentYear-30, now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location()),
			want:     30,
		},
		{
			name:     "Birthday tomorrow",
			birthday: time.Date(currentYear-30, now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()),
			want:     29,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateAge(tt.birthday)
			if got != tt.want {
				t.Errorf("CalculateAge() with birthday %v = %v, want %v",
					tt.birthday, got, tt.want)
			}
		})
	}
}

func TestAddDuration(t *testing.T) {
	now := time.Now().Unix()

	// Test adding 1 hour
	duration := time.Hour
	result := AddDuration(duration)

	// The result should be approximately now + 1 hour in seconds
	expected := now + int64(duration/time.Second)

	// Allow for a small difference due to time passing during test execution
	if result < expected-1 || result > expected+1 {
		t.Errorf("AddDuration() = %v, want approximately %v", result, expected)
	}
}

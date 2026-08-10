package utils

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
)

func TestCreatePasswordAndCompare(t *testing.T) {
	password := "SecretPass123!"

	hashedPassword, err := CreatePassword(password)
	if err != nil {
		t.Fatalf("expected no error creating password, got %v", err)
	}

	ok, err := ComparePasswords(password, hashedPassword)
	if !ok || err != nil {
		t.Fatalf("expected ComparePasswords to succeed, got ok=%v, err=%v", ok, err)
	}

	okFail, errFail := ComparePasswords("WrongPass", hashedPassword)
	if okFail || errFail == nil {
		t.Fatalf("expected ComparePasswords to fail for wrong password, got ok=%v, err=%v", okFail, errFail)
	}
}

func TestCreatePassword_TooLong(t *testing.T) {
	longPassword := strings.Repeat("a", 73)
	_, err := CreatePassword(longPassword)
	if err == nil {
		t.Fatal("expected error for password exceeding 72 bytes, got nil")
	}
}

func TestGetCadenceStartAndEndTime(t *testing.T) {
	// Fixed target date: Wednesday, August 12, 2026 15:30:00 UTC
	targetDate := time.Date(2026, time.August, 12, 15, 30, 0, 0, time.UTC)

	t.Run("Weekly Cadence (Sunday 00:00 to Saturday 23:59:59)", func(t *testing.T) {
		start, end, err := GetCadenceStartAndEndTime(core.WeeklyCadence, targetDate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedStart := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
		expectedEnd := time.Date(2026, time.August, 15, 23, 59, 59, 999999999, time.UTC)
		if !start.Equal(expectedStart) {
			t.Errorf("expected start %v, got %v", expectedStart, start)
		}
		if !end.Equal(expectedEnd) {
			t.Errorf("expected end %v, got %v", expectedEnd, end)
		}
	})

	t.Run("Monthly Cadence (1st 00:00 to last day 23:59:59)", func(t *testing.T) {
		start, end, err := GetCadenceStartAndEndTime(core.MonthlyCadence, targetDate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedStart := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
		expectedEnd := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
		if !start.Equal(expectedStart) {
			t.Errorf("expected start %v, got %v", expectedStart, start)
		}
		if !end.Equal(expectedEnd) {
			t.Errorf("expected end %v, got %v", expectedEnd, end)
		}
	})

	t.Run("Yearly Cadence (Jan 1 00:00 to Dec 31 23:59:59)", func(t *testing.T) {
		start, end, err := GetCadenceStartAndEndTime(core.YearlyCadence, targetDate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		expectedEnd := time.Date(2026, time.December, 31, 23, 59, 59, 999999999, time.UTC)
		if !start.Equal(expectedStart) {
			t.Errorf("expected start %v, got %v", expectedStart, start)
		}
		if !end.Equal(expectedEnd) {
			t.Errorf("expected end %v, got %v", expectedEnd, end)
		}
	})

	t.Run("Daily Cadence", func(t *testing.T) {
		start, end, err := GetCadenceStartAndEndTime(core.DailyCadence, targetDate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedStart := time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC)
		expectedEnd := time.Date(2026, time.August, 12, 23, 59, 59, 999999999, time.UTC)
		if !start.Equal(expectedStart) {
			t.Errorf("expected start %v, got %v", expectedStart, start)
		}
		if !end.Equal(expectedEnd) {
			t.Errorf("expected end %v, got %v", expectedEnd, end)
		}
	})

	t.Run("Invalid Cadence", func(t *testing.T) {
		_, _, err := GetCadenceStartAndEndTime(core.Cadence("invalid"), targetDate)
		if err == nil {
			t.Fatal("expected error for invalid cadence, got nil")
		}
	})
}

func TestGetCadenceStartAndEndTime_Randomized(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	cadences := []core.Cadence{
		core.DailyCadence,
		core.WeeklyCadence,
		core.MonthlyCadence,
		core.YearlyCadence,
	}

	for i := 0; i < 100; i++ {
		year := 2000 + r.Intn(50)
		month := time.Month(1 + r.Intn(12))
		day := 1 + r.Intn(28)
		hour := r.Intn(24)
		minute := r.Intn(60)
		sec := r.Intn(60)

		targetDate := time.Date(year, month, day, hour, minute, sec, 0, time.UTC)

		for _, cadence := range cadences {
			start, end, err := GetCadenceStartAndEndTime(cadence, targetDate)
			if err != nil {
				t.Fatalf("unexpected error for date %v cadence %s: %v", targetDate, cadence, err)
			}

			if targetDate.Before(start) || targetDate.After(end) {
				t.Errorf("targetDate %v not in range [%v, %v] for cadence %s", targetDate, start, end, cadence)
			}

			if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 || start.Nanosecond() != 0 {
				t.Errorf("start time %v not midnight for cadence %s", start, cadence)
			}

			if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 || end.Nanosecond() != 999999999 {
				t.Errorf("end time %v not end-of-day for cadence %s", end, cadence)
			}

			switch cadence {
			case core.DailyCadence:
				if start.Year() != year || start.Month() != month || start.Day() != day {
					t.Errorf("daily start %v mismatched date", start)
				}
			case core.WeeklyCadence:
				if start.Weekday() != time.Sunday {
					t.Errorf("weekly start weekday %v is not Sunday for targetDate %v", start.Weekday(), targetDate)
				}
				if end.Weekday() != time.Saturday {
					t.Errorf("weekly end weekday %v is not Saturday for targetDate %v", end.Weekday(), targetDate)
				}
			case core.MonthlyCadence:
				if start.Day() != 1 || start.Month() != month || start.Year() != year {
					t.Errorf("monthly start %v mismatch for targetDate %v", start, targetDate)
				}
			case core.YearlyCadence:
				if start.Month() != time.January || start.Day() != 1 || start.Year() != year {
					t.Errorf("yearly start %v mismatch for targetDate %v", start, targetDate)
				}
			}
		}
	}
}

func TestGetCadenceStartAndEndTime_EdgeCases(t *testing.T) {
	t.Run("Leap Year Feb 29 (2024)", func(t *testing.T) {
		target := time.Date(2024, time.February, 29, 12, 0, 0, 0, time.UTC)
		start, end, err := GetCadenceStartAndEndTime(core.MonthlyCadence, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedStart := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
		expectedEnd := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
		if !start.Equal(expectedStart) {
			t.Errorf("expected start %v, got %v", expectedStart, start)
		}
		if !end.Equal(expectedEnd) {
			t.Errorf("expected end %v, got %v", expectedEnd, end)
		}
		if end.Day() != 29 {
			t.Errorf("expected Feb leap year end day 29, got %d", end.Day())
		}
	})

	t.Run("Non-Leap Year Feb 28 (2025)", func(t *testing.T) {
		target := time.Date(2025, time.February, 15, 12, 0, 0, 0, time.UTC)
		_, end, err := GetCadenceStartAndEndTime(core.MonthlyCadence, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if end.Day() != 28 {
			t.Errorf("expected Feb non-leap year end day 28, got %d", end.Day())
		}
	})

	t.Run("30-Day Month April (2026)", func(t *testing.T) {
		target := time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC)
		_, end, err := GetCadenceStartAndEndTime(core.MonthlyCadence, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if end.Day() != 30 {
			t.Errorf("expected April end day 30, got %d", end.Day())
		}
	})

	t.Run("31-Day Month July (2026)", func(t *testing.T) {
		target := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
		_, end, err := GetCadenceStartAndEndTime(core.MonthlyCadence, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if end.Day() != 31 {
			t.Errorf("expected July end day 31, got %d", end.Day())
		}
	})

	t.Run("December to January Year Rollover", func(t *testing.T) {
		target := time.Date(2026, time.December, 31, 23, 0, 0, 0, time.UTC)
		start, end, err := GetCadenceStartAndEndTime(core.MonthlyCadence, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedStart := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
		expectedEnd := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
		if !start.Equal(expectedStart) {
			t.Errorf("expected start %v, got %v", expectedStart, start)
		}
		if !end.Equal(expectedEnd) {
			t.Errorf("expected end %v, got %v", expectedEnd, end)
		}
	})

	t.Run("Cross-Year Weekly Boundary (Dec 31, 2025 Wednesday)", func(t *testing.T) {
		target := time.Date(2025, time.December, 31, 12, 0, 0, 0, time.UTC) // Wednesday
		start, end, err := GetCadenceStartAndEndTime(core.WeeklyCadence, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedStart := time.Date(2025, time.December, 28, 0, 0, 0, 0, time.UTC)  // Sunday
		expectedEnd := time.Date(2026, time.January, 3, 23, 59, 59, 999999999, time.UTC) // Saturday
		if !start.Equal(expectedStart) {
			t.Errorf("expected start %v, got %v", expectedStart, start)
		}
		if !end.Equal(expectedEnd) {
			t.Errorf("expected end %v, got %v", expectedEnd, end)
		}
	})
}

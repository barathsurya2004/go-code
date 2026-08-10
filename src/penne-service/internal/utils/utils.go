package utils

import (
	"fmt"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"golang.org/x/crypto/bcrypt"
)

func ComparePasswords(password string, hashedPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return false, err
	}
	return true, nil
}

func CreatePassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func GetCadenceStartAndEndTime(cadence core.Cadence, targetDate time.Time) (time.Time, time.Time, error) {
	loc := targetDate.Location()
	year, month, day := targetDate.Date()

	switch cadence {
	case core.DailyCadence:
		start := time.Date(year, month, day, 0, 0, 0, 0, loc)
		end := time.Date(year, month, day, 23, 59, 59, 999999999, loc)
		return start, end, nil

	case core.WeeklyCadence:
		weekday := int(targetDate.Weekday()) // Sunday = 0, Monday = 1, ..., Saturday = 6
		sunday := targetDate.AddDate(0, 0, -weekday)
		sy, sm, sd := sunday.Date()
		start := time.Date(sy, sm, sd, 0, 0, 0, 0, loc)
		end := time.Date(sy, sm, sd+6, 23, 59, 59, 999999999, loc)
		return start, end, nil

	case core.MonthlyCadence:
		start := time.Date(year, month, 1, 0, 0, 0, 0, loc)
		end := time.Date(year, month+1, 1, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
		return start, end, nil

	case core.YearlyCadence:
		start := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
		end := time.Date(year, time.December, 31, 23, 59, 59, 999999999, loc)
		return start, end, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid cadence: %s", cadence)
	}
}

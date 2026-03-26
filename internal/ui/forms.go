package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseOptionalDate normalizes HTML date inputs and rejects invalid values.
func ParseOptionalDate(raw string) (*string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}

	if _, err := time.Parse(time.DateOnly, value); err != nil {
		return nil, fmt.Errorf("invalid date %q", value)
	}

	return &value, nil
}

// ParseOptionalInt normalizes optional integer fields from HTML forms.
func ParseOptionalInt(raw string) (*int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("invalid integer %q", value)
	}

	return &parsed, nil
}

// ParseOptionalFloat normalizes optional decimal fields from HTML forms.
func ParseOptionalFloat(raw string) (*float64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid decimal %q", value)
	}

	return &parsed, nil
}

// NormalizeCurrency keeps currency codes uppercase and applies a fallback.
func NormalizeCurrency(raw string, fallback string) string {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value != "" {
		return value
	}

	value = strings.ToUpper(strings.TrimSpace(fallback))
	if value != "" {
		return value
	}

	return "MXN"
}

package lead

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseMoney parses a CSV money cell into a float64.
// Empty string returns (nil, nil). Accepts values like "52000", "$45,000.00".
func ParseMoney(s string) (*float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	cleaned := strings.ReplaceAll(s, "$", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimSpace(cleaned)
	v, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid money value %q", s)
	}
	return &v, nil
}

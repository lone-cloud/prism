package util

import (
	"fmt"
	"strings"
	"time"
)

func FormatPhoneNumber(number string) string {
	if number == "" {
		return number
	}

	// US/Canada: +1 (234) 567-8901
	if strings.HasPrefix(number, "+1") && len(number) == 12 {
		return fmt.Sprintf("+1 (%s) %s-%s", number[2:5], number[5:8], number[8:])
	}

	// International: +XX XXXX XXXX
	if strings.HasPrefix(number, "+") && len(number) > 4 {
		digits := number[1:]
		var countryCode, rest string

		for i := 1; i <= 3 && i < len(digits); i++ {
			if digits[i] < '0' || digits[i] > '9' {
				break
			}
			countryCode = digits[:i+1]
			rest = digits[i+1:]
		}

		if len(rest) >= 3 {
			var parts []string
			for len(rest) > 0 {
				size := 3
				if len(rest) == 4 || len(rest) == 8 {
					size = 4
				}
				if size > len(rest) {
					size = len(rest)
				}
				parts = append(parts, rest[:size])
				rest = rest[size:]
			}
			return "+" + countryCode + " " + strings.Join(parts, " ")
		}
	}

	return number
}

func FormatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, " ")
}

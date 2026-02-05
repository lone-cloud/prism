package signal

import (
	"fmt"
	"strings"
)

func FormatPhoneNumber(number string) string {
	if number == "" {
		return number
	}

	if strings.HasPrefix(number, "+1") && len(number) == 12 {
		return fmt.Sprintf("+1 (%s) %s-%s", number[2:5], number[5:8], number[8:])
	}

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

package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	TypeInteger   = "integer"
	TypeReal      = "real"
	TypeChar      = "char"
	TypeString    = "string"
	TypeMoney     = "$"
	TypeMoneyInvl = "$Invl"

	// Максимум: 10,000,000,000,000.00 (у центах: 1,000,000,000,000,000)
	MaxMoneyCents int64 = 10_000_000_000_000_00
)

var SupportedTypes = map[string]bool{
	TypeInteger:   true,
	TypeReal:      true,
	TypeChar:      true,
	TypeString:    true,
	TypeMoney:     true,
	TypeMoneyInvl: true,
}

// ParseMoney парсить грошовий рядок (наприклад "$1,250.50", "100.5", "$500") у центи (int64)
func ParseMoney(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "$")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")

	if s == "" {
		return 0, errors.New("money value cannot be empty")
	}

	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, errors.New("invalid money format: multiple decimal points")
	}

	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer part of money: %w", err)
	}
	if dollars < 0 {
		return 0, errors.New("money cannot be negative")
	}

	var cents int64 = 0
	if len(parts) == 2 {
		centsStr := parts[1]
		if len(centsStr) == 1 {
			centsStr += "0"
		} else if len(centsStr) > 2 {
			return 0, errors.New("money cannot have more than 2 decimal digits")
		}
		c, err := strconv.ParseInt(centsStr, 10, 64)
		if err != nil {
			return 0, errors.New("invalid cents part of money")
		}
		cents = c
	}

	totalCents := dollars*100 + cents
	if totalCents > MaxMoneyCents {
		return 0, fmt.Errorf("money value exceeds limit of $10,000,000,000,000.00")
	}

	return totalCents, nil
}

// FormatMoney форматує центи у нормалізований рядок, наприклад "$1,250.50"
func FormatMoney(cents int64) string {
	dollars := cents / 100
	c := cents % 100
	dStr := strconv.FormatInt(dollars, 10)

	// Додаємо коми через кожні 3 розряди
	var result []rune
	for i, ch := range dStr {
		if (len(dStr)-i)%3 == 0 && i != 0 {
			result = append(result, ',')
		}
		result = append(result, ch)
	}
	return fmt.Sprintf("$%s.%02d", string(result), c)
}

// ParseMoneyInvl парсить інтервал грошей: "$10.00 - $50.00" або "[$10.00, $50.00]"
func ParseMoneyInvl(raw string) (int64, int64, error) {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, "[]()")

	var parts []string
	if strings.Contains(s, "-") {
		parts = strings.Split(s, "-")
	} else if strings.Contains(s, "..") {
		parts = strings.Split(s, "..")
	} else if strings.Contains(s, ",") {
		parts = strings.Split(s, ",")
	} else {
		return 0, 0, errors.New("invalid interval separator, use '-' or '..'")
	}

	if len(parts) != 2 {
		return 0, 0, errors.New("interval must contain exactly two money values: [from, to]")
	}

	fromCents, err := ParseMoney(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("interval 'from' error: %w", err)
	}

	toCents, err := ParseMoney(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("interval 'to' error: %w", err)
	}

	if fromCents > toCents {
		return 0, 0, errors.New("invalid interval: 'from' cannot be greater than 'to'")
	}

	return fromCents, toCents, nil
}

// ValidateAndNormalizeValue валідує вхідний рядок відповідно до типу та повертає його нормалізований вигляд
func ValidateAndNormalizeValue(colType string, rawValue string) (string, error) {
	val := strings.TrimSpace(rawValue)

	switch colType {
	case TypeInteger:
		_, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return "", fmt.Errorf("value '%s' is not a valid integer", val)
		}
		return val, nil

	case TypeReal:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return "", fmt.Errorf("value '%s' is not a valid real number", val)
		}
		return fmt.Sprintf("%g", f), nil

	case TypeChar:
		runes := []rune(val)
		if len(runes) != 1 {
			return "", fmt.Errorf("char type must be strictly 1 character, got %d", len(runes))
		}
		return val, nil

	case TypeString:
		return val, nil

	case TypeMoney:
		cents, err := ParseMoney(val)
		if err != nil {
			return "", err
		}
		return FormatMoney(cents), nil

	case TypeMoneyInvl:
		from, to, err := ParseMoneyInvl(val)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("[%s - %s]", FormatMoney(from), FormatMoney(to)), nil

	default:
		return "", fmt.Errorf("unsupported column type: '%s'", colType)
	}
}

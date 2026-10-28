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
	TypeMoneyInvl = "$Invl" // Гроші з контролем діапазону

	MaxMoneyCents int64 = 10_000_000_000_000_00 // 10 трлн
)

var SupportedTypes = map[string]bool{
	TypeInteger:   true,
	TypeReal:      true,
	TypeChar:      true,
	TypeString:    true,
	TypeMoney:     true,
	TypeMoneyInvl: true,
}

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
		return 0, errors.New("multiple decimal points")
	}

	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer part: %w", err)
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
			return 0, errors.New("maximum 2 decimal places allowed")
		}
		c, err := strconv.ParseInt(centsStr, 10, 64)
		if err != nil {
			return 0, errors.New("invalid cents part")
		}
		cents = c
	}

	totalCents := dollars*100 + cents
	if totalCents > MaxMoneyCents {
		return 0, fmt.Errorf("exceeds limit of $10,000,000,000,000.00")
	}

	return totalCents, nil
}

func FormatMoney(cents int64) string {
	dollars := cents / 100
	c := cents % 100
	dStr := strconv.FormatInt(dollars, 10)

	var result []rune
	for i, ch := range dStr {
		if (len(dStr)-i)%3 == 0 && i != 0 {
			result = append(result, ',')
		}
		result = append(result, ch)
	}
	return fmt.Sprintf("$%s.%02d", string(result), c)
}

// ValidateAndNormalizeValue валідує вхідне значення з урахуванням конфігурації колонки
func ValidateAndNormalizeValue(col Column, rawValue string) (string, error) {
	val := strings.TrimSpace(rawValue)

	switch col.Type {
	case TypeInteger:
		_, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return "", fmt.Errorf("value '%s' is not an integer", val)
		}
		return val, nil

	case TypeReal:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return "", fmt.Errorf("value '%s' is not a real number", val)
		}
		return fmt.Sprintf("%g", f), nil

	case TypeChar:
		runes := []rune(val)
		if len(runes) != 1 {
			return "", fmt.Errorf("char must be exactly 1 character, got %d", len(runes))
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
		cents, err := ParseMoney(val)
		if err != nil {
			return "", fmt.Errorf("invalid money: %w", err)
		}

		// Перевірка діапазону значень, налаштованого для колонки
		if col.RangeMin != "" {
			minCents, err := ParseMoney(col.RangeMin)
			if err == nil && cents < minCents {
				return "", fmt.Errorf("value %s is below allowed minimum %s", FormatMoney(cents), FormatMoney(minCents))
			}
		}
		if col.RangeMax != "" {
			maxCents, err := ParseMoney(col.RangeMax)
			if err == nil && cents > maxCents {
				return "", fmt.Errorf("value %s exceeds allowed maximum %s", FormatMoney(cents), FormatMoney(maxCents))
			}
		}

		return FormatMoney(cents), nil

	default:
		return "", fmt.Errorf("unsupported type: %s", col.Type)
	}
}
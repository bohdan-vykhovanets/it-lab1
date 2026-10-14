package domain

import (
	"testing"
)

// Тест 1: Валідація грошового типу $
func TestMoneyValidation(t *testing.T) {
	// Валідні випадки
	validCases := []struct {
		input    string
		expected string
	}{
		{"$100.50", "$100.50"},
		{"5000", "$5,000.00"},
		{"$1,250,000.99", "$1,250,000.99"},
		{"0", "$0.00"},
		{"10000000000000.00", "$10,000,000,000,000.00"},
	}

	for _, tc := range validCases {
		res, err := ValidateAndNormalizeValue(TypeMoney, tc.input)
		if err != nil {
			t.Errorf("Valid money '%s' gave unexpected error: %v", tc.input, err)
		}
		if res != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, res)
		}
	}

	// Невалідні випадки
	invalidCases := []string{
		"-50.00",              // Від'ємні
		"100.999",             // Більше 2 знаків після коми
		"10000000000000.01",   // Перевищення ліміту 10 трлн
		"abc",                 // Не число
	}

	for _, input := range invalidCases {
		_, err := ValidateAndNormalizeValue(TypeMoney, input)
		if err == nil {
			t.Errorf("Invalid money '%s' was accepted, expected error", input)
		}
	}
}

// Тест 2: Валідація інтервалу $Invl
func TestMoneyIntervalValidation(t *testing.T) {
	validIntervals := []string{
		"$10.00 - $50.00",
		"[$100, $200]",
		"1,000.00 - 5,000.00",
		"$10.00 - $10.00", // min == max дозволено
	}

	for _, inv := range validIntervals {
		_, err := ValidateAndNormalizeValue(TypeMoneyInvl, inv)
		if err != nil {
			t.Errorf("Valid interval '%s' failed: %v", inv, err)
		}
	}

	invalidIntervals := []string{
		"$50.00 - $10.00", // From > To
		"$100",            // Не інтервал
		"$10 - invalid",   // Помилка в To
	}

	for _, inv := range invalidIntervals {
		_, err := ValidateAndNormalizeValue(TypeMoneyInvl, inv)
		if err == nil {
			t.Errorf("Invalid interval '%s' was accepted, expected error", inv)
		}
	}
}

// Тест 3: Тестування індивідуальної операції варіанту (Проєкція таблиці)
func TestTableProjection(t *testing.T) {
	cols := []Column{
		{Name: "ID", Type: TypeInteger},
		{Name: "Name", Type: TypeString},
		{Name: "Balance", Type: TypeMoney},
		{Name: "CreditLimit", Type: TypeMoneyInvl},
	}

	table, err := NewTable("Accounts", cols)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_ = table.AddRow([]string{"1", "Alice", "$500.00", "$100 - $1,000"})
	_ = table.AddRow([]string{"2", "Bob", "$1,200.50", "$500 - $2,000"})

	// Виконуємо проєкцію на дві колонки: Name та Balance
	projected, err := table.Project("ProjectedAccounts", []string{"Name", "Balance"})
	if err != nil {
		t.Fatalf("Projection failed: %v", err)
	}

	// Перевірка кількості колонок
	if len(projected.Columns) != 2 {
		t.Fatalf("Expected 2 columns in projection, got %d", len(projected.Columns))
	}
	if projected.Columns[0].Name != "Name" || projected.Columns[1].Name != "Balance" {
		t.Errorf("Column names in projection mismatch")
	}

	// Перевірка рядків
	if len(projected.Rows) != 2 {
		t.Fatalf("Expected 2 rows in projection, got %d", len(projected.Rows))
	}
	if projected.Rows[0].Values[0] != "Alice" || projected.Rows[0].Values[1] != "$500.00" {
		t.Errorf("Row 0 projection data mismatch: %v", projected.Rows[0].Values)
	}
	if projected.Rows[1].Values[0] != "Bob" || projected.Rows[1].Values[1] != "$1,200.50" {
		t.Errorf("Row 1 projection data mismatch: %v", projected.Rows[1].Values)
	}
}

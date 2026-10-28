package domain

import (
	"os"
	"testing"
)

// Тест 1: Валідація грошового типу $
func TestMoneyValidation(t *testing.T) {
	col := Column{Name: "Amount", Type: TypeMoney}

	validCases := []struct {
		input    string
		expected string
	}{
		{"$100.50", "$100.50"},
		{"5000", "$5,000.00"},
		{"$1,250,000.99", "$1,250,000.99"},
		{"0", "$0.00"},
	}

	for _, tc := range validCases {
		res, err := ValidateAndNormalizeValue(col, tc.input)
		if err != nil {
			t.Errorf("Valid money '%s' failed: %v", tc.input, err)
		}
		if res != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, res)
		}
	}

	invalidCases := []string{"-50.00", "100.999", "abc", "10000000000000.01"}
	for _, input := range invalidCases {
		_, err := ValidateAndNormalizeValue(col, input)
		if err == nil {
			t.Errorf("Invalid money '%s' was accepted, expected error", input)
		}
	}
}

// Тест 2: Валідація допустимого діапазону для $Invl
func TestMoneyRangeValidation(t *testing.T) {
	// Колонка з діапазоном від $100 до $1,000
	col := Column{
		Name:     "Salary",
		Type:     TypeMoneyInvl,
		RangeMin: "$100.00",
		RangeMax: "$1,000.00",
	}

	// Значення в межах діапазону
	validValues := []string{"$100.00", "$500.00", "$1,000.00"}
	for _, val := range validValues {
		_, err := ValidateAndNormalizeValue(col, val)
		if err != nil {
			t.Errorf("Value '%s' should be valid in range [%s - %s]: %v", val, col.RangeMin, col.RangeMax, err)
		}
	}

	// Значення поза межами діапазону
	invalidValues := []string{"$50.00", "$1,500.00", "0"}
	for _, val := range invalidValues {
		_, err := ValidateAndNormalizeValue(col, val)
		if err == nil {
			t.Errorf("Value '%s' is out of range [%s - %s], but was accepted", val, col.RangeMin, col.RangeMax)
		}
	}
}

// Тест 3: Операція проєкції
func TestTableProjection(t *testing.T) {
	cols := []Column{
		{Name: "ID", Type: TypeInteger},
		{Name: "Title", Type: TypeString},
		{Name: "Price", Type: TypeMoney},
		{Name: "Budget", Type: TypeMoneyInvl, RangeMin: "$10.00", RangeMax: "$1,000.00"},
	}

	table, err := NewTable("Products", cols)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_ = table.AddRow([]string{"1", "Laptop", "$1,200.00", "$500.00"})
	_ = table.AddRow([]string{"2", "Mouse", "$25.00", "$20.00"})

	proj, err := table.Project("Projected", []string{"Title", "Budget"})
	if err != nil {
		t.Fatalf("Projection failed: %v", err)
	}

	if len(proj.Columns) != 2 || len(proj.Rows) != 2 {
		t.Fatalf("Projection dimensions invalid")
	}
	if proj.Rows[0].Values[1] != "$500.00" {
		t.Errorf("Projected value mismatch")
	}
}

// Тест 4: Валідація при імпорті даних із файлу
func TestImportValidation(t *testing.T) {
	db := NewDatabase("TestDB")
	invalidJson := `{
		"name": "CorruptedDB",
		"tables": {
			"T1": {
				"name": "T1",
				"columns": [{"name": "Col1", "type": "$Invl", "range_min": "$100", "range_max": "$500"}],
				"rows": [{"values": ["$999.00"]}]
			}
		}
	}`

	tmpFile := "corrupted_test.json"
	_ = os.WriteFile(tmpFile, []byte(invalidJson), 0644)
	defer os.Remove(tmpFile)

	err := db.LoadFromFile(tmpFile)
	if err == nil {
		t.Errorf("Expected import to fail due to value $999.00 exceeding range, but it passed")
	}
}
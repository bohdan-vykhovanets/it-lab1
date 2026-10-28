package domain

import (
	"errors"
	"fmt"
)

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	RangeMin string `json:"range_min,omitempty"` // Наприклад "$100.00"
	RangeMax string `json:"range_max,omitempty"` // Наприклад "$5,000.00"
}

type Row struct {
	Values []string `json:"values"`
}

type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
	Rows    []Row    `json:"rows"`
}

func NewTable(name string, cols []Column) (*Table, error) {
	if len(name) == 0 {
		return nil, errors.New("table name cannot be empty")
	}
	if len(cols) == 0 {
		return nil, errors.New("table must have at least one column")
	}

	seen := make(map[string]bool)
	for i, col := range cols {
		if col.Name == "" {
			return nil, errors.New("column name cannot be empty")
		}
		if seen[col.Name] {
			return nil, fmt.Errorf("duplicate column name: %s", col.Name)
		}
		seen[col.Name] = true

		if !SupportedTypes[col.Type] {
			return nil, fmt.Errorf("unsupported type '%s' for column '%s'", col.Type, col.Name)
		}

		// Якщо тип $Invl, валідуємо сам діапазон колонки
		if col.Type == TypeMoneyInvl {
			if col.RangeMin == "" {
				col.RangeMin = "$0.00"
			}
			if col.RangeMax == "" {
				col.RangeMax = "$10,000,000,000,000.00"
			}
			minC, errMin := ParseMoney(col.RangeMin)
			maxC, errMax := ParseMoney(col.RangeMax)
			if errMin != nil || errMax != nil {
				return nil, fmt.Errorf("invalid range limits in column '%s'", col.Name)
			}
			if minC > maxC {
				return nil, fmt.Errorf("column '%s': RangeMin cannot be greater than RangeMax", col.Name)
			}
			cols[i].RangeMin = FormatMoney(minC)
			cols[i].RangeMax = FormatMoney(maxC)
		}
	}

	return &Table{
		Name:    name,
		Columns: cols,
		Rows:    make([]Row, 0),
	}, nil
}

func (t *Table) AddRow(rawValues []string) error {
	if len(rawValues) != len(t.Columns) {
		return fmt.Errorf("row values count (%d) mismatch column count (%d)", len(rawValues), len(t.Columns))
	}

	normalized := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		normVal, err := ValidateAndNormalizeValue(col, rawValues[i])
		if err != nil {
			return fmt.Errorf("column '%s' validation failed: %w", col.Name, err)
		}
		normalized[i] = normVal
	}

	t.Rows = append(t.Rows, Row{Values: normalized})
	return nil
}

func (t *Table) UpdateRow(idx int, rawValues []string) error {
	if idx < 0 || idx >= len(t.Rows) {
		return errors.New("row index out of range")
	}
	if len(rawValues) != len(t.Columns) {
		return fmt.Errorf("values count (%d) mismatch column count (%d)", len(rawValues), len(t.Columns))
	}

	normalized := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		normVal, err := ValidateAndNormalizeValue(col, rawValues[i])
		if err != nil {
			return fmt.Errorf("column '%s' error: %w", col.Name, err)
		}
		normalized[i] = normVal
	}

	t.Rows[idx] = Row{Values: normalized}
	return nil
}

func (t *Table) DeleteRow(idx int) error {
	if idx < 0 || idx >= len(t.Rows) {
		return errors.New("row index out of range")
	}
	t.Rows = append(t.Rows[:idx], t.Rows[idx+1:]...)
	return nil
}

func (t *Table) Project(newTableName string, columnNames []string) (*Table, error) {
	if len(columnNames) == 0 {
		return nil, errors.New("projection requires at least one column")
	}

	colMap := make(map[string]int)
	for i, c := range t.Columns {
		colMap[c.Name] = i
	}

	var newCols []Column
	var targetIndices []int

	for _, name := range columnNames {
		idx, exists := colMap[name]
		if !exists {
			return nil, fmt.Errorf("column '%s' does not exist in table '%s'", name, t.Name)
		}
		newCols = append(newCols, t.Columns[idx])
		targetIndices = append(targetIndices, idx)
	}

	projectedTable, err := NewTable(newTableName, newCols)
	if err != nil {
		return nil, err
	}

	for _, row := range t.Rows {
		newRowValues := make([]string, len(targetIndices))
		for i, originalIdx := range targetIndices {
			newRowValues[i] = row.Values[originalIdx]
		}
		projectedTable.Rows = append(projectedTable.Rows, Row{Values: newRowValues})
	}

	return projectedTable, nil
}
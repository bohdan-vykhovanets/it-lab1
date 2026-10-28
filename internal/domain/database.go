package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Database struct {
	Name   string            `json:"name"`
	Tables map[string]*Table `json:"tables"`
	mu     sync.RWMutex      `json:"-"`
}

func NewDatabase(name string) *Database {
	return &Database{
		Name:   name,
		Tables: make(map[string]*Table),
	}
}

func (db *Database) AddTable(t *Table) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.Tables[t.Name]; exists {
		return fmt.Errorf("table '%s' already exists", t.Name)
	}
	db.Tables[t.Name] = t
	return nil
}

func (db *Database) GetTable(name string) (*Table, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	t, exists := db.Tables[name]
	if !exists {
		return nil, fmt.Errorf("table '%s' not found", name)
	}
	return t, nil
}

func (db *Database) DropTable(name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.Tables[name]; !exists {
		return fmt.Errorf("table '%s' not found", name)
	}
	delete(db.Tables, name)
	return nil
}

func (db *Database) SaveToFile(filepath string) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize database: %w", err)
	}

	return os.WriteFile(filepath, data, 0644)
}

// LoadFromFile зчитує файл та проводить СУВОРУ ВАЛІДАЦІЮ ВСІХ ІМПОРТОВАНИХ ДАНИХ
func (db *Database) LoadFromFile(filepath string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read database file: %w", err)
	}

	var loaded struct {
		Name   string            `json:"name"`
		Tables map[string]*Table `json:"tables"`
	}
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("failed to parse database json: %w", err)
	}

	if loaded.Name == "" {
		return fmt.Errorf("validation error: imported database name cannot be empty")
	}

	validatedTables := make(map[string]*Table)

	// Валідація кожної таблиці та кожного поля у файлі
	for tableName, tbl := range loaded.Tables {
		if tbl == nil {
			return fmt.Errorf("table '%s' is null", tableName)
		}

		// 1. Валідація схеми колонок (включаючи діапазони)
		validTbl, err := NewTable(tbl.Name, tbl.Columns)
		if err != nil {
			return fmt.Errorf("table '%s' schema validation error: %w", tableName, err)
		}

		// 2. Валідація кожного рядка та його клітинок
		for rowIdx, row := range tbl.Rows {
			if len(row.Values) != len(validTbl.Columns) {
				return fmt.Errorf("table '%s', row #%d: expected %d values, got %d",
					tableName, rowIdx+1, len(validTbl.Columns), len(row.Values))
			}

			normRow := make([]string, len(validTbl.Columns))
			for colIdx, col := range validTbl.Columns {
				normVal, err := ValidateAndNormalizeValue(col, row.Values[colIdx])
				if err != nil {
					return fmt.Errorf("import validation failed in table '%s', row #%d, column '%s': %w",
						tableName, rowIdx+1, col.Name, err)
				}
				normRow[colIdx] = normVal
			}
			validTbl.Rows = append(validTbl.Rows, Row{Values: normRow})
		}

		validatedTables[tableName] = validTbl
	}

	// Застосовуємо імпорт лише якщо ВСІ перевірки успішно пройдено
	db.Name = loaded.Name
	db.Tables = validatedTables
	return nil
}
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

func (db *Database) LoadFromFile(filepath string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read database file: %w", err)
	}

	var loaded Database
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("failed to parse database json: %w", err)
	}

	db.Name = loaded.Name
	db.Tables = loaded.Tables
	return nil
}

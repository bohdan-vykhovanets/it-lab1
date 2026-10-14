package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"db-server/internal/domain"
)

type ServerAPI struct {
	DB *domain.Database
}

func NewServerAPI(db *domain.Database) *ServerAPI {
	return &ServerAPI{DB: db}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// EnableCORS додає CORS заголовки для зв'язку з WPF та веб-клієнтами
func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (api *ServerAPI) GetDatabase(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.DB)
}

func (api *ServerAPI) CreateDatabase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "Invalid database name")
		return
	}
	api.DB.Name = req.Name
	api.DB.Tables = make(map[string]*domain.Table)
	writeJSON(w, http.StatusOK, map[string]string{"status": "database created", "name": req.Name})
}

func (api *ServerAPI) SaveDatabase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Filename == "" {
		req.Filename = "database.json"
	}
	if err := api.DB.SaveToFile(req.Filename); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved", "file": req.Filename})
}

func (api *ServerAPI) LoadDatabase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Filename == "" {
		req.Filename = "database.json"
	}
	if err := api.DB.LoadFromFile(req.Filename); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "loaded", "name": api.DB.Name})
}

func (api *ServerAPI) CreateTable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string          `json:"name"`
		Columns []domain.Column `json:"columns"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	tbl, err := domain.NewTable(req.Name, req.Columns)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := api.DB.AddTable(tbl); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tbl)
}

func (api *ServerAPI) GetTable(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")
	tbl, err := api.DB.GetTable(tableName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tbl)
}

func (api *ServerAPI) DropTable(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")
	if err := api.DB.DropTable(tableName); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "table dropped"})
}

func (api *ServerAPI) AddRow(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")
	tbl, err := api.DB.GetTable(tableName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req struct {
		Values []string `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid row values payload")
		return
	}

	if err := tbl.AddRow(req.Values); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Повертаємо щойно доданий валідований рядок
	writeJSON(w, http.StatusCreated, tbl.Rows[len(tbl.Rows)-1])
}

func (api *ServerAPI) UpdateRow(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")
	tbl, err := api.DB.GetTable(tableName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	rowIdxStr := r.PathValue("idx")
	idx, err := strconv.Atoi(rowIdxStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid row index")
		return
	}

	var req struct {
		Values []string `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	if err := tbl.UpdateRow(idx, req.Values); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Повертаємо оновлений і нормалізований рядок
	writeJSON(w, http.StatusOK, tbl.Rows[idx])
}

func (api *ServerAPI) DeleteRow(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")
	tbl, err := api.DB.GetTable(tableName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	rowIdxStr := r.PathValue("idx")
	idx, err := strconv.Atoi(rowIdxStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid row index")
		return
	}

	if err := tbl.DeleteRow(idx); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "row deleted"})
}

func (api *ServerAPI) ProjectTable(w http.ResponseWriter, r *http.Request) {
	tableName := r.PathValue("name")
	tbl, err := api.DB.GetTable(tableName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req struct {
		NewTableName string   `json:"new_table_name"`
		Columns      []string `json:"columns"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid projection payload")
		return
	}

	projectedTbl, err := tbl.Project(req.NewTableName, req.Columns)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := api.DB.AddTable(projectedTbl); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, projectedTbl)
}

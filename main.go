package main

import (
	"log"
	"net/http"
	"os"

	"db-server/internal/api"
	"db-server/internal/domain"
)

func main() {
	db := domain.NewDatabase("DefaultDB")
	serverAPI := api.NewServerAPI(db)

	mux := http.NewServeMux()

	// REST API ендпоінти
	mux.HandleFunc("GET /api/database", serverAPI.GetDatabase)
	mux.HandleFunc("POST /api/database/new", serverAPI.CreateDatabase)
	mux.HandleFunc("POST /api/database/save", serverAPI.SaveDatabase)
	mux.HandleFunc("POST /api/database/load", serverAPI.LoadDatabase)

	mux.HandleFunc("POST /api/tables", serverAPI.CreateTable)
	mux.HandleFunc("GET /api/tables/{name}", serverAPI.GetTable)
	mux.HandleFunc("DELETE /api/tables/{name}", serverAPI.DropTable)

	mux.HandleFunc("POST /api/tables/{name}/rows", serverAPI.AddRow)
	mux.HandleFunc("PUT /api/tables/{name}/rows/{idx}", serverAPI.UpdateRow)
	mux.HandleFunc("DELETE /api/tables/{name}/rows/{idx}", serverAPI.DeleteRow)

	// Індивідуальна операція (Проєкція)
	mux.HandleFunc("POST /api/tables/{name}/project", serverAPI.ProjectTable)

	// Роздача статичних файлів веб-інтерфейсу з папки ./web
	fileServer := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fileServer)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler := api.EnableCORS(mux)

	log.Printf("Starting DBMS server on port %s...", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}

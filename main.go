package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading properties")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/healthz", readinessHandler)
	mux.HandleFunc("GET /v1/err", errorHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	fmt.Println("START")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Problem: %v", err)
	}
	fmt.Println("STOP")
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	type ReadinessResponse struct {
		Status string `json:"status"`
	}

	resp := ReadinessResponse{
		Status: "ok",
	}

	respondWithJSON(w, 200, resp)
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	type ErrorResponse struct {
		Error string `json:"error"`
	}

	resp := ErrorResponse{
		Error: "something went wrong",
	}

	respondWithJSON(w, 500, resp)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	payload := map[string]string{"error": msg}
	respondWithJSON(w, code, payload)
}

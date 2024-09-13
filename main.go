package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/halfdan87/boot-go-blog-aggregator/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	DB *database.Queries
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading properties")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbUrl := os.Getenv("DB_CONNECTION_STRING")

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	dbQueries := database.New(db)

	apiConfig := apiConfig{
		DB: dbQueries,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/healthz", readinessHandler)
	mux.HandleFunc("GET /v1/err", errorHandler)
	mux.HandleFunc("POST /v1/users", postUsersHandler(apiConfig))
	mux.HandleFunc("GET /v1/users", getUsersHandler(apiConfig))

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

func postUsersHandler(apiConfig apiConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		type UsersRequest struct {
			Name string `json:"name"`
		}

		var req UsersRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respondWithError(w, 400, "Error decoding request")
			return
		}

		type User struct {
			ID        int       `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Name      string    `json:"name"`
		}

		context := context.Background()
		userParams := database.InsertUserParams{
			ID:        uuid.New(),
			CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			Name:      req.Name,
		}

		user, err := apiConfig.DB.InsertUser(context, userParams)
		if err != nil {
			respondWithError(w, 500, "Error getting users")
			return
		}

		respondWithJSON(w, 200, user)
	}
}

func getUsersHandler(apiConfig apiConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		auth := r.Header.Get("Authorization")

		if auth == "" {
			respondWithError(w, 401, "Unauthorized")
			return
		}

		apiKey, err := getApiKeyFromAuth(auth)
		if err != nil {
			respondWithError(w, 401, "Unauthorized")
			return
		}

		type User struct {
			ID        int       `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Name      string    `json:"name"`
		}

		context := context.Background()

		user, err := apiConfig.DB.GetUserByApiKey(context, apiKey)
		if err != nil {
			respondWithError(w, 500, "Error getting user")
			return
		}

		respondWithJSON(w, 200, user)
	}
}

func getApiKeyFromAuth(auth string) (string, error) {
	token := strings.Split(auth, " ")
	if len(token) != 2 {
		return "", errors.New("Invalid token")
	}

	return token[1], nil
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

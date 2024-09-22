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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/halfdan87/boot-go-blog-aggregator/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	DB *database.Queries
}

type authedHandler func(http.ResponseWriter, *http.Request, database.User)

func (cfg *apiConfig) authedHandler(handler authedHandler) func(http.ResponseWriter, *http.Request) {
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

		user, err := cfg.DB.GetUserByApiKey(context.Background(), apiKey)
		if err != nil {
			respondWithError(w, 500, "Error getting user")
			return
		}

		handler(w, r, user)
	}
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

	router := chi.NewRouter()
	v1Router := chi.NewRouter()

	v1Router.Get("/healthz", readinessHandler)
	v1Router.Get("/err", errorHandler)
	v1Router.Post("/users", postUsersHandler(apiConfig))
	v1Router.Get("/users", apiConfig.authedHandler(getUsersHandler(apiConfig)))
	v1Router.Post("/feeds", apiConfig.authedHandler(postFeedsHandler(apiConfig)))
	v1Router.Get("/feeds", getFeedsHandler(apiConfig))

	v1Router.Post("/feed_follows", apiConfig.authedHandler(postFeedFollowHandler(apiConfig)))
	v1Router.Delete("/feed_follows/{feed_id}", apiConfig.authedHandler(deleteFeedFollowHandler(apiConfig)))
	v1Router.Get("/feed_follows", apiConfig.authedHandler(getUserFeedFollowsHandler(apiConfig)))

	router.Mount("/v1", v1Router)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
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

func getUsersHandler(apiConfig apiConfig) func(w http.ResponseWriter, r *http.Request, user database.User) {
	return func(w http.ResponseWriter, r *http.Request, user database.User) {
		respondWithJSON(w, 200, user)
	}
}

func postFeedsHandler(apiConfig apiConfig) func(w http.ResponseWriter, r *http.Request, user database.User) {
	return func(w http.ResponseWriter, r *http.Request, user database.User) {
		type FeedRequest struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}

		var req FeedRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respondWithError(w, 400, "Error decoding request")
			return
		}

		context := context.Background()
		feedParams := database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			Name:      req.Name,
			Url:       req.URL,
			UserID:    user.ID,
		}

		feed, err := apiConfig.DB.CreateFeed(context, feedParams)
		if err != nil {
			log.Printf("Error creating feed: %v", err)
			respondWithError(w, 500, "Error getting feeds")
			return
		}

		feedFollowParams := database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			UserID:    user.ID,
			FeedID:    feed.ID,
		}
		_, err = apiConfig.DB.CreateFeedFollow(context, feedFollowParams)
		if err != nil {
			log.Printf("Error creating feed follow: %v", err)
			respondWithError(w, 500, "Error getting feeds")
			return
		}

		respondWithJSON(w, 200, feed)
	}
}

func getFeedsHandler(apiConfig apiConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		context := context.Background()
		feeds, err := apiConfig.DB.GetFeeds(context)
		if err != nil {
			log.Printf("Error getting feeds: %v", err)
			respondWithError(w, 500, "Error getting feeds")
			return
		}

		respondWithJSON(w, 200, feeds)
	}
}

func postFeedFollowHandler(apiConfig apiConfig) func(w http.ResponseWriter, r *http.Request, user database.User) {
	return func(w http.ResponseWriter, r *http.Request, user database.User) {
		type FeedFollowRequest struct {
			FeedID uuid.UUID `json:"feed_id"`
		}

		var req FeedFollowRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respondWithError(w, 400, "Error decoding request")
			return
		}

		context := context.Background()
		feedFollowParams := database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
			UserID:    user.ID,
			FeedID:    req.FeedID,
		}

		feedFollow, err := apiConfig.DB.CreateFeedFollow(context, feedFollowParams)
		if err != nil {
			log.Printf("Error creating feed follow: %v", err)
			respondWithError(w, 500, "Error getting feeds")
			return
		}

		respondWithJSON(w, 200, feedFollow)
	}
}

func deleteFeedFollowHandler(apiConfig apiConfig) func(w http.ResponseWriter, r *http.Request, user database.User) {
	return func(w http.ResponseWriter, r *http.Request, user database.User) {
		vars := chi.URLParam(r, "feed_id")
		feedID, err := uuid.Parse(vars)
		if err != nil {
			respondWithError(w, 400, "Error decoding request")
			return
		}

		context := context.Background()
		err = apiConfig.DB.DeleteFeedFollow(context, feedID)
		if err != nil {
			log.Printf("Error deleting feed follow: %v", err)
			respondWithError(w, 500, "Error deleting feed follow")
			return
		}

		respondWithJSON(w, 200, nil)
	}
}

func getUserFeedFollowsHandler(apiConfig apiConfig) func(w http.ResponseWriter, r *http.Request, user database.User) {
	return func(w http.ResponseWriter, r *http.Request, user database.User) {
		context := context.Background()
		feedFollows, err := apiConfig.DB.GetUserFeedFollows(context, user.ID)
		if err != nil {
			log.Printf("Error getting feed follows: %v", err)
			respondWithError(w, 500, "Error getting feed follows")
			return
		}

		respondWithJSON(w, 200, feedFollows)
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
	if response != nil {
		w.Write(response)
	}
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	payload := map[string]string{"error": msg}
	respondWithJSON(w, code, payload)
}

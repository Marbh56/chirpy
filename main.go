package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type apiConfig struct {
	fileserverHits int
}

func (cfg *apiConfig) middlewareMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) adminMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	visitCount := cfg.fileserverHits
	htmlResponse := fmt.Sprintf(`
	<html>

	<body>
    	<h1>Welcome, Chirpy Admin</h1>
    	<p>Chirpy has been visited %d times!</p>
	</body>

	</html>
	`, visitCount)

	w.Write([]byte(htmlResponse))

}

func (cfg *apiConfig) validateChrip(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": "Something went wrong"})
		return
	}
	if len(params.Body) <= 140 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		// json.NewEncoder(w).Encode(map[string]bool{"valid": true})
		badWords := []string{"kerfuffle", "sharbert", "fornax"}
		words := strings.Split(params.Body, " ")
		cleanedWords := []string{}
		for _, word := range words {
			isProfane := false
			for _, badWord := range badWords {
				if strings.ToLower(word) == badWord {
					cleanedWords = append(cleanedWords, "****")
					isProfane = true
					break
				}
			}
			if !isProfane {
				cleanedWords = append(cleanedWords, word)
			}
		}
		cleanedBody := strings.Join(cleanedWords, " ")
		response := map[string]string{"cleaned_body": cleanedBody}
		json.NewEncoder(w).Encode(response)

	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "Chirp is too long"})
	}

}

func (cfg *apiConfig) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Hits: %d", cfg.fileserverHits)))
}

func (cfg *apiConfig) handleReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Counter reset successfully."))
}

func getHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	mux := http.NewServeMux()
	fmt.Print("starting server...")

	apiCfg := &apiConfig{}

	// File server for /app/ path
	fs := http.FileServer(http.Dir("."))
	mux.Handle("/app/*", apiCfg.middlewareMetrics(http.StripPrefix("/app/", fs)))

	// Use the correct path setup with HTTP method in mux.HandleFunc
	mux.HandleFunc("GET /api/healthz", getHealth)
	mux.HandleFunc("GET /api/metrics", apiCfg.handleMetrics)
	mux.HandleFunc("GET /admin/metrics", apiCfg.adminMetrics)
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.validateChrip)

	// Other handlers without method prefix
	mux.HandleFunc("/api/reset", apiCfg.handleReset)

	// Start the server
	http.ListenAndServe(":8080", mux)
}

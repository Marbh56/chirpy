package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type apiConfig struct {
	fileserverHits int
}

type Chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}


type DB struct {
	path string
	mux  *sync.RWMutex
	currentID int
}

func NewDB(path string) (*DB, error){
	db := &DB{
		path: path,
		mux: &sync.RWMutex{},
	}

	err := db.ensureDB()
	if err != nil {
		return nil, err
	}

	data, err := db.loadDB()
	if err != nil {
		return nil, err
	}
	if len(data.Chrips) > 0 {
		for id := range data.Chirps {
			if id >= db.currentID {
				db.currentID = id + 1
			}
		}
	}

	return db, nil

}

func (db *DB) ensureDB() error {
	_, err := os.Stat(db.path)
	if os.IsNotExist(err) {
		dbStructure := DBStructure{Chrips: make(map[int]Chirp)}
		return db.writeDB(&dbStructure)
	}
	return err
}

func (db *DB) loadDB() (DBStructure, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()
	bytes, err := os.ReadFile(db.path)
	if err != nil {
		return DBStructure{}, err
	}

	var dbStructure DBStructure
	err = json.Unmarshal(bytes, &dbStructure)
	if err != nil {
		return DBStructure{}, err
	}

	return dbStructure, nil
}

func (db *DB) writeDB(dbStructure *DBStructure) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	bytes, err := json.MarshalIndent(dbStructure, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(db.path, bytes, 0644)
}

func (db *DB) CreateChirp(body string) (Chrip, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	data, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	chirp := Chirp{
		ID: db.currentID
		Body: body,
	}

	data.Chirps[db.currentID] = chirp
	db.currentID++

	err = db.writeDB(&data)
	if err != nil {
		return Chirp{}, err
	}

	return chirp, nil
}

func (db *DB) GetChirps() ([]Chirp, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()

	data, err := db.loadDB()
	if err != nil {
		return nil, err
	}

	chirps := make([]Chirp, 0, len(data.Chirps))
	for _, chirp := range data.Chirps {
		chirps = append(chirps, chirp)
	}

	sort.Slice(chirps, func(i, j int) bool {
		return chirps[i].ID < chirps[j].ID
	})

	return chirps, nil
}

func createChirpHandler(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		 body, exist := requestBody["body"]
		 if !exist || len(body) == 0 {
			http.Error(w, "Invalid chirp body", http.StatusBadRequest)
			return
		 }

		 chirp, err := db.CreateChirp(body)
		 if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		 }

		 w.WriteHeader(http.StatusCreated)
		 if err := json.NewEncoder(w).Encode(chirp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		 }
		 
	}
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

func (cfg *apiConfig) isValidChrip(w http.ResponseWriter, r *http.Request) {
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

	// Handlers for the act of chirping
	mux.HandleFunc("POST /api/chirps", ????)
	mux.HandleFunc("GET /api/chirps", ????)

	// Other handlers without method prefix
	mux.HandleFunc("/api/reset", apiCfg.handleReset)

	// Start the server
	http.ListenAndServe(":8080", mux)
}

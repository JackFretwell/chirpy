package main

import(
	"net/http"
	"time"
	"log"
	"sync/atomic"
	"fmt"
	"encoding/json"
	"strings"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}


func healthCheck(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) writeNumberOfRequests(w http.ResponseWriter, req *http.Request) {
	hits := cfg.fileserverHits.Load()
	formattedText := fmt.Sprintf(`
		<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>
		</html>`,
	hits)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(formattedText))
}

func (cfg *apiConfig) resetFileserverHits (w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
}

func profanityFilter(s string) string {
	splitString := strings.Split(s, " ")
	for i := 0; i < len(splitString); i++ {
		if strings.ToLower(splitString[i]) == "kerfuffle" || strings.ToLower(splitString[i]) == "sharbert" || strings.ToLower(splitString[i]) == "fornax" {
			splitString[i] = "****"
		}
	}

	return strings.Join(splitString, " ")
}

func validateChirp(w http.ResponseWriter, req *http.Request) {
	type chirpBody struct {
		Body string `json:"body"`
	}

	type chirpValid struct {
		CleanedBody string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(req.Body)
	c := chirpBody{}
	err := decoder.Decode(&c)
	if err != nil {
		respondWithError(w, 400, "An occured when decoding the Chirp")
		return
	}

	if len(c.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	cleanText := profanityFilter(c.Body)

	respBody := chirpValid{
		CleanedBody: cleanText,
	}

	respondWithJSON(w, 200, respBody)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnVals struct {
		Error string `json:"error"`
	}

	respBody := returnVals{
		Error: msg,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}


func main() {
	cfg := apiConfig{}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", healthCheck)
	mux.HandleFunc("POST /api/validate_chirp", validateChirp)

	mux.HandleFunc("POST /admin/reset", cfg.resetFileserverHits)
	mux.HandleFunc("GET /admin/metrics", cfg.writeNumberOfRequests)

	s := &http.Server {
		Addr:			":8080",
		Handler:		mux,
		ReadTimeout:	10 * time.Second,
		WriteTimeout:	10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	//log.Printf("Serving files from %s on port: %s\n", )
	log.Fatal(s.ListenAndServe())
}

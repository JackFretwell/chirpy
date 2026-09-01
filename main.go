package main

import(
	"net/http"
	"time"
	"log"
	"sync/atomic"
	"fmt"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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


func main() {
	cfg := apiConfig{}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

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

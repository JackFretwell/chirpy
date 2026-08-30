package main

import(
	"net/http"
	"time"
	"log"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(".")))

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
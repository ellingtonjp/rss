package main

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"
)

//go:embed templates static
var staticFiles embed.FS

var apiKey string

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("key")), []byte(apiKey)) == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func main() {
	dbPath := flag.String("db", "", "path to SQLite database file (required)")
	flag.Parse()
	if *dbPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	apiKey = os.Getenv("RSS_API_KEY")
	if apiKey == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			log.Fatal("failed to generate API key:", err)
		}
		apiKey = hex.EncodeToString(b)
		log.Fatal("No RSS_API_KEY set")
	}

	if err := initDB(*dbPath); err != nil {
		log.Fatal("failed to init database:", err)
	}

	initTemplates()
	startScheduler()

	mux := http.NewServeMux()

	// Static files
	sub, _ := fs.Sub(staticFiles, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))

	// Pages
	mux.HandleFunc("GET /{$}", handleIndex)
	mux.HandleFunc("POST /preview", handlePreview)
	mux.HandleFunc("POST /preview/test", handlePreviewTest)
	mux.HandleFunc("POST /feeds", handleCreateFeed)
	mux.HandleFunc("POST /feeds/rss", handleCreateRSSFeed)
	mux.HandleFunc("GET /feeds", handleListFeeds)
	mux.HandleFunc("GET /feeds/opml", requireAPIKey(handleOPML))
	mux.HandleFunc("GET /feeds/{id}", handleFeedDetail)
	mux.HandleFunc("GET /feeds/{id}/edit", handleEditFeed)
	mux.HandleFunc("POST /feeds/{id}/edit", handleUpdateFeed)
	mux.HandleFunc("GET /feeds/{id}/rss", requireAPIKey(handleFeedRSS))
	mux.HandleFunc("DELETE /feeds/{id}", handleDeleteFeed)

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", logRequests(mux)))
}

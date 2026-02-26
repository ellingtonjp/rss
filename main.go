package main

import (
	"log"
	"net/http"
	"time"
)

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	if err := initDB("feeds.db"); err != nil {
		log.Fatal("failed to init database:", err)
	}

	initTemplates()
	startScheduler()

	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Pages
	mux.HandleFunc("GET /{$}", handleIndex)
	mux.HandleFunc("POST /preview", handlePreview)
	mux.HandleFunc("POST /preview/test", handlePreviewTest)
	mux.HandleFunc("POST /feeds", handleCreateFeed)
	mux.HandleFunc("POST /feeds/rss", handleCreateRSSFeed)
	mux.HandleFunc("GET /feeds", handleListFeeds)
	mux.HandleFunc("GET /feeds/opml", handleOPML)
	mux.HandleFunc("GET /feeds/{id}", handleFeedDetail)
	mux.HandleFunc("GET /feeds/{id}/edit", handleEditFeed)
	mux.HandleFunc("POST /feeds/{id}/edit", handleUpdateFeed)
	mux.HandleFunc("GET /feeds/{id}/rss", handleFeedRSS)
	mux.HandleFunc("DELETE /feeds/{id}", handleDeleteFeed)

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", logRequests(mux)))
}

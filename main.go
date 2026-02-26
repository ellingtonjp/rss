package main

import (
	"log"
	"net/http"
)

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
	mux.HandleFunc("GET /feeds", handleListFeeds)
	mux.HandleFunc("GET /feeds/{id}", handleFeedDetail)
	mux.HandleFunc("GET /feeds/{id}/rss", handleFeedRSS)
	mux.HandleFunc("DELETE /feeds/{id}", handleDeleteFeed)

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

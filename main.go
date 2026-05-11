package main

import (
	"log"
	"net/http"

	"real-time-forum/database"
	"real-time-forum/handlers"
)

func main() {
	database.Init("./forum.db")
	go handlers.GlobalHub.Run()

	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve index.html for the root and any non-API path (SPA fallback)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.ServeFile(w, r, "./static/index.html")
			return
		}
		http.ServeFile(w, r, "./static/index.html")
	})

	// Auth
	mux.HandleFunc("/api/register", handlers.Register)
	mux.HandleFunc("/api/login", handlers.Login)
	mux.HandleFunc("/api/logout", handlers.Logout)
	mux.HandleFunc("/api/me", handlers.Me)

	// Posts
	mux.HandleFunc("/api/categories", handlers.AuthMiddleware(handlers.GetCategories))
	mux.HandleFunc("/api/posts", handlers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetPosts(w, r)
		case http.MethodPost:
			handlers.CreatePost(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/posts/", handlers.AuthMiddleware(handlers.GetPost))
	mux.HandleFunc("/api/comments", handlers.AuthMiddleware(handlers.CreateComment))

	// Messages
	mux.HandleFunc("/api/messages", handlers.AuthMiddleware(handlers.GetMessages))
	mux.HandleFunc("/api/conversations", handlers.AuthMiddleware(handlers.GetConversations))

	// WebSocket
	mux.HandleFunc("/ws", handlers.ServeWS)

	log.Println("Forum running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

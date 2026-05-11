package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Nickname  string    `json:"nickname"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	Gender    string    `json:"gender"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Post struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Nickname   string     `json:"nickname"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Categories []Category `json:"categories"`
	Comments   []Comment  `json:"comments,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Comment struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	UserID    string    `json:"user_id"`
	Nickname  string    `json:"nickname"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID         string    `json:"id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	// Populated on read
	SenderNick   string `json:"sender_nick"`
	ReceiverNick string `json:"receiver_nick"`
}

// UserStatus is sent over WebSocket to describe online/offline state.
type UserStatus struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Online   bool   `json:"online"`
}

// WSMessage is the envelope for every WebSocket message.
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// ConversationEntry represents a user in the sidebar ordered by last message.
type ConversationEntry struct {
	UserID      string    `json:"user_id"`
	Nickname    string    `json:"nickname"`
	Online      bool      `json:"online"`
	LastMessage string    `json:"last_message"`
	LastAt      time.Time `json:"last_at"`
	HasMessages bool      `json:"has_messages"`
}

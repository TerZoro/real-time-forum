package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(path string) {
	var err error
	DB, err = sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err = DB.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	createTables()
}

func createTables() {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id         TEXT PRIMARY KEY,
		nickname   TEXT UNIQUE NOT NULL,
		first_name TEXT NOT NULL,
		last_name  TEXT NOT NULL,
		email      TEXT UNIQUE NOT NULL,
		age        INTEGER NOT NULL,
		gender     TEXT NOT NULL,
		password   TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id         TEXT PRIMARY KEY,
		user_id    TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS categories (
		id   INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS posts (
		id          TEXT PRIMARY KEY,
		user_id     TEXT NOT NULL,
		title       TEXT NOT NULL,
		content     TEXT NOT NULL,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS post_categories (
		post_id     TEXT NOT NULL,
		category_id INTEGER NOT NULL,
		PRIMARY KEY (post_id, category_id),
		FOREIGN KEY (post_id)     REFERENCES posts(id)      ON DELETE CASCADE,
		FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS comments (
		id         TEXT PRIMARY KEY,
		post_id    TEXT NOT NULL,
		user_id    TEXT NOT NULL,
		content    TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (post_id) REFERENCES posts(id)    ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id)    ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS messages (
		id          TEXT PRIMARY KEY,
		sender_id   TEXT NOT NULL,
		receiver_id TEXT NOT NULL,
		content     TEXT NOT NULL,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (sender_id)   REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (receiver_id) REFERENCES users(id) ON DELETE CASCADE
	);

	INSERT OR IGNORE INTO categories (name) VALUES
		('Technology'),
		('Gaming'),
		('Movies & TV'),
		('Music'),
		('Sports'),
		('Science'),
		('Politics'),
		('Other');
	`

	if _, err := DB.Exec(schema); err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}
}

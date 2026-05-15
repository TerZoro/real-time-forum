package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"real-time-forum/database"
	"real-time-forum/models"

	"github.com/gofrs/uuid"
)

func GetCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(`SELECT id, name FROM categories ORDER BY name`)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cats []models.Category
	for rows.Next() {
		var c models.Category
		rows.Scan(&c.ID, &c.Name)
		cats = append(cats, c)
	}
	jsonOK(w, cats)
}

type createPostRequest struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	Categories []int  `json:"categories"`
}

func CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := GetSessionUser(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)

	if req.Title == "" || req.Content == "" || len(req.Categories) == 0 {
		jsonError(w, "title, content and at least one category are required", http.StatusBadRequest)
		return
	}

	id, _ := uuid.NewV4()
	_, err := database.DB.Exec(
		`INSERT INTO posts (id, user_id, title, content) VALUES (?, ?, ?, ?)`,
		id.String(), user.ID, req.Title, req.Content,
	)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	for _, catID := range req.Categories {
		database.DB.Exec(
			`INSERT OR IGNORE INTO post_categories (post_id, category_id) VALUES (?, ?)`,
			id.String(), catID,
		)
	}

	post, _ := getPostByID(id.String(), user.ID)
	jsonOK(w, post)
}

func GetPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	me, _ := GetSessionUser(r)

	rows, err := database.DB.Query(
		`SELECT p.id, p.user_id, u.nickname, p.title, p.content, p.created_at,
		        COALESCE((SELECT SUM(value) FROM post_likes WHERE post_id = p.id), 0) AS score,
		        COALESCE((SELECT value FROM post_likes WHERE post_id = p.id AND user_id = ?), 0) AS user_vote
		 FROM posts p
		 JOIN users u ON p.user_id = u.id
		 ORDER BY p.created_at DESC`,
		me.ID,
	)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		rows.Scan(&p.ID, &p.UserID, &p.Nickname, &p.Title, &p.Content, &p.CreatedAt, &p.Score, &p.UserVote)
		p.Categories = getPostCategories(p.ID)
		posts = append(posts, p)
	}

	if posts == nil {
		posts = []models.Post{}
	}
	jsonOK(w, posts)
}

func GetPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/posts/")
	if id == "" {
		jsonError(w, "post id required", http.StatusBadRequest)
		return
	}

	me, _ := GetSessionUser(r)
	post, err := getPostByID(id, me.ID)
	if err != nil {
		jsonError(w, "post not found", http.StatusNotFound)
		return
	}

	// Load comments
	post.Comments = getPostComments(post.ID)
	jsonOK(w, post)
}

type createCommentRequest struct {
	PostID  string `json:"post_id"`
	Content string `json:"content"`
}

func CreateComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := GetSessionUser(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	req.PostID = strings.TrimSpace(req.PostID)

	if req.Content == "" || req.PostID == "" {
		jsonError(w, "post_id and content are required", http.StatusBadRequest)
		return
	}

	id, _ := uuid.NewV4()
	_, err := database.DB.Exec(
		`INSERT INTO comments (id, post_id, user_id, content) VALUES (?, ?, ?, ?)`,
		id.String(), req.PostID, user.ID, req.Content,
	)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	var comment models.Comment
	database.DB.QueryRow(
		`SELECT c.id, c.post_id, c.user_id, u.nickname, c.content, c.created_at
		 FROM comments c JOIN users u ON c.user_id = u.id
		 WHERE c.id = ?`, id.String(),
	).Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Nickname,
		&comment.Content, &comment.CreatedAt)

	jsonOK(w, comment)
}

func getPostByID(id string, userID string) (models.Post, error) {
	var p models.Post
	err := database.DB.QueryRow(
		`SELECT p.id, p.user_id, u.nickname, p.title, p.content, p.created_at,
		        COALESCE((SELECT SUM(value) FROM post_likes WHERE post_id = p.id), 0) AS score,
		        COALESCE((SELECT value FROM post_likes WHERE post_id = p.id AND user_id = ?), 0) AS user_vote
		 FROM posts p
		 JOIN users u ON p.user_id = u.id
		 WHERE p.id = ?`,
		userID, id,
	).Scan(&p.ID, &p.UserID, &p.Nickname, &p.Title, &p.Content, &p.CreatedAt, &p.Score, &p.UserVote)
	if err != nil {
		return p, err
	}
	p.Categories = getPostCategories(p.ID)
	return p, nil
}

func getPostCategories(postID string) []models.Category {
	rows, err := database.DB.Query(
		`SELECT c.id, c.name FROM categories c
		 JOIN post_categories pc ON c.id = pc.category_id
		 WHERE pc.post_id = ?`, postID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var cats []models.Category
	for rows.Next() {
		var c models.Category
		rows.Scan(&c.ID, &c.Name)
		cats = append(cats, c)
	}
	return cats
}

func getPostComments(postID string) []models.Comment {
	rows, err := database.DB.Query(
		`SELECT c.id, c.post_id, c.user_id, u.nickname, c.content, c.created_at
		 FROM comments c JOIN users u ON c.user_id = u.id
		 WHERE c.post_id = ? ORDER BY c.created_at ASC`, postID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Nickname, &c.Content, &c.CreatedAt)
		comments = append(comments, c)
	}
	return comments
}

func LikePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := GetSessionUser(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	postID := strings.TrimPrefix(r.URL.Path, "/api/posts/")
	postID = strings.TrimSuffix(postID, "/like")
	if postID == "" {
		jsonError(w, "post id required", http.StatusBadRequest)
		return
	}

	var body struct {
		Value int `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || (body.Value != 1 && body.Value != -1) {
		jsonError(w, "value must be 1 or -1", http.StatusBadRequest)
		return
	}

	var current int
	err := database.DB.QueryRow(
		`SELECT value FROM post_likes WHERE post_id = ? AND user_id = ?`,
		postID, user.ID,
	).Scan(&current)

	if err == nil && current == body.Value {
		database.DB.Exec(`DELETE FROM post_likes WHERE post_id = ? AND user_id = ?`, postID, user.ID)
	} else {
		database.DB.Exec(
			`INSERT INTO post_likes (post_id, user_id, value) VALUES (?, ?, ?)
			 ON CONFLICT (post_id, user_id) DO UPDATE SET value = excluded.value`,
			postID, user.ID, body.Value,
		)
	}

	var score int
	database.DB.QueryRow(
		`SELECT COALESCE(SUM(value), 0) FROM post_likes WHERE post_id = ?`,
		postID,
	).Scan(&score)

	jsonOK(w, map[string]int{"score": score})
}

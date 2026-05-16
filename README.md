# Real-Time Forum

A full-stack forum with live chat, built from scratch with Go and vanilla JavaScript.

---

## Features

- **Register & login** — sign up with nickname, email, name, age, gender. Login with nickname or email
- **Session auth** — HttpOnly cookie, persists for 24 hours
- **Post feed** — browse all posts with categories, author, and timestamp
- **Create posts** — title, content, pick one or more categories
- **Comments** — comment on any post
- **Like / dislike posts** — Reddit-style vote buttons, one vote per user, toggle off by clicking again
- **Private chat** — real-time direct messages between users
- **Message history** — paginated, loads 10 messages at a time, scroll up to load older
- **Online presence** — see who is online in the sidebar, live updates when users join or leave
- **Single page app** — no full page reloads, instant navigation
- **Responsive** — works on desktop, tablet, and mobile

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go |
| Database | SQLite |
| Real-time | WebSocket (gorilla/websocket) |
| Frontend | Vanilla JavaScript (no frameworks) |
| Auth | bcrypt passwords, session cookies |

---

## Project Structure

```
real-time-forum/
├── main.go                  # HTTP server, all routes
├── database/
│   └── db.go                # SQLite init and schema
├── models/
│   └── models.go            # Go structs (User, Post, Message, etc.)
├── handlers/
│   ├── auth.go              # Register, Login, Logout, session middleware
│   ├── posts.go             # Posts, comments, like/dislike
│   ├── messages.go          # Paginated message history, conversations sidebar
│   └── websocket.go         # WebSocket hub, real-time delivery
└── static/
    ├── index.html           # Single page shell (all views inside)
    ├── css/
    │   └── style.css        # Dark theme, responsive layout
    └── js/
        ├── app.js           # Router, boots everything, mobile sidebar toggle
        ├── auth.js          # Login and register forms
        ├── posts.js         # Feed, single post, comments, new post, voting
        ├── messages.js      # Chat, sidebar, pagination
        └── websocket.js     # WebSocket client, reconnect logic
```

---

## Requirements

- Go 1.21 or later
- GCC (required by go-sqlite3 for CGO)

On macOS GCC comes with Xcode Command Line Tools:
```
xcode-select --install
```

On Linux (Debian/Ubuntu):
```
sudo apt install gcc
```

---

## Run

```bash
git clone <repo-url>
cd real-time-forum
go run .
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

The database file `forum.db` is created automatically on first run. No migrations needed.

> **Note:** always use `go run .` during development, not a pre-compiled binary.

---

## API

All API routes return JSON. All routes except `/api/register`, `/api/login`, and `/api/me` require a valid session cookie.

### Auth

| Method | Route | Description |
|---|---|---|
| POST | /api/register | Create account |
| POST | /api/login | Login |
| POST | /api/logout | Logout |
| GET | /api/me | Get current user from session |

### Posts

| Method | Route | Description |
|---|---|---|
| GET | /api/posts | All posts with score and current user vote |
| POST | /api/posts | Create post |
| GET | /api/posts/:id | Single post with comments, score, and current user vote |
| POST | /api/posts/:id/like | Like or dislike a post (`{"value": 1}` or `{"value": -1}`) |
| POST | /api/comments | Add comment |
| GET | /api/categories | All categories |

### Messages

| Method | Route | Description |
|---|---|---|
| GET | /api/messages?with=userID&offset=0 | Paginated message history (10 per page) |
| GET | /api/conversations | All users ordered by last message |

### WebSocket

| Route | Description |
|---|---|
| /ws | WebSocket connection (requires session cookie) |

WebSocket message types:

| Type | Direction | Description |
|---|---|---|
| online_users | server → client | Full list of online users on connect |
| presence | server → client | Single user came online or went offline |
| private_message | both directions | Send or receive a chat message |

---

## Categories

The following categories are seeded automatically:

Technology, Gaming, Movies & TV, Music, Sports, Science, Politics, Other

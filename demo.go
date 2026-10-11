package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"
)

// Demo mode fills the app with sample data so the layout can be looked at
// without a backend, a database or an account:
//
//	FYNCHAT_DEMO=1 go run .
//
// It serves the same routes the real backend does, from memory, so nothing in
// the app takes a different path — the point is to see the real views, not a
// mock of them.
func demoEnabled() bool { return os.Getenv("FYNCHAT_DEMO") != "" }

// startDemoServer returns a base URL serving canned data and a stop function.
func startDemoServer() (string, func()) {
	now := time.Now()

	channels := []map[string]any{
		{"id": demoChannelGeneral, "channel_name": "general", "channel_avatar": "", "channel_type": "group"},
		{"id": demoChannelDesign, "channel_name": "design-team", "channel_avatar": "", "channel_type": "channel"},
		{"id": demoChannelDM, "channel_name": "ada", "channel_avatar": "", "channel_type": "p2p"},
	}

	// A conversation long enough to show grouping, both bubble styles, the
	// date separators and a wrapped message.
	messages := []map[string]any{
		demoMessage("m1", demoChannelGeneral, 2, "morning everyone — the desktop build is finally green", now.Add(-72*time.Hour)),
		demoMessage("m2", demoChannelGeneral, 2, "CI passed on all three platforms", now.Add(-72*time.Hour+2*time.Minute)),
		demoMessage("m3", demoChannelGeneral, 3, "nice. does the channel list still 404?", now.Add(-71*time.Hour)),
		demoMessage("m4", demoChannelGeneral, 1, "no — we added GET /api/channels yesterday. try the reload button in the header", now.Add(-70*time.Hour)),
		demoMessage("m5", demoChannelGeneral, 3, "even better. this is a deliberately long message so you can see how a bubble wraps once it runs out of width instead of stretching all the way across the window", now.Add(time.Hour)),
		demoMessage("m6", demoChannelGeneral, 1, "and the send button locks while it is in flight now", now.Add(time.Hour-time.Minute)),
		demoMessage("m7", demoChannelGeneral, 1, "ready to ship it", now.Add(50*time.Minute)),
	}

	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, body any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	}

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"token": "demo-token",
			"user":  map[string]any{"id": 1, "name": "Ada", "username": "ada", "email": "ada@example.com"},
		})
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"token": "demo-token",
			"user":  map[string]any{"id": 1, "name": "Ada", "username": "ada", "email": "ada@example.com"},
		})
	})
	mux.HandleFunc("/api/channels", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, channels)
	})
	mux.HandleFunc("/api/channel/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, map[string]string{"channel_name": "general", "channel_avatar": ""})
	})
	mux.HandleFunc("/api/channel/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"channel_name": "general", "channel_avatar": ""})
	})
	mux.HandleFunc("/api/message/create/", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Content string `json:"content"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, demoMessage(fmt.Sprintf("sent-%d", time.Now().UnixNano()), demoChannelGeneral, 1, body.Content, time.Now()))
	})
	mux.HandleFunc("/api/channels/", func(w http.ResponseWriter, r *http.Request) {
		channelID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/channels/"), "/messages")

		var out []map[string]any
		for _, m := range messages {
			if m["channel_id"] == channelID {
				out = append(out, m)
			}
		}
		writeJSON(w, out)
	})

	srv := httptest.NewServer(mux)
	return srv.URL, srv.Close
}

const (
	demoChannelGeneral = "11111111-1111-4111-8111-111111111111"
	demoChannelDesign  = "22222222-2222-4222-8222-222222222222"
	demoChannelDM      = "33333333-3333-4333-8333-333333333333"
)

func demoMessage(id, channelID string, sender int, content string, at time.Time) map[string]any {
	return map[string]any{
		"id":         id,
		"channel_id": channelID,
		"sender_id":  sender,
		"content":    content,
		"created_at": at.Format(time.RFC3339),
	}
}

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer starts an httptest server, points the package-level baseURL at
// it for the duration of the test and returns the server.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	old := baseURL
	baseURL = srv.URL
	t.Cleanup(func() {
		baseURL = old
		srv.Close()
	})
	return srv
}

// decodeBody reads and closes the request body as JSON.
func decodeBody(t *testing.T, r *http.Request) map[string]string {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode body %q: %v", raw, err)
	}
	return out
}

func TestBaseURLFromEnv(t *testing.T) {
	t.Run("default when unset", func(t *testing.T) {
		t.Setenv("FYNCHAT_API_URL", "")
		if got := baseURLFromEnv(); got != defaultBaseURL {
			t.Errorf("baseURLFromEnv() = %q, want %q", got, defaultBaseURL)
		}
	})

	t.Run("from environment", func(t *testing.T) {
		t.Setenv("FYNCHAT_API_URL", "https://chat.example.com/")
		if got := baseURLFromEnv(); got != "https://chat.example.com" {
			t.Errorf("baseURLFromEnv() = %q, want %q", got, "https://chat.example.com")
		}
	})
}

func TestLogin(t *testing.T) {
	newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/login" {
			t.Errorf("got %s %s, want POST /login", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("login must not send Authorization, got %q", got)
		}
		if got := decodeBody(t, r)["email"]; got != "me@example.com" {
			t.Errorf("email = %q, want me@example.com", got)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"access_token":"jwt-123","user":{"id":1}}`)
	})

	c := NewHTTPClient()
	token, err := c.Login("me@example.com", "hunter22")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token != "jwt-123" {
		t.Errorf("token = %q, want jwt-123", token)
	}
	if c.Token != "jwt-123" {
		t.Errorf("client token not stored, got %q", c.Token)
	}
}

// The fynchat-backend handlers return "token" instead of "access_token".
func TestLoginAcceptsBackendTokenField(t *testing.T) {
	newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"token":"backend-token","user":{"id":1}}`)
	})

	c := NewHTTPClient()
	token, err := c.Login("me@example.com", "hunter22")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token != "backend-token" {
		t.Errorf("token = %q, want backend-token", token)
	}
}

func TestLoginError(t *testing.T) {
	newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":"invalid credentials"}`)
	})

	c := NewHTTPClient()
	token, err := c.Login("me@example.com", "wrong")
	if err == nil {
		t.Fatal("expected an error for 401 response")
	}
	if token != "" {
		t.Errorf("token = %q, want empty on failure", token)
	}
	if c.Token != "" {
		t.Errorf("client token = %q, want empty on failure", c.Token)
	}
}

func TestRegister(t *testing.T) {
	newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/register" {
			t.Errorf("path = %q, want /register", r.URL.Path)
		}
		body := decodeBody(t, r)
		want := map[string]string{
			"name": "Ada", "username": "ada", "email": "ada@example.com", "password": "hunter22",
		}
		for k, v := range want {
			if body[k] != v {
				t.Errorf("body[%q] = %q, want %q", k, body[k], v)
			}
		}
		io.WriteString(w, `{"access_token":"jwt-new","user":{"id":7}}`)
	})

	c := NewHTTPClient()
	token, err := c.Register("Ada", "ada", "ada@example.com", "hunter22")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if token != "jwt-new" {
		t.Errorf("token = %q, want jwt-new", token)
	}
}

func TestListChannelsSendsBearer(t *testing.T) {
	c := NewHTTPClient()
	c.Token = "jwt-abc"

	newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/channels" {
			t.Errorf("got %s %s, want GET /api/channels", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer jwt-abc" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer jwt-abc")
		}
		io.WriteString(w, `[
			{"id":"c1","channel_name":"general","channel_avatar":"g.png"},
			{"id":"c2","channel_name":"random","channel_avatar":"r.png"}
		]`)
	})

	channels, err := c.ListChannels()
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("got %d channels, want 2", len(channels))
	}
	if channels[0].ID != "c1" || channels[0].ChannelName != "general" || string(channels[0].ChannelAvatar) != "g.png" {
		t.Errorf("channels[0] = %+v", channels[0])
	}
}

// Opening a private chat is a get-or-create: the client asks for the chat with
// a person and does not care whether that made one. What it must send is the
// peer, since that is the only thing identifying the conversation.
func TestOpenPrivateChatSendsThePeerOnly(t *testing.T) {
	c := NewHTTPClient()
	c.Token = "jwt-abc"

	newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/channel/create" {
			t.Errorf("got %s %s, want POST /api/channel/create", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer jwt-abc" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer jwt-abc")
		}

		// Decoded as map[string]any rather than through decodeBody, because
		// peer_id is a number and decodeBody only handles string values.
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode body %q: %v", raw, err)
		}

		if body["channel_type"] != "p2p" {
			t.Errorf("channel_type = %v, want p2p", body["channel_type"])
		}
		if body["peer_id"] != float64(42) {
			t.Errorf("peer_id = %v, want 42", body["peer_id"])
		}
		// The name belongs to the backend, which names the chat after the two
		// people in it, so sending one would leave it guessing.
		if _, present := body["name"]; present {
			t.Errorf("body = %v; a private chat must not send a name", body)
		}

		io.WriteString(w, `{"id":"c9","channel_name":"Grace Hopper","channel_avatar":"","channel_type":"p2p"}`)
	})

	channel, err := c.OpenPrivateChat(42)
	if err != nil {
		t.Fatalf("OpenPrivateChat: %v", err)
	}
	if channel.ChannelName != "Grace Hopper" || channel.ChannelType != "p2p" {
		t.Errorf("channel = %+v", channel)
	}
}

// Adding a contact sends the handle, because that is what names an account.
func TestAddContactSendsTheHandle(t *testing.T) {
	c := NewHTTPClient()
	c.Token = "jwt-abc"

	newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/contacts" {
			t.Errorf("got %s %s, want POST /api/contacts", r.Method, r.URL.Path)
		}
		body := decodeBody(t, r)
		if body["username"] != "kj" {
			t.Errorf("body = %v, want the handle", body)
		}

		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"id":4,"name":"Katherine Johnson","username":"kj","avatar_url":""}`)
	})

	contact, err := c.AddContact("kj")
	if err != nil {
		t.Fatalf("AddContact: %v", err)
	}
	if contact.ID != 4 || contact.Name != "Katherine Johnson" || contact.Username != "kj" {
		t.Errorf("contact = %+v", contact)
	}
}

// The contacts the private-chat screen offers come from their own route.
func TestListContacts(t *testing.T) {
	c := NewHTTPClient()
	c.Token = "jwt-abc"

	newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/contacts" {
			t.Errorf("got %s %s, want GET /api/contacts", r.Method, r.URL.Path)
		}
		io.WriteString(w, `[
			{"id":2,"name":"Grace Hopper","username":"amazing","avatar_url":"g.png"},
			{"id":3,"name":"Alan Turing","username":"turing","avatar_url":""}
		]`)
	})

	contacts, err := c.ListContacts()
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if len(contacts) != 2 {
		t.Fatalf("got %d contacts, want 2", len(contacts))
	}
	if contacts[0].ID != 2 || contacts[0].Name != "Grace Hopper" || contacts[0].Username != "amazing" {
		t.Errorf("contacts[0] = %+v", contacts[0])
	}
	if contacts[1].Avatar != "" {
		t.Errorf("contacts[1] = %+v", contacts[1])
	}
}

func TestCreateChannel(t *testing.T) {
	c := NewHTTPClient()
	c.Token = "jwt-abc"

	newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/channel/create" {
			t.Errorf("got %s %s, want POST /api/channel/create", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer jwt-abc" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer jwt-abc")
		}
		body := decodeBody(t, r)
		if body["channel_type"] != "text" || body["name"] != "dev" || body["avatar_url"] != "d.png" {
			t.Errorf("body = %v", body)
		}
		// A named channel has no second member, so peer_id must be absent
		// rather than sent as zero: 0 would be indistinguishable from a peer
		// that was never set.
		if _, present := body["peer_id"]; present {
			t.Errorf("body = %v; a named channel must not send peer_id", body)
		}
		io.WriteString(w, `{"channel_name":"dev","channel_avatar":"d.png"}`)
	})

	channel, err := c.CreateChannel("text", "dev", []byte("d.png"))
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if channel.ChannelName != "dev" || string(channel.ChannelAvatar) != "d.png" {
		t.Errorf("channel = %+v", channel)
	}
}

func TestListMessages(t *testing.T) {
	c := NewHTTPClient()
	c.Token = "jwt-abc"

	newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/channels/c1/messages" {
			t.Errorf("got %s %s, want GET /api/channels/c1/messages", r.Method, r.URL.Path)
		}
		io.WriteString(w, `[
			{"id":"m1","channel_id":"c1","sender_id":1,"content":"hi","created_at":"2026-10-11T12:00:00Z"},
			{"id":"m2","channel_id":"c1","sender_id":2,"content":"hey","created_at":"2026-10-11T12:00:05Z"}
		]`)
	})

	messages, err := c.ListMessages("c1")
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(messages))
	}
	if messages[1].ID != "m2" || messages[1].SenderID != 2 || messages[1].Content != "hey" {
		t.Errorf("messages[1] = %+v", messages[1])
	}
}

func TestSendMessage(t *testing.T) {
	c := NewHTTPClient()
	c.Token = "jwt-abc"

	newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/message/create/c1" {
			t.Errorf("got %s %s, want POST /api/message/create/c1", r.Method, r.URL.Path)
		}
		if got := decodeBody(t, r)["content"]; got != "hello world" {
			t.Errorf("content = %q, want hello world", got)
		}
		io.WriteString(w, `{"id":"m9","channel_id":"c1","sender_id":1,"content":"hello world","created_at":"2026-10-11T12:01:00Z"}`)
	})

	msg, err := c.SendMessage("c1", "hello world")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg.ID != "m9" || msg.Content != "hello world" || msg.ChannelID != "c1" {
		t.Errorf("message = %+v", msg)
	}
}

func TestMalformedJSONIsReported(t *testing.T) {
	newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `not json at all`)
	})

	c := NewHTTPClient()
	if _, err := c.ListChannels(); err == nil {
		t.Fatal("expected a decode error for a non-JSON response")
	}
}

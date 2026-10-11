package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"fyne.io/fyne/v2"
)

// defaultBaseURL is used when FYNCHAT_API_URL is not set.
const defaultBaseURL = "http://localhost:8080"

// Keys used to store the session in the app preferences.
const (
	prefToken  = "session.token"
	prefUserID = "session.user_id"
)

// Routes served by FynChat/fynchat-backend (internal/routes/routes.go).
// Everything that needs a JWT lives under the authenticated /api group.
const (
	routeLogin         = "/login"
	routeRegister      = "/register"
	routeChannels      = "/api/channels"
	routeChannelCreate = "/api/channel/create"
	routeChannelMsgs   = "/api/channels/%s/messages"
	routeMessageCreate = "/api/message/create/%s"
)

// baseURL is the root of the FynChat backend. Set FYNCHAT_API_URL to point
// the app at another host, e.g. FYNCHAT_API_URL=http://localhost:8000.
//
//	var baseURL = baseURLFromEnv()
var baseURL = baseURLFromEnv()

// baseURLFromEnv returns FYNCHAT_API_URL without a trailing slash, falling
// back to defaultBaseURL when the variable is empty.
func baseURLFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("FYNCHAT_API_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultBaseURL
}

// Channel is a chat channel as returned by the backend.
type Channel struct {
	ID            string `json:"id"`
	ChannelName   string `json:"channel_name"`
	ChannelAvatar string `json:"channel_avatar"`
	ChannelType   string `json:"channel_type"`
}

// Message is a single chat message as returned by the backend.
type Message struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	SenderID  int    `json:"sender_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// HTTPClient talks to the FynChat backend. It holds the auth token used
// for authenticated requests, and the id of the logged-in user, which the UI
// needs to tell "my" messages apart from everyone else's.
type HTTPClient struct {
	Token  string
	UserID int
	Client *http.Client

	prefs fyne.Preferences
}

// NewHTTPClient returns an HTTPClient with a default http.Client.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{Client: &http.Client{}}
}

// AttachPersistence makes the client remember the session between runs, so
// the app opens on the channel list instead of the login screen. Pass nil to
// keep the session in memory only, which is what the tests do.
func (c *HTTPClient) AttachPersistence(prefs fyne.Preferences) {
	c.prefs = prefs
	if prefs == nil {
		return
	}

	c.Token = prefs.String(prefToken)
	c.UserID = prefs.Int(prefUserID)
}

// Logout clears the session in memory and on disk.
func (c *HTTPClient) Logout() {
	c.Token = ""
	c.UserID = 0

	if c.prefs != nil {
		// Fyne's Preferences has no Remove, so an empty value is the
		// "logged out" state: AttachPersistence reads "" as no token.
		c.prefs.SetString(prefToken, "")
		c.prefs.SetInt(prefUserID, 0)
	}
}

// do sends a request and decodes the JSON response into out.
// If authenticated is true, the Authorization: Bearer <token> header is set.
func (c *HTTPClient) do(method, path string, body interface{}, authenticated bool, out interface{}) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, string(snippet))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode %s %s response: %w", method, path, err)
		}
	}
	return nil
}

type authResponse struct {
	AccessToken string   `json:"access_token"`
	Token       string   `json:"token"` // fynchat-backend names it "token"
	User        authUser `json:"user"`
}

// authUser is the user object the backend returns next to the token. Only
// the id is needed by the UI; the rest is kept so a future view can show a
// profile without another request.
type authUser struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// accessToken returns the token from the response, accepting both the
// "access_token" field used by the API spec and the "token" field returned
// by the current fynchat-backend handlers.
func (r authResponse) accessToken() string {
	if r.AccessToken != "" {
		return r.AccessToken
	}
	return r.Token
}

// store keeps the token and the signed-in user's id on the client, and on
// disk when persistence is attached.
func (c *HTTPClient) store(r authResponse) string {
	c.Token = r.accessToken()
	c.UserID = r.User.ID

	if c.prefs != nil {
		c.prefs.SetString(prefToken, c.Token)
		c.prefs.SetInt(prefUserID, c.UserID)
	}
	return c.Token
}

// Login authenticates with email/password and stores the returned token.
func (c *HTTPClient) Login(email, password string) (string, error) {
	body := map[string]string{"email": email, "password": password}
	var resp authResponse
	if err := c.do(http.MethodPost, routeLogin, body, false, &resp); err != nil {
		return "", err
	}
	return c.store(resp), nil
}

// Register creates a new account and stores the returned token.
func (c *HTTPClient) Register(name, username, email, password string) (string, error) {
	body := map[string]string{"name": name, "username": username, "email": email, "password": password}
	var resp authResponse
	if err := c.do(http.MethodPost, routeRegister, body, false, &resp); err != nil {
		return "", err
	}
	return c.store(resp), nil
}

// ListChannels returns all channels visible to the authenticated user.
func (c *HTTPClient) ListChannels() ([]Channel, error) {
	var channels []Channel
	if err := c.do(http.MethodGet, routeChannels, nil, true, &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

// CreateChannel creates a new channel and returns it.
func (c *HTTPClient) CreateChannel(channelType, name, avatar string) (Channel, error) {
	body := map[string]string{"channel_type": channelType, "name": name, "avatar_url": avatar}
	var channel Channel
	if err := c.do(http.MethodPost, routeChannelCreate, body, true, &channel); err != nil {
		return Channel{}, err
	}
	return channel, nil
}

// ListMessages returns all messages in the given channel.
func (c *HTTPClient) ListMessages(channelID string) ([]Message, error) {
	var messages []Message
	if err := c.do(http.MethodGet, fmt.Sprintf(routeChannelMsgs, channelID), nil, true, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// SendMessage posts a new message to the given channel and returns it.
func (c *HTTPClient) SendMessage(channelID, content string) (Message, error) {
	body := map[string]string{"content": content}
	var message Message
	if err := c.do(http.MethodPost, fmt.Sprintf(routeMessageCreate, channelID), body, true, &message); err != nil {
		return Message{}, err
	}
	return message, nil
}

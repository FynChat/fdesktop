package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"fyne.io/fyne/v2"
)

// Keys used to store the session and small bits of UI state in the app
// preferences, so the app opens where you left it.
const (
	prefToken       = "session.token"
	prefUserID      = "session.user_id"
	prefUserName    = "session.user_name"
	prefLastChannel = "session.last_channel"
	prefThemeDark   = "ui.dark_theme"
)

// defaultBaseURL is used when FYNCHAT_API_URL is not set.
const defaultBaseURL = "http://localhost:8080"

// Prefs wraps the app preferences in typed helpers, so the storage layout of
// the session and the small bits of UI state lives in one place.
type Prefs struct {
	p fyne.Preferences
}

// NewPrefs wraps the app preferences.
func NewPrefs(p fyne.Preferences) *Prefs { return &Prefs{p: p} }

// LastChannel returns the id of the channel that was open last time, if any.
func (s *Prefs) LastChannel() string { return s.p.String(prefLastChannel) }

// SetLastChannel remembers the channel to reopen on the next launch.
func (s *Prefs) SetLastChannel(id string) { s.p.SetString(prefLastChannel, id) }

// UserName returns the name of the signed-in user, if it was remembered.
func (s *Prefs) UserName() string { return s.p.String(prefUserName) }

// DarkTheme reports whether the dark theme is selected.
func (s *Prefs) DarkTheme() bool { return s.p.BoolWithFallback(prefThemeDark, true) }

// SetDarkTheme stores the theme choice.
func (s *Prefs) SetDarkTheme(dark bool) { s.p.SetBool(prefThemeDark, dark) }

// apply puts the stored theme on the running app.
func (s *Prefs) apply() {
	if s.DarkTheme() {
		currentIsDark = true
		fyne.CurrentApp().Settings().SetTheme(newTelegramTheme())
	} else {
		currentIsDark = false
		fyne.CurrentApp().Settings().SetTheme(newLightTheme())
	}
}

// Routes served by FynChat/fynchat-backend (internal/routes/routes.go).
// Everything that needs a JWT lives under the authenticated /api group.
const (
	routeLogin         = "/login"
	routeRegister      = "/register"
	routeChannels      = "/api/channels"
	routeChannelCreate = "/api/channel/create"
	routeChannelMsgs   = "/api/channels/%s/messages"
	routeMessageCreate = "/api/message/create/%s"
	routeContacts      = "/api/contacts"
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
	ChannelAvatar []byte `json:"channel_avatar"`
	ChannelType   string `json:"channel_type"`
}

// UnmarshalJSON handles the fact that the backend sends channel_avatar as a
// plain string (like "d.png" or base64), but Go's []byte expects base64.
// This custom unmarshaler accepts both formats.
func (c *Channel) UnmarshalJSON(data []byte) error {
	type Alias Channel
	aux := &struct {
		ChannelAvatar interface{} `json:"channel_avatar"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Convert the avatar data to []byte
	if aux.ChannelAvatar != nil {
		switch v := aux.ChannelAvatar.(type) {
		case string:
			c.ChannelAvatar = []byte(v)
		case nil:
			c.ChannelAvatar = nil
		}
	}

	return nil
}

// Message is a single chat message as returned by the backend.
type Message struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	SenderID  int    `json:"sender_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// Contact is a person who can be started off in a private chat.
type Contact struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Avatar   string `json:"avatar_url"`
}

// HTTPClient talks to the FynChat backend. It holds the auth token used
// for authenticated requests, and the id of the logged-in user, which the UI
// needs to tell "my" messages apart from everyone else's.
type HTTPClient struct {
	Token  string
	UserID int
	Name   string
	Client *http.Client

	prefs            fyne.Preferences
	onSessionExpired func()
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
	c.Name = prefs.String(prefUserName)
}

// Logout clears the session in memory and on disk.
func (c *HTTPClient) Logout() {
	c.Token = ""
	c.UserID = 0
	c.Name = ""

	if c.prefs != nil {
		// Fyne's Preferences has no Remove, so an empty value is the
		// "logged out" state: AttachPersistence reads "" as no token.
		c.prefs.SetString(prefToken, "")
		c.prefs.SetInt(prefUserID, 0)
		c.prefs.SetString(prefUserName, "")
	}
}

// ErrUnauthorized is returned when the backend rejects the stored token.
// Views use it to send the user back to the login screen instead of showing a
// dialog for every request that fails from then on.
var ErrUnauthorized = errors.New("your session has expired, please sign in again")

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

		// A 401 on an authenticated call means the token expired. /login and
		// /register also answer 401 for a wrong password, which is not an
		// expired session, so only the authenticated ones count.
		if resp.StatusCode == http.StatusUnauthorized && authenticated {
			c.sessionExpired()
			return fmt.Errorf("%s %s: %w", method, path, ErrUnauthorized)
		}
		return fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, string(snippet))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode %s %s response: %w", method, path, err)
		}
	}
	return nil
}

// SetSessionExpiredHandler registers a callback run when the backend rejects
// the token. It is called on the goroutine that made the request, so the
// handler must marshal to the UI thread with fyne.Do itself.
func (c *HTTPClient) SetSessionExpiredHandler(fn func()) {
	c.onSessionExpired = fn
}

func (c *HTTPClient) sessionExpired() {
	if c.onSessionExpired != nil {
		c.onSessionExpired()
	}
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
	c.Name = r.User.Name

	if c.prefs != nil {
		c.prefs.SetString(prefToken, c.Token)
		c.prefs.SetInt(prefUserID, c.UserID)
		c.prefs.SetString(prefUserName, c.Name)
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

// CreateChannel creates a named channel: a group or a broadcast. A private
// chat is not created here — it comes from a contact, see OpenPrivateChat.
func (c *HTTPClient) CreateChannel(channelType, name string, avatarData []byte) (Channel, error) {
	body := map[string]interface{}{"channel_type": channelType, "name": name}
	if len(avatarData) > 0 {
		// Send avatar as base64 encoded string for JSON transport
		body["avatar_url"] = string(avatarData)
	}
	var channel Channel
	if err := c.do(http.MethodPost, routeChannelCreate, body, true, &channel); err != nil {
		return Channel{}, err
	}
	return channel, nil
}

// OpenPrivateChat returns the private chat with the given person, creating it
// if it does not exist yet.
//
// It is deliberately a "get or create" rather than a plain create, because a
// private chat is identified by the pair of people in it: there can only ever
// be one, and asking for it twice must not make two. The backend decides which
// of the two happened; the client only cares about the channel that comes back.
func (c *HTTPClient) OpenPrivateChat(peerID int) (Channel, error) {
	body := map[string]interface{}{"channel_type": "p2p", "peer_id": peerID}

	var channel Channel
	if err := c.do(http.MethodPost, routeChannelCreate, body, true, &channel); err != nil {
		return Channel{}, err
	}
	return channel, nil
}

// AddContact adds someone to the user's contacts.
//
// The handle is what identifies them, not a display name: the backend has to
// resolve it to a real account, and picking a name is how you end up with two
// contacts for one person.
func (c *HTTPClient) AddContact(username string) (Contact, error) {
	body := map[string]string{"username": username}

	var contact Contact
	if err := c.do(http.MethodPost, routeContacts, body, true, &contact); err != nil {
		return Contact{}, err
	}
	return contact, nil
}

// ListContacts returns the people who can be started off in a private chat.
func (c *HTTPClient) ListContacts() ([]Contact, error) {
	var contacts []Contact
	if err := c.do(http.MethodGet, routeContacts, nil, true, &contacts); err != nil {
		return nil, err
	}
	return contacts, nil
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

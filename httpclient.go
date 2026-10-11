package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const baseURL = "http://localhost:8080"

// Channel is a chat channel as returned by the backend.
type Channel struct {
	ID            string `json:"id"`
	ChannelName   string `json:"channel_name"`
	ChannelAvatar string `json:"channel_avatar"`
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
// for authenticated requests.
type HTTPClient struct {
	Token  string
	Client *http.Client
}

// NewHTTPClient returns an HTTPClient with a default http.Client.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{Client: &http.Client{}}
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
	AccessToken string          `json:"access_token"`
	User        json.RawMessage `json:"user"`
}

// Login authenticates with email/password and stores the returned token.
func (c *HTTPClient) Login(email, password string) (string, error) {
	body := map[string]string{"email": email, "password": password}
	var resp authResponse
	if err := c.do(http.MethodPost, "/login", body, false, &resp); err != nil {
		return "", err
	}
	c.Token = resp.AccessToken
	return resp.AccessToken, nil
}

// Register creates a new account and stores the returned token.
func (c *HTTPClient) Register(name, username, email, password string) (string, error) {
	body := map[string]string{"name": name, "username": username, "email": email, "password": password}
	var resp authResponse
	if err := c.do(http.MethodPost, "/register", body, false, &resp); err != nil {
		return "", err
	}
	c.Token = resp.AccessToken
	return resp.AccessToken, nil
}

// ListChannels returns all channels visible to the authenticated user.
func (c *HTTPClient) ListChannels() ([]Channel, error) {
	var channels []Channel
	if err := c.do(http.MethodGet, "/channels", nil, true, &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

// CreateChannel creates a new channel and returns it.
func (c *HTTPClient) CreateChannel(channelType, name, avatar string) (Channel, error) {
	body := map[string]string{"channel_type": channelType, "name": name, "avatar_url": avatar}
	var channel Channel
	if err := c.do(http.MethodPost, "/channels", body, true, &channel); err != nil {
		return Channel{}, err
	}
	return channel, nil
}

// ListMessages returns all messages in the given channel.
func (c *HTTPClient) ListMessages(channelID string) ([]Message, error) {
	var messages []Message
	if err := c.do(http.MethodGet, "/channels/"+channelID+"/messages", nil, true, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// SendMessage posts a new message to the given channel and returns it.
func (c *HTTPClient) SendMessage(channelID, content string) (Message, error) {
	body := map[string]string{"content": content}
	var message Message
	if err := c.do(http.MethodPost, "/channels/"+channelID+"/messages", body, true, &message); err != nil {
		return Message{}, err
	}
	return message, nil
}

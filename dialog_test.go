package main

import (
	"strings"
	"testing"
)

// The dialog's job is to hand the backend one of the three types it accepts,
// with the values trimmed, and to catch the obvious mistake before sending.
func TestCleanChannelInput(t *testing.T) {
	kind, name, avatar, err := cleanChannelInput("group", "  general  ", " https://x/y.png ")
	if err != nil {
		t.Fatalf("a valid channel was refused: %v", err)
	}
	if kind != "group" {
		t.Errorf("type = %q, want group", kind)
	}
	if name != "general" {
		t.Errorf("name = %q, want the trimmed value", name)
	}
	if avatar != "https://x/y.png" {
		t.Errorf("avatar = %q, want the trimmed value", avatar)
	}

	if _, _, _, err := cleanChannelInput("group", "   ", ""); err == nil {
		t.Error("an empty name must be refused")
	}
	if _, _, _, err := cleanChannelInput("broadcast", "news", ""); err == nil {
		t.Error("a type the backend does not accept must be refused")
	}

	// Case and padding on the type are tolerated, since the picker is the
	// usual route but a caller might build the value another way.
	if k, _, _, err := cleanChannelInput(" P2P ", "ada", ""); err != nil || k != "p2p" {
		t.Errorf("cleanChannelInput(P2P) = %q, %v; want p2p, nil", k, err)
	}
}

// The picker must offer exactly the types the backend accepts, each with
// something to read. The order is a presentation choice — the most common
// kind is offered first — so only the set is checked.
func TestChannelTypesMatchTheBackend(t *testing.T) {
	want := map[string]bool{"p2p": true, "group": true, "channel": true}

	if len(channelTypes) != len(want) {
		t.Fatalf("%d channel types, want %d", len(channelTypes), len(want))
	}
	for _, ct := range channelTypes {
		if !want[ct.value] {
			t.Errorf("%q is not a type the backend accepts", ct.value)
		}
		delete(want, ct.value)

		if strings.TrimSpace(ct.label) == "" || strings.TrimSpace(ct.blurb) == "" {
			t.Errorf("%s is missing a label or a description", ct.value)
		}
	}
	for v := range want {
		t.Errorf("no button for channel type %q", v)
	}
}

// Demo mode serves what the desktop client expects, so a missing route there
// would show up as a blank screen rather than a crash.
func TestDemoServerServesTheClient(t *testing.T) {
	old := baseURL
	url, stop := startDemoServer()
	defer func() { baseURL = old; stop() }()
	baseURL = url

	c := NewHTTPClient()
	if _, err := c.Login("demo@example.com", "demo"); err != nil {
		t.Fatalf("demo login: %v", err)
	}
	if c.UserID == 0 || c.Name == "" {
		t.Errorf("demo user = (%d, %q), want an id and a name", c.UserID, c.Name)
	}

	channels, err := c.ListChannels()
	if err != nil {
		t.Fatalf("demo ListChannels: %v", err)
	}
	if len(channels) < 2 {
		t.Fatalf("demo returned %d channels, want at least 2", len(channels))
	}

	// The first channel has to have messages, otherwise the layout under
	// review is an empty screen.
	messages, err := c.ListMessages(channels[0].ID)
	if err != nil {
		t.Fatalf("demo ListMessages: %v", err)
	}
	if len(messages) < 3 {
		t.Fatalf("demo returned %d messages, want at least 3", len(messages))
	}

	sent, err := c.SendMessage(channels[0].ID, "hello from the demo")
	if err != nil {
		t.Fatalf("demo SendMessage: %v", err)
	}
	if sent.Content != "hello from the demo" {
		t.Errorf("sent content = %q", sent.Content)
	}

}

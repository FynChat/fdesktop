package main

import (
	"strings"
	"testing"
)

// The dialog's job is to hand the backend one of the types it accepts, with the
// values trimmed, and to catch the obvious mistake before sending.
func TestCleanChannelInput(t *testing.T) {
	kind, name, err := cleanChannelInput("group", "  general  ")
	if err != nil {
		t.Fatalf("a valid channel was refused: %v", err)
	}
	if kind != "group" {
		t.Errorf("type = %q, want group", kind)
	}
	if name != "general" {
		t.Errorf("name = %q, want the trimmed value", name)
	}

	if _, _, err := cleanChannelInput("group", "   "); err == nil {
		t.Error("an empty name must be refused")
	}
	if _, _, err := cleanChannelInput("broadcast", "news"); err == nil {
		t.Error("a type the backend does not accept must be refused")
	}

	// Case and padding on the type are tolerated, since the picker is the
	// usual route but a caller might build the value another way.
	if k, _, err := cleanChannelInput(" GROUP ", "news"); err != nil || k != "group" {
		t.Errorf("cleanChannelInput(GROUP) = %q, %v; want group, nil", k, err)
	}
}

// A private chat is not one of the types this dialog may send. It is get-or-
// create from a contact, so offering it here would send the backend a request
// that has no peer and cannot be satisfied.
func TestPrivateChatIsNotAChannelTypeHere(t *testing.T) {
	for _, ct := range channelTypes {
		if ct.value == "p2p" {
			t.Fatal("the new-channel dialog must not offer a private chat")
		}
	}

	if _, _, err := cleanChannelInput("p2p", "ada"); err == nil {
		t.Error("p2p must be refused, because a private chat is started from a contact")
	}
}

// The picker must offer exactly the types the backend accepts from this dialog,
// each with something to read.
func TestChannelTypesMatchTheBackend(t *testing.T) {
	want := map[string]bool{"group": true, "channel": true}

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

// A handle is what identifies an account, so the form has to accept the
// spellings people actually type and reject what cannot be a handle at all.
func TestCleanContactHandle(t *testing.T) {
	// The @ is the part worth handling: people type it from habit, and a
	// handle with and without it must reach the backend as one value.
	if got, err := cleanContactHandle("  @ada  "); err != nil || got != "ada" {
		t.Errorf("cleanContactHandle(@ada) = %q, %v; want ada, nil", got, err)
	}
	if got, err := cleanContactHandle("ada"); err != nil || got != "ada" {
		t.Errorf("cleanContactHandle(ada) = %q, %v; want ada, nil", got, err)
	}

	// A bare "@" is not somebody's name.
	if _, err := cleanContactHandle("@"); err == nil {
		t.Error("@ alone must be refused")
	}
	if _, err := cleanContactHandle("   "); err == nil {
		t.Error("an empty handle must be refused")
	}
	if _, err := cleanContactHandle("ada lovelace"); err == nil {
		t.Error("a handle with a space in it must be refused")
	}
}

// Searching contacts must match on the name and the handle, and an empty
// search must show everything rather than nothing.
func TestContactFilter(t *testing.T) {
	contacts := []Contact{
		{ID: 1, Name: "Ada Lovelace", Username: "ada"},
		{ID: 2, Name: "Grace Hopper", Username: "amazing"},
		{ID: 3, Name: "Alan Turing", Username: "turing"},
	}

	if got := contactFilter(contacts, ""); len(got) != len(contacts) {
		t.Errorf("an empty search left %d of %d contacts", len(got), len(contacts))
	}
	if got := contactFilter(contacts, "   "); len(got) != len(contacts) {
		t.Errorf("a blank search left %d of %d contacts", len(got), len(contacts))
	}

	if got := contactFilter(contacts, "ada"); len(got) != 1 || got[0].ID != 1 {
		t.Errorf("searching the name matched %d contacts, want just Ada", len(got))
	}
	if got := contactFilter(contacts, "AMAZING"); len(got) != 1 || got[0].ID != 2 {
		t.Errorf("searching the handle matched %d contacts, want just Grace", len(got))
	}
	if got := contactFilter(contacts, "a"); len(got) != 3 {
		t.Errorf("a one-letter search matched %d contacts, want all 3", len(got))
	}
	if got := contactFilter(contacts, "nobody"); len(got) != 0 {
		t.Errorf("a search with no matches left %d contacts, want none", len(got))
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

	// The private-chat path: a contact is chosen, and the backend hands back
	// the chat with them — creating it the first time and returning the same
	// one afterwards.
	contacts, err := c.ListContacts()
	if err != nil {
		t.Fatalf("demo ListContacts: %v", err)
	}
	if len(contacts) == 0 {
		t.Fatal("demo returned no contacts, so the contacts screen would be empty")
	}

	first, err := c.OpenPrivateChat(contacts[0].ID)
	if err != nil {
		t.Fatalf("demo OpenPrivateChat: %v", err)
	}
	if first.ChannelName != contacts[0].Name {
		t.Errorf("private channel = %q, want it named after %q", first.ChannelName, contacts[0].Name)
	}

	// Asking again must not make a second chat with the same person.
	again, err := c.OpenPrivateChat(contacts[0].ID)
	if err != nil {
		t.Fatalf("demo OpenPrivateChat (again): %v", err)
	}
	if again.ID != first.ID {
		t.Errorf("second call returned channel %q, want the first one, %q", again.ID, first.ID)
	}

	// A new contact is added by handle, and the name comes back from the
	// backend rather than being invented here.
	added, err := c.AddContact("newbie")
	if err != nil {
		t.Fatalf("demo AddContact: %v", err)
	}
	if added.Username != "newbie" || added.Name == "" || added.ID == 0 {
		t.Errorf("added contact = %+v, want a handle and a name from the backend", added)
	}

	// Adding the same person twice is a mistake worth reporting, not a
	// duplicate contact.
	if _, err := c.AddContact("newbie"); err == nil {
		t.Error("adding the same handle twice must fail")
	}
}

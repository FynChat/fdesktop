package main

import "testing"

// The forms must catch the obvious mistakes locally: a round trip to the
// backend for a missing email is wasted time and a confusing error.
func TestEmailValidator(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{"ada@example.com", true},
		{"ADA@EXAMPLE.COM", true},
		{"", true}, // empty is caught by the form validators instead
		{"ada", false},
		{"@example.com", false},
		{"ada@", false},
		{"ada example@x.com", false},
	}

	for _, c := range cases {
		err := emailValidator(c.input)
		if (err == nil) != c.valid {
			t.Errorf("emailValidator(%q) = %v, want valid=%v", c.input, err, c.valid)
		}
	}
}

func TestValidateLogin(t *testing.T) {
	if err := validateLogin("ada@example.com", "hunter22"); err != nil {
		t.Errorf("valid credentials rejected: %v", err)
	}
	if err := validateLogin("", "hunter22"); err == nil {
		t.Error("empty email accepted")
	}
	if err := validateLogin("ada@example.com", ""); err == nil {
		t.Error("empty password accepted")
	}
	if err := validateLogin("not-an-email", "hunter22"); err == nil {
		t.Error("malformed email accepted")
	}
}

func TestValidateRegistration(t *testing.T) {
	if err := validateRegistration("Ada", "ada", "ada@example.com", "hunter22"); err != nil {
		t.Errorf("valid registration rejected: %v", err)
	}

	cases := map[string]struct {
		name, username, email, password string
	}{
		"no name":            {"", "ada", "ada@example.com", "hunter22"},
		"no username":        {"Ada", "", "ada@example.com", "hunter22"},
		"no email":           {"Ada", "ada", "", "hunter22"},
		"bad email":          {"Ada", "ada", "ada@", "hunter22"},
		"short password":     {"Ada", "ada", "ada@example.com", "short"},
		"password exactly 7": {"Ada", "ada", "ada@example.com", "1234567"},
	}
	for name, c := range cases {
		if err := validateRegistration(c.name, c.username, c.email, c.password); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}

	// Exactly the minimum must pass.
	if err := validateRegistration("Ada", "ada", "ada@example.com", "12345678"); err != nil {
		t.Errorf("8-character password rejected: %v", err)
	}
}

package main

import (
	"testing"
)

func TestStripBotMention(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"mentionAtStart", "<@U12345> hello", "hello"},
		{"mentionInMiddle", "hey <@U12345> do something", "hey  do something"},
		{"multipleMentions", "<@U12345> <@U67890> hi", "hi"},
		{"noMention", "hello world", "hello world"},
		{"emptyAfterMention", "<@U12345>", ""},
		{"mentionWithSpaces", "  <@U12345>  hello  ", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripBotMention(tt.input)
			if got != tt.want {
				t.Errorf("stripBotMention(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplitMessage(t *testing.T) {
	t.Run("shortMessage", func(t *testing.T) {
		chunks := splitMessage("hello", 3000)
		if len(chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0] != "hello" {
			t.Errorf("chunk = %q, want %q", chunks[0], "hello")
		}
	})

	t.Run("longMessageSplitsAtNewline", func(t *testing.T) {
		msg := "line 1\nline 2\nline 3"
		chunks := splitMessage(msg, 10)
		if len(chunks) < 2 {
			t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
		}
	})

	t.Run("longMessageNoNewline", func(t *testing.T) {
		msg := "aaaaaaaaaa"
		chunks := splitMessage(msg, 5)
		if len(chunks) != 2 {
			t.Fatalf("expected 2 chunks, got %d", len(chunks))
		}
	})
}

func TestExtractSessionIDFromOutput(t *testing.T) {
	t.Run("validOutput", func(t *testing.T) {
		output := `{"sessionID":"sess-abc-123"}
{"type":"message","content":"hello"}
`
		got := extractSessionIDFromOutput(output)
		if got != "sess-abc-123" {
			t.Errorf("extractSessionIDFromOutput = %q, want %q", got, "sess-abc-123")
		}
	})

	t.Run("noSessionID", func(t *testing.T) {
		output := `{"type":"message","content":"hello"}
`
		got := extractSessionIDFromOutput(output)
		if got != "" {
			t.Errorf("extractSessionIDFromOutput = %q, want empty", got)
		}
	})

	t.Run("emptyOutput", func(t *testing.T) {
		got := extractSessionIDFromOutput("")
		if got != "" {
			t.Errorf("extractSessionIDFromOutput = %q, want empty", got)
		}
	})

	t.Run("invalidJSON", func(t *testing.T) {
		got := extractSessionIDFromOutput("not json at all")
		if got != "" {
			t.Errorf("extractSessionIDFromOutput = %q, want empty", got)
		}
	})
}

func TestParseSlackBotConfig(t *testing.T) {
	t.Run("missingBotToken", func(t *testing.T) {
		t.Setenv("SLACK_BOT_TOKEN", "")
		t.Setenv("SLACK_APP_TOKEN", "xapp-test")
		t.Setenv("SLACK_ALLOWED_USERS", "U123")

		_, err := parseSlackBotConfig("/tmp")
		if err == nil {
			t.Error("expected error for missing SLACK_BOT_TOKEN")
		}
	})

	t.Run("missingAppToken", func(t *testing.T) {
		t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")
		t.Setenv("SLACK_APP_TOKEN", "")
		t.Setenv("SLACK_ALLOWED_USERS", "U123")

		_, err := parseSlackBotConfig("/tmp")
		if err == nil {
			t.Error("expected error for missing SLACK_APP_TOKEN")
		}
	})

	t.Run("missingAllowedUsers", func(t *testing.T) {
		t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")
		t.Setenv("SLACK_APP_TOKEN", "xapp-test")
		t.Setenv("SLACK_ALLOWED_USERS", "")

		_, err := parseSlackBotConfig("/tmp")
		if err == nil {
			t.Error("expected error for missing SLACK_ALLOWED_USERS")
		}
	})

	t.Run("validConfig", func(t *testing.T) {
		t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")
		t.Setenv("SLACK_APP_TOKEN", "xapp-test")
		t.Setenv("SLACK_ALLOWED_USERS", "U123,U456, U789")

		cfg, err := parseSlackBotConfig("/tmp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.botToken != "xoxb-test" {
			t.Errorf("botToken = %q, want %q", cfg.botToken, "xoxb-test")
		}
		if cfg.appToken != "xapp-test" {
			t.Errorf("appToken = %q, want %q", cfg.appToken, "xapp-test")
		}
		if !cfg.allowedUsers["U123"] {
			t.Error("U123 should be allowed")
		}
		if !cfg.allowedUsers["U456"] {
			t.Error("U456 should be allowed")
		}
		if !cfg.allowedUsers["U789"] {
			t.Error("U789 should be allowed")
		}
		if cfg.allowedUsers["U000"] {
			t.Error("U000 should not be allowed")
		}
		if cfg.rootDir != "/tmp" {
			t.Errorf("rootDir = %q, want %q", cfg.rootDir, "/tmp")
		}
	})
}

func TestAllowlist(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")
	t.Setenv("SLACK_APP_TOKEN", "xapp-test")
	t.Setenv("SLACK_ALLOWED_USERS", "U123,U456")

	cfg, err := parseSlackBotConfig("/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bot := &slackBot{config: cfg}

	if !bot.isAllowed("U123") {
		t.Error("U123 should be allowed")
	}
	if !bot.isAllowed("U456") {
		t.Error("U456 should be allowed")
	}
	if bot.isAllowed("U789") {
		t.Error("U789 should be denied (not in allowlist)")
	}
	if bot.isAllowed("") {
		t.Error("empty user should be denied")
	}
}

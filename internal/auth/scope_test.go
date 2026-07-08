package auth

import "testing"

func TestMatchScope_ExactMatch(t *testing.T) {
	if !MatchScope("gpt-4", "gpt-4") {
		t.Error("expected exact match to return true")
	}
}

func TestMatchScope_ExactMismatch(t *testing.T) {
	if MatchScope("gpt-4", "gpt-3.5") {
		t.Error("expected mismatched model to return false")
	}
}

func TestMatchScope_Wildcard(t *testing.T) {
	if !MatchScope("*", "gpt-4") {
		t.Error("expected wildcard to match any model")
	}
}

func TestMatchScope_MultipleScopes_MatchSecond(t *testing.T) {
	if !MatchScope("gpt-3.5,gpt-4,gpt-4-turbo", "gpt-4") {
		t.Error("expected second scope to match")
	}
}

func TestMatchScope_MultipleScopes_NoMatch(t *testing.T) {
	if MatchScope("gpt-3.5,gpt-4", "claude-3") {
		t.Error("expected no match")
	}
}

func TestMatchScope_EmptyScopes(t *testing.T) {
	if MatchScope("", "gpt-4") {
		t.Error("expected empty scopes to not match")
	}
}

func TestMatchScope_WhitespaceInScopes(t *testing.T) {
	if !MatchScope("gpt-3.5, gpt-4", "gpt-4") {
		t.Error("expected scope with whitespace to match")
	}
}

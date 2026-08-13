package service

import "testing"

func TestValidateFilmArtifactPayloadKeepsActingInItsDomain(t *testing.T) {
	if err := validateFilmArtifactPayload(FilmArtifactTypeActing, map[string]any{
		"objective": "conceal mistake",
		"obstacle":  "XiaoPi questions him",
		"tactic":    "dismiss then redirect",
		"beat":      "4",
	}); err != nil {
		t.Fatal(err)
	}
	if err := validateFilmArtifactPayload(FilmArtifactTypeActing, map[string]any{"camera": "slow push"}); err == nil {
		t.Fatal("acting must not be allowed to override camera facts")
	}
	if err := validateFilmArtifactPayload("unknown", map[string]any{}); err == nil {
		t.Fatal("unknown artifact types must be rejected")
	}
}

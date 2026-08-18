package service

import (
	"strings"
	"testing"
)

func TestSanitizeAPICallPayloadRedactsNestedCredentials(t *testing.T) {
	raw := []byte(`{"prompt":"keep the word token in this prompt","headers":[{"name":"Authorization","value":"Bearer provider-secret"}],"token":"session-secret","credential":"gateway-secret","key":"access-secret","image_url":"https://example.com/image.png?signature=real-signature&x=1"}`)
	got := SanitizeAPICallPayload(raw, "application/json")
	for _, secret := range []string{"provider-secret", "session-secret", "gateway-secret", "access-secret", "real-signature"} {
		if strings.Contains(got, secret) {
			t.Fatalf("sanitized payload contains %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "keep the word token in this prompt") {
		t.Fatalf("sanitizer removed a non-credential prompt value: %s", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("sanitized payload did not include a redaction marker: %s", got)
	}
}

func TestSanitizeAPICallPayloadPreservesSafeFields(t *testing.T) {
	raw := []byte(`{"model":"gpt-image-2","size":"1024x1024","n":1,"metadata":{"shotId":"shot-1"}}`)
	got := SanitizeAPICallPayload(raw, "application/json")
	for _, value := range []string{"gpt-image-2", "1024x1024", "shot-1"} {
		if !strings.Contains(got, value) {
			t.Fatalf("safe value %q was lost: %s", value, got)
		}
	}
}

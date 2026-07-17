package envelope

import (
	"strings"
	"testing"
)

func TestParseSuccessEnvelope(t *testing.T) {
	raw := []byte(`{"spec":{"type":"bar","width":10,"height":5,"data":{"series":[{"name":"a","values":[{"y":1}]}]}},"warnings":[],"size":{"width":10,"height":5}}`)
	s, warns, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.Type != "bar" || s.Width != 10 {
		t.Fatalf("spec mismatch: %+v", s)
	}
	if warns == nil {
		t.Fatal("warnings must be non-nil")
	}
}

func TestParseErrorEnvelope(t *testing.T) {
	_, _, err := Parse([]byte(`{"error":{"message":"Unknown chart type \"X\""}}`))
	if err == nil || !strings.Contains(err.Error(), "Unknown chart type") {
		t.Fatalf("error envelope must surface the message, got %v", err)
	}
}

func TestParseGarbage(t *testing.T) {
	if _, _, err := Parse([]byte(`nope`)); err == nil {
		t.Fatal("expected error for non-JSON")
	}
}

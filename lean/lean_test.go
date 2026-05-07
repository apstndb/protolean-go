package lean

import (
	"strings"
	"testing"
)

func TestEncodePrimitives(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want string
	}{
		{"true", true, "T"},
		{"false", false, "F"},
		{"null", nil, "_"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"string", "hello", `"hello"`}, // root strings always quoted
		{"empty string", "", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Encode(tt.v)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("Encode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEncodeObject(t *testing.T) {
	v := map[string]any{
		"name":   "Alice",
		"age":    30,
		"active": true,
	}

	got, err := Encode(v)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("LEAN:\n%s", got)

	if !strings.Contains(got, "name:") {
		t.Errorf("expected name field")
	}
	if !strings.Contains(got, "age:30") {
		t.Errorf("expected age field without space")
	}
	if !strings.Contains(got, "active:T") {
		t.Errorf("expected active as T")
	}
}

func TestEncodeTabular(t *testing.T) {
	v := map[string]any{
		"users": []any{
			map[string]any{"id": 1, "name": "Alice", "active": true},
			map[string]any{"id": 2, "name": "Bob", "active": false},
		},
	}

	got, err := Encode(v)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("LEAN:\n%s", got)

	if !strings.Contains(got, "users[2]:") {
		t.Errorf("expected tabular header")
	}
	if !strings.Contains(got, "\t") {
		t.Errorf("expected tab delimiter")
	}
}

func TestEncodeFlatArray(t *testing.T) {
	v := map[string]any{
		"scores": []any{95, 87, 42},
	}

	got, err := Encode(v)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("LEAN:\n%s", got)

	if !strings.Contains(got, "scores[3]:95\t87\t42") {
		t.Errorf("expected flat array, got: %q", got)
	}
}

func TestEncodeDotFlatten(t *testing.T) {
	v := map[string]any{
		"meta": map[string]any{
			"version": "2.1.0",
			"debug":   false,
		},
	}

	got, err := Encode(v)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("LEAN:\n%s", got)

	if !strings.Contains(got, "meta.version:2.1.0") {
		t.Errorf("expected dot-flattened key, got:\n%s", got)
	}
}

func TestRoundTrip(t *testing.T) {
	original := map[string]any{
		"meta": map[string]any{
			"version": "2.1.0",
			"debug":   false,
		},
		"users": []any{
			map[string]any{"id": 1, "name": "Alice", "email": "alice@ex.com", "active": true},
			map[string]any{"id": 2, "name": "Bob", "email": "bob@ex.com", "active": false},
		},
		"tags":  []any{},
		"notes": []any{1, "hello", map[string]any{"key": "val"}},
	}

	enc, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	t.Logf("LEAN:\n%s", enc)
}

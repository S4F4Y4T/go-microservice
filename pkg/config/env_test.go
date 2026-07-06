package config

import (
	"testing"
	"time"
)

func TestGetEnvInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	if got := GetEnvInt("TEST_INT", 0); got != 42 {
		t.Errorf("GetEnvInt() = %d, want 42", got)
	}
}

func TestGetEnvIntMissing(t *testing.T) {
	if got := GetEnvInt("TEST_INT_MISSING", 7); got != 7 {
		t.Errorf("GetEnvInt() = %d, want 7 (default)", got)
	}
}

func TestGetEnvIntInvalid(t *testing.T) {
	t.Setenv("TEST_INT_INVALID", "not-a-number")
	if got := GetEnvInt("TEST_INT_INVALID", 7); got != 7 {
		t.Errorf("GetEnvInt() = %d, want 7 (default on parse error)", got)
	}
}

func TestGetEnvDuration(t *testing.T) {
	t.Setenv("TEST_DURATION", "5s")
	if got := GetEnvDuration("TEST_DURATION", time.Second); got != 5*time.Second {
		t.Errorf("GetEnvDuration() = %v, want 5s", got)
	}
}

func TestGetEnvDurationInvalid(t *testing.T) {
	t.Setenv("TEST_DURATION_INVALID", "not-a-duration")
	if got := GetEnvDuration("TEST_DURATION_INVALID", time.Minute); got != time.Minute {
		t.Errorf("GetEnvDuration() = %v, want default 1m", got)
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("TEST_BOOL", tt.value)
			if got := GetEnvBool("TEST_BOOL", !tt.want); got != tt.want {
				t.Errorf("GetEnvBool(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestGetEnvBoolInvalid(t *testing.T) {
	t.Setenv("TEST_BOOL_INVALID", "not-a-bool")
	if got := GetEnvBool("TEST_BOOL_INVALID", true); got != true {
		t.Errorf("GetEnvBool() = %v, want default true", got)
	}
}

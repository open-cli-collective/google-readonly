package testutil

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// Equal checks that got equals want using comparable constraint.
func Equal[T comparable](t testing.TB, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// NoError fails the test immediately if err is not nil.
func NoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Error checks that err is not nil.
func Error(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ErrorIs checks that err matches target using errors.Is.
func ErrorIs(t testing.TB, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("got error %v, want error matching %v", err, target)
	}
}

// Contains checks that s contains substr.
func Contains(t testing.TB, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

// NotContains checks that s does not contain substr.
func NotContains(t testing.TB, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected %q to not contain %q", s, substr)
	}
}

// Len checks that the slice has the expected length.
func Len[T any](t testing.TB, slice []T, want int) {
	t.Helper()
	if len(slice) != want {
		t.Errorf("got length %d, want %d", len(slice), want)
	}
}

// Nil checks that val is nil.
// Uses reflection to handle nil slices, maps, pointers, channels, and functions
// that appear non-nil when boxed into an interface.
func Nil(t testing.TB, val any) {
	t.Helper()
	if val == nil {
		return
	}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		if v.IsNil() {
			return
		}
	}
	t.Errorf("got %v, want nil", val)
}

// NotNil fails the test immediately if val is nil.
func NotNil(t testing.TB, val any) {
	t.Helper()
	if val == nil {
		t.Fatal("got nil, want non-nil")
	}
}

// True checks that condition is true.
func True(t testing.TB, condition bool) {
	t.Helper()
	if !condition {
		t.Error("got false, want true")
	}
}

// False checks that condition is false.
func False(t testing.TB, condition bool) {
	t.Helper()
	if condition {
		t.Error("got true, want false")
	}
}

// Empty checks that s is the empty string.
func Empty(t testing.TB, s string) {
	t.Helper()
	if s != "" {
		t.Errorf("got %q, want empty string", s)
	}
}

// NotEmpty checks that s is not the empty string.
func NotEmpty(t testing.TB, s string) {
	t.Helper()
	if s == "" {
		t.Error("got empty string, want non-empty")
	}
}

// Greater checks that a > b.
func Greater(t testing.TB, a, b int) {
	t.Helper()
	if a <= b {
		t.Errorf("got %d, want greater than %d", a, b)
	}
}

// GreaterOrEqual checks that a >= b.
func GreaterOrEqual(t testing.TB, a, b int) {
	t.Helper()
	if a < b {
		t.Errorf("got %d, want >= %d", a, b)
	}
}

// Less checks that a < b.
func Less(t testing.TB, a, b int) {
	t.Helper()
	if a >= b {
		t.Errorf("got %d, want less than %d", a, b)
	}
}

// SliceContains checks that the slice contains the target value.
func SliceContains[T comparable](t testing.TB, slice []T, target T) {
	t.Helper()
	for _, v := range slice {
		if v == target {
			return
		}
	}
	t.Errorf("slice %v does not contain %v", slice, target)
}

// LenSlice checks that an arbitrary slice has the expected length.
// Use this when Len's type parameter cannot be inferred.
func LenSlice(t testing.TB, length, want int) {
	t.Helper()
	if length != want {
		t.Errorf("got length %d, want %d", length, want)
	}
}

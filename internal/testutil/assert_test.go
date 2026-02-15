package testutil

import (
	"errors"
	"testing"
)

// mockT captures test failures without stopping the outer test.
type mockT struct {
	testing.TB
	failed  bool
	message string
}

func (m *mockT) Helper()                        {}
func (m *mockT) Errorf(format string, a ...any) { m.failed = true }
func (m *mockT) Error(a ...any)                 { m.failed = true }
func (m *mockT) Fatalf(format string, a ...any) { m.failed = true }
func (m *mockT) Fatal(a ...any)                 { m.failed = true }

func TestEqual(t *testing.T) {
	t.Run("passes on equal values", func(t *testing.T) {
		mt := &mockT{}
		Equal(mt, 42, 42)
		if mt.failed {
			t.Error("Equal should not fail for equal values")
		}
	})

	t.Run("fails on unequal values", func(t *testing.T) {
		mt := &mockT{}
		Equal(mt, 1, 2)
		if !mt.failed {
			t.Error("Equal should fail for unequal values")
		}
	})

	t.Run("works with strings", func(t *testing.T) {
		mt := &mockT{}
		Equal(mt, "hello", "hello")
		if mt.failed {
			t.Error("Equal should not fail for equal strings")
		}
	})
}

func TestNoError(t *testing.T) {
	t.Run("passes on nil error", func(t *testing.T) {
		mt := &mockT{}
		NoError(mt, nil)
		if mt.failed {
			t.Error("NoError should not fail for nil error")
		}
	})

	t.Run("fails on non-nil error", func(t *testing.T) {
		mt := &mockT{}
		NoError(mt, errors.New("boom"))
		if !mt.failed {
			t.Error("NoError should fail for non-nil error")
		}
	})
}

func TestError(t *testing.T) {
	t.Run("passes on non-nil error", func(t *testing.T) {
		mt := &mockT{}
		Error(mt, errors.New("boom"))
		if mt.failed {
			t.Error("Error should not fail for non-nil error")
		}
	})

	t.Run("fails on nil error", func(t *testing.T) {
		mt := &mockT{}
		Error(mt, nil)
		if !mt.failed {
			t.Error("Error should fail for nil error")
		}
	})
}

func TestErrorIs(t *testing.T) {
	sentinel := errors.New("sentinel")

	t.Run("passes when errors match", func(t *testing.T) {
		mt := &mockT{}
		ErrorIs(mt, sentinel, sentinel)
		if mt.failed {
			t.Error("ErrorIs should not fail for matching errors")
		}
	})

	t.Run("fails when errors don't match", func(t *testing.T) {
		mt := &mockT{}
		ErrorIs(mt, errors.New("other"), sentinel)
		if !mt.failed {
			t.Error("ErrorIs should fail for non-matching errors")
		}
	})
}

func TestContains(t *testing.T) {
	t.Run("passes when string contains substr", func(t *testing.T) {
		mt := &mockT{}
		Contains(mt, "hello world", "world")
		if mt.failed {
			t.Error("Contains should not fail when substr is present")
		}
	})

	t.Run("fails when string doesn't contain substr", func(t *testing.T) {
		mt := &mockT{}
		Contains(mt, "hello world", "xyz")
		if !mt.failed {
			t.Error("Contains should fail when substr is absent")
		}
	})
}

func TestNotContains(t *testing.T) {
	t.Run("passes when string doesn't contain substr", func(t *testing.T) {
		mt := &mockT{}
		NotContains(mt, "hello world", "xyz")
		if mt.failed {
			t.Error("NotContains should not fail when substr is absent")
		}
	})

	t.Run("fails when string contains substr", func(t *testing.T) {
		mt := &mockT{}
		NotContains(mt, "hello world", "world")
		if !mt.failed {
			t.Error("NotContains should fail when substr is present")
		}
	})
}

func TestLen(t *testing.T) {
	t.Run("passes on correct length", func(t *testing.T) {
		mt := &mockT{}
		Len(mt, []int{1, 2, 3}, 3)
		if mt.failed {
			t.Error("Len should not fail for correct length")
		}
	})

	t.Run("fails on wrong length", func(t *testing.T) {
		mt := &mockT{}
		Len(mt, []int{1, 2}, 3)
		if !mt.failed {
			t.Error("Len should fail for wrong length")
		}
	})

	t.Run("works with empty slice", func(t *testing.T) {
		mt := &mockT{}
		Len(mt, []string{}, 0)
		if mt.failed {
			t.Error("Len should not fail for empty slice with want 0")
		}
	})
}

func TestNil(t *testing.T) {
	t.Run("passes on nil", func(t *testing.T) {
		mt := &mockT{}
		Nil(mt, nil)
		if mt.failed {
			t.Error("Nil should not fail for nil")
		}
	})

	t.Run("fails on non-nil", func(t *testing.T) {
		mt := &mockT{}
		Nil(mt, "something")
		if !mt.failed {
			t.Error("Nil should fail for non-nil")
		}
	})
}

func TestNotNil(t *testing.T) {
	t.Run("passes on non-nil", func(t *testing.T) {
		mt := &mockT{}
		NotNil(mt, "something")
		if mt.failed {
			t.Error("NotNil should not fail for non-nil")
		}
	})

	t.Run("fails on nil", func(t *testing.T) {
		mt := &mockT{}
		NotNil(mt, nil)
		if !mt.failed {
			t.Error("NotNil should fail for nil")
		}
	})
}

func TestTrue(t *testing.T) {
	t.Run("passes on true", func(t *testing.T) {
		mt := &mockT{}
		True(mt, true)
		if mt.failed {
			t.Error("True should not fail for true")
		}
	})

	t.Run("fails on false", func(t *testing.T) {
		mt := &mockT{}
		True(mt, false)
		if !mt.failed {
			t.Error("True should fail for false")
		}
	})
}

func TestFalse(t *testing.T) {
	t.Run("passes on false", func(t *testing.T) {
		mt := &mockT{}
		False(mt, false)
		if mt.failed {
			t.Error("False should not fail for false")
		}
	})

	t.Run("fails on true", func(t *testing.T) {
		mt := &mockT{}
		False(mt, true)
		if !mt.failed {
			t.Error("False should fail for true")
		}
	})
}

func TestEmpty(t *testing.T) {
	t.Run("passes on empty string", func(t *testing.T) {
		mt := &mockT{}
		Empty(mt, "")
		if mt.failed {
			t.Error("Empty should not fail for empty string")
		}
	})

	t.Run("fails on non-empty string", func(t *testing.T) {
		mt := &mockT{}
		Empty(mt, "hello")
		if !mt.failed {
			t.Error("Empty should fail for non-empty string")
		}
	})
}

func TestNotEmpty(t *testing.T) {
	t.Run("passes on non-empty string", func(t *testing.T) {
		mt := &mockT{}
		NotEmpty(mt, "hello")
		if mt.failed {
			t.Error("NotEmpty should not fail for non-empty string")
		}
	})

	t.Run("fails on empty string", func(t *testing.T) {
		mt := &mockT{}
		NotEmpty(mt, "")
		if !mt.failed {
			t.Error("NotEmpty should fail for empty string")
		}
	})
}

func TestGreater(t *testing.T) {
	t.Run("passes when a > b", func(t *testing.T) {
		mt := &mockT{}
		Greater(mt, 5, 3)
		if mt.failed {
			t.Error("Greater should not fail when a > b")
		}
	})

	t.Run("fails when a == b", func(t *testing.T) {
		mt := &mockT{}
		Greater(mt, 3, 3)
		if !mt.failed {
			t.Error("Greater should fail when a == b")
		}
	})

	t.Run("fails when a < b", func(t *testing.T) {
		mt := &mockT{}
		Greater(mt, 2, 3)
		if !mt.failed {
			t.Error("Greater should fail when a < b")
		}
	})
}

func TestGreaterOrEqual(t *testing.T) {
	t.Run("passes when a > b", func(t *testing.T) {
		mt := &mockT{}
		GreaterOrEqual(mt, 5, 3)
		if mt.failed {
			t.Error("GreaterOrEqual should not fail when a > b")
		}
	})

	t.Run("passes when a == b", func(t *testing.T) {
		mt := &mockT{}
		GreaterOrEqual(mt, 3, 3)
		if mt.failed {
			t.Error("GreaterOrEqual should not fail when a == b")
		}
	})

	t.Run("fails when a < b", func(t *testing.T) {
		mt := &mockT{}
		GreaterOrEqual(mt, 2, 3)
		if !mt.failed {
			t.Error("GreaterOrEqual should fail when a < b")
		}
	})
}

func TestLess(t *testing.T) {
	t.Run("passes when a < b", func(t *testing.T) {
		mt := &mockT{}
		Less(mt, 2, 5)
		if mt.failed {
			t.Error("Less should not fail when a < b")
		}
	})

	t.Run("fails when a == b", func(t *testing.T) {
		mt := &mockT{}
		Less(mt, 3, 3)
		if !mt.failed {
			t.Error("Less should fail when a == b")
		}
	})

	t.Run("fails when a > b", func(t *testing.T) {
		mt := &mockT{}
		Less(mt, 5, 3)
		if !mt.failed {
			t.Error("Less should fail when a > b")
		}
	})
}

package focus_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/components/focus"
)

func TestFocusManager_cycles(t *testing.T) {
	fm := focus.New(3) // 3 panels

	if fm.Current() != 0 {
		t.Errorf("initial focus = %d, want 0", fm.Current())
	}

	fm.Next()
	if fm.Current() != 1 {
		t.Errorf("after Next = %d, want 1", fm.Current())
	}

	fm.Next()
	fm.Next()
	if fm.Current() != 0 {
		t.Errorf("after 3x Next = %d, want 0 (wrapped)", fm.Current())
	}
}

func TestFocusManager_prev(t *testing.T) {
	fm := focus.New(3)

	fm.Prev()
	if fm.Current() != 2 {
		t.Errorf("Prev from 0 = %d, want 2 (wrapped)", fm.Current())
	}
}

func TestFocusManager_isFocused(t *testing.T) {
	fm := focus.New(3)

	if !fm.IsFocused(0) {
		t.Error("expected index 0 to be focused initially")
	}
	if fm.IsFocused(1) {
		t.Error("expected index 1 to not be focused initially")
	}

	fm.Next()
	if !fm.IsFocused(1) {
		t.Error("expected index 1 to be focused after Next")
	}
	if fm.IsFocused(0) {
		t.Error("expected index 0 to not be focused after Next")
	}
}

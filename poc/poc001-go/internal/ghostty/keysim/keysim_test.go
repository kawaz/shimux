package keysim

import (
	"fmt"
	"testing"

	"github.com/kawaz/shimux/internal/ghostty"
)

// TestKeySimControllerImplementsController はコンパイル時にインターフェース適合を検証する。
func TestKeySimControllerImplementsController(t *testing.T) {
	var _ ghostty.Controller = &KeySimController{}
}

// mockExecutor はosascriptの代わりに呼び出しを記録するヘルパー。
type mockExecutor struct {
	scripts []string
	err     error
}

func (m *mockExecutor) exec(script string) error {
	m.scripts = append(m.scripts, script)
	return m.err
}

func TestNewWindow(t *testing.T) {
	mock := &mockExecutor{}
	ctrl := NewWithExecutor(mock.exec)

	err := ctrl.NewWindow()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.scripts) != 1 {
		t.Fatalf("expected 1 script call, got %d", len(mock.scripts))
	}
	want := `tell application "System Events" to tell process "ghostty" to keystroke "n" using {command down}`
	if mock.scripts[0] != want {
		t.Errorf("script =\n  %q\nwant\n  %q", mock.scripts[0], want)
	}
}

func TestNewTab(t *testing.T) {
	mock := &mockExecutor{}
	ctrl := NewWithExecutor(mock.exec)

	err := ctrl.NewTab()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.scripts) != 1 {
		t.Fatalf("expected 1 script call, got %d", len(mock.scripts))
	}
	want := `tell application "System Events" to tell process "ghostty" to keystroke "t" using {command down}`
	if mock.scripts[0] != want {
		t.Errorf("script =\n  %q\nwant\n  %q", mock.scripts[0], want)
	}
}

func TestNewSplit(t *testing.T) {
	tests := []struct {
		name      string
		direction ghostty.SplitDirection
		want      string
		wantErr   bool
	}{
		{
			name:      "SplitRight generates super+d",
			direction: ghostty.SplitRight,
			want:      `tell application "System Events" to tell process "ghostty" to keystroke "d" using {command down}`,
		},
		{
			name:      "SplitDown generates super+shift+d",
			direction: ghostty.SplitDown,
			want:      `tell application "System Events" to tell process "ghostty" to keystroke "D" using {command down, shift down}`,
		},
		{
			name:      "SplitLeft is unsupported",
			direction: ghostty.SplitLeft,
			wantErr:   true,
		},
		{
			name:      "SplitUp is unsupported",
			direction: ghostty.SplitUp,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecutor{}
			ctrl := NewWithExecutor(mock.exec)

			err := ctrl.NewSplit(tt.direction)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				// osascript should not be called for unsupported directions
				if len(mock.scripts) != 0 {
					t.Errorf("expected no script calls for unsupported direction, got %d", len(mock.scripts))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(mock.scripts) != 1 {
				t.Fatalf("expected 1 script call, got %d", len(mock.scripts))
			}
			if mock.scripts[0] != tt.want {
				t.Errorf("script =\n  %q\nwant\n  %q", mock.scripts[0], tt.want)
			}
		})
	}
}

func TestGotoSplit(t *testing.T) {
	tests := []struct {
		name      string
		direction ghostty.GotoDirection
		want      string
	}{
		{
			name:      "GotoNext generates super+]",
			direction: ghostty.GotoNext,
			want:      `tell application "System Events" to tell process "ghostty" to keystroke "]" using {command down}`,
		},
		{
			name:      "GotoPrevious generates super+[",
			direction: ghostty.GotoPrevious,
			want:      `tell application "System Events" to tell process "ghostty" to keystroke "[" using {command down}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecutor{}
			ctrl := NewWithExecutor(mock.exec)

			err := ctrl.GotoSplit(tt.direction)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(mock.scripts) != 1 {
				t.Fatalf("expected 1 script call, got %d", len(mock.scripts))
			}
			if mock.scripts[0] != tt.want {
				t.Errorf("script =\n  %q\nwant\n  %q", mock.scripts[0], tt.want)
			}
		})
	}
}

func TestGotoSplitInvalidDirection(t *testing.T) {
	mock := &mockExecutor{}
	ctrl := NewWithExecutor(mock.exec)

	// Use an invalid GotoDirection value
	err := ctrl.GotoSplit(ghostty.GotoDirection(99))
	if err == nil {
		t.Fatal("expected error for invalid direction, got nil")
	}
	if len(mock.scripts) != 0 {
		t.Errorf("expected no script calls for invalid direction, got %d", len(mock.scripts))
	}
}

func TestCloseSurface(t *testing.T) {
	mock := &mockExecutor{}
	ctrl := NewWithExecutor(mock.exec)

	err := ctrl.CloseSurface()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.scripts) != 1 {
		t.Fatalf("expected 1 script call, got %d", len(mock.scripts))
	}
	want := `tell application "System Events" to tell process "ghostty" to keystroke "w" using {command down}`
	if mock.scripts[0] != want {
		t.Errorf("script =\n  %q\nwant\n  %q", mock.scripts[0], want)
	}
}

func TestExecScriptError(t *testing.T) {
	execErr := fmt.Errorf("osascript failed")
	mock := &mockExecutor{err: execErr}
	ctrl := NewWithExecutor(mock.exec)

	err := ctrl.NewWindow()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// The script should still be called
	if len(mock.scripts) != 1 {
		t.Fatalf("expected 1 script call, got %d", len(mock.scripts))
	}
}

func TestNewConstructor(t *testing.T) {
	ctrl := New()
	if ctrl == nil {
		t.Fatal("New() returned nil")
	}
	if ctrl.execScript == nil {
		t.Fatal("execScript is nil")
	}
}

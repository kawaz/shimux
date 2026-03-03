package ghostty_test

import (
	"fmt"
	"testing"

	"github.com/kawaz/shimux/internal/ghostty"
	"github.com/kawaz/shimux/internal/ghostty/ghosttytest"
)

func TestMockController(t *testing.T) {
	mock := &ghosttytest.MockController{}

	t.Run("NewSplit records call", func(t *testing.T) {
		err := mock.NewSplit(ghostty.SplitDown)
		if err != nil {
			t.Fatal(err)
		}
		if len(mock.Calls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(mock.Calls))
		}
		if mock.Calls[0].Method != "NewSplit" {
			t.Errorf("method = %q, want NewSplit", mock.Calls[0].Method)
		}
	})

	t.Run("SplitErr returns error", func(t *testing.T) {
		errMock := &ghosttytest.MockController{SplitErr: fmt.Errorf("test error")}
		err := errMock.NewSplit(ghostty.SplitDown)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("GotoSplit records call", func(t *testing.T) {
		m := &ghosttytest.MockController{}
		err := m.GotoSplit(ghostty.GotoNext)
		if err != nil {
			t.Fatal(err)
		}
		if len(m.Calls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(m.Calls))
		}
		if m.Calls[0].Method != "GotoSplit" {
			t.Errorf("method = %q, want GotoSplit", m.Calls[0].Method)
		}
	})

	t.Run("GotoErr returns error", func(t *testing.T) {
		errMock := &ghosttytest.MockController{GotoErr: fmt.Errorf("goto error")}
		err := errMock.GotoSplit(ghostty.GotoNext)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("CloseSurface records call", func(t *testing.T) {
		m := &ghosttytest.MockController{}
		err := m.CloseSurface()
		if err != nil {
			t.Fatal(err)
		}
		if len(m.Calls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(m.Calls))
		}
		if m.Calls[0].Method != "CloseSurface" {
			t.Errorf("method = %q, want CloseSurface", m.Calls[0].Method)
		}
	})

	t.Run("CloseErr returns error", func(t *testing.T) {
		errMock := &ghosttytest.MockController{CloseErr: fmt.Errorf("close error")}
		err := errMock.CloseSurface()
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("NewWindow records call", func(t *testing.T) {
		m := &ghosttytest.MockController{}
		err := m.NewWindow()
		if err != nil {
			t.Fatal(err)
		}
		if len(m.Calls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(m.Calls))
		}
		if m.Calls[0].Method != "NewWindow" {
			t.Errorf("method = %q, want NewWindow", m.Calls[0].Method)
		}
	})

	t.Run("NewTab records call", func(t *testing.T) {
		m := &ghosttytest.MockController{}
		err := m.NewTab()
		if err != nil {
			t.Fatal(err)
		}
		if len(m.Calls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(m.Calls))
		}
		if m.Calls[0].Method != "NewTab" {
			t.Errorf("method = %q, want NewTab", m.Calls[0].Method)
		}
	})

	t.Run("MockController implements Controller", func(t *testing.T) {
		var _ ghostty.Controller = &ghosttytest.MockController{}
	})
}

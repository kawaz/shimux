package pane

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestManagerRegister(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "panes.json"))

	// TC-PM001a
	t.Run("first registration returns 0", func(t *testing.T) {
		p, err := m.Register("/dev/ttys001")
		if err != nil {
			t.Fatal(err)
		}
		if p.ID != "0" {
			t.Errorf("ID = %q, want %q", p.ID, "0")
		}
	})

	// TC-PM001b
	t.Run("second registration returns 1", func(t *testing.T) {
		p, err := m.Register("/dev/ttys002")
		if err != nil {
			t.Fatal(err)
		}
		if p.ID != "1" {
			t.Errorf("ID = %q, want %q", p.ID, "1")
		}
	})

	// TC-PM001c
	t.Run("10 registrations sequential", func(t *testing.T) {
		m2 := NewManager(filepath.Join(t.TempDir(), "panes.json"))
		for i := 0; i < 10; i++ {
			p, err := m2.Register(fmt.Sprintf("/dev/ttys%03d", i))
			if err != nil {
				t.Fatal(err)
			}
			expected := fmt.Sprintf("%d", i)
			if p.ID != expected {
				t.Errorf("registration %d: ID = %q, want %q", i, p.ID, expected)
			}
		}
	})
}

func TestManagerGet(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "panes.json"))
	m.Register("/dev/ttys001")

	// TC-PM002a
	t.Run("get existing pane", func(t *testing.T) {
		p, err := m.Get("0")
		if err != nil {
			t.Fatal(err)
		}
		if p.TTY != "/dev/ttys001" {
			t.Errorf("TTY = %q, want /dev/ttys001", p.TTY)
		}
	})

	// TC-PM002b
	t.Run("get nonexistent pane", func(t *testing.T) {
		_, err := m.Get("99")
		if err == nil {
			t.Error("expected error for nonexistent pane")
		}
	})
}

func TestManagerUnregister(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "panes.json"))
	m.Register("/dev/ttys001")
	m.Register("/dev/ttys002")

	// TC-PM003a
	t.Run("unregister existing", func(t *testing.T) {
		err := m.Unregister("0")
		if err != nil {
			t.Fatal(err)
		}
		_, err = m.Get("0")
		if err == nil {
			t.Error("expected error for unregistered pane")
		}
	})

	// TC-PM003b
	t.Run("remaining pane still accessible", func(t *testing.T) {
		p, err := m.Get("1")
		if err != nil {
			t.Fatal(err)
		}
		if p.TTY != "/dev/ttys002" {
			t.Errorf("TTY = %q, want /dev/ttys002", p.TTY)
		}
	})

	// TC-PM003c
	t.Run("unregister nonexistent", func(t *testing.T) {
		err := m.Unregister("99")
		if err == nil {
			t.Error("expected error for nonexistent pane")
		}
	})
}

func TestManagerList(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "panes.json"))
	m.Register("/dev/ttys001")
	m.Register("/dev/ttys002")
	m.Register("/dev/ttys003")

	t.Run("list returns all panes", func(t *testing.T) {
		panes := m.List()
		if len(panes) != 3 {
			t.Errorf("len(panes) = %d, want 3", len(panes))
		}
	})

	t.Run("list preserves order", func(t *testing.T) {
		order := m.OrderedList()
		if len(order) != 3 {
			t.Fatalf("len(order) = %d, want 3", len(order))
		}
		if order[0] != "0" || order[1] != "1" || order[2] != "2" {
			t.Errorf("order = %v, want [0, 1, 2]", order)
		}
	})
}

func TestManagerActive(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "panes.json"))
	m.Register("/dev/ttys001")
	m.Register("/dev/ttys002")

	t.Run("first pane is active by default", func(t *testing.T) {
		p := m.GetActive()
		if p == nil {
			t.Fatal("expected active pane")
		}
		if p.ID != "0" {
			t.Errorf("active ID = %q, want %q", p.ID, "0")
		}
	})

	t.Run("set active changes active pane", func(t *testing.T) {
		err := m.SetActive("1")
		if err != nil {
			t.Fatal(err)
		}
		p := m.GetActive()
		if p == nil {
			t.Fatal("expected active pane")
		}
		if p.ID != "1" {
			t.Errorf("active ID = %q, want %q", p.ID, "1")
		}
	})

	t.Run("set active to nonexistent pane fails", func(t *testing.T) {
		err := m.SetActive("99")
		if err == nil {
			t.Error("expected error for nonexistent pane")
		}
	})
}

func TestManagerSaveLoad(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "panes.json")
	m := NewManager(stateFile)
	m.Register("/dev/ttys001")
	m.Register("/dev/ttys002")

	// TC-PM004a: save + load round trip
	t.Run("save and load round trip", func(t *testing.T) {
		if err := m.Save(); err != nil {
			t.Fatal(err)
		}

		m2 := NewManager(stateFile)
		if err := m2.Load(); err != nil {
			t.Fatal(err)
		}

		panes := m2.List()
		if len(panes) != 2 {
			t.Errorf("loaded %d panes, want 2", len(panes))
		}
	})

	// TC-PM004b: load preserves IDs
	t.Run("load preserves IDs", func(t *testing.T) {
		m2 := NewManager(stateFile)
		if err := m2.Load(); err != nil {
			t.Fatal(err)
		}

		p, err := m2.Get("0")
		if err != nil {
			t.Fatal(err)
		}
		if p.TTY != "/dev/ttys001" {
			t.Errorf("TTY = %q, want /dev/ttys001", p.TTY)
		}
	})

	// TC-PM004c: load preserves nextID
	t.Run("load preserves nextID counter", func(t *testing.T) {
		m2 := NewManager(stateFile)
		if err := m2.Load(); err != nil {
			t.Fatal(err)
		}

		p, err := m2.Register("/dev/ttys003")
		if err != nil {
			t.Fatal(err)
		}
		if p.ID != "2" {
			t.Errorf("next ID = %q, want %q", p.ID, "2")
		}
	})

	// TC-PM004d: load nonexistent file returns error
	t.Run("load nonexistent file returns error", func(t *testing.T) {
		m2 := NewManager(filepath.Join(t.TempDir(), "nonexistent.json"))
		err := m2.Load()
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestManagerSaveLoadNewFields(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "panes.json")
	m := NewManager(stateFile)
	p, err := m.Register("/dev/ttys005")
	if err != nil {
		t.Fatal(err)
	}
	p.PID = 42000
	p.CurrentPath = "/home/user/project"

	// TC-PM005a: save + load preserves PID
	t.Run("save and load preserves PID", func(t *testing.T) {
		if err := m.Save(); err != nil {
			t.Fatal(err)
		}
		m2 := NewManager(stateFile)
		if err := m2.Load(); err != nil {
			t.Fatal(err)
		}
		p2, err := m2.Get("0")
		if err != nil {
			t.Fatal(err)
		}
		if p2.PID != 42000 {
			t.Errorf("PID = %d, want 42000", p2.PID)
		}
	})

	// TC-PM005b: save + load preserves CurrentPath
	t.Run("save and load preserves CurrentPath", func(t *testing.T) {
		m2 := NewManager(stateFile)
		if err := m2.Load(); err != nil {
			t.Fatal(err)
		}
		p2, err := m2.Get("0")
		if err != nil {
			t.Fatal(err)
		}
		if p2.CurrentPath != "/home/user/project" {
			t.Errorf("CurrentPath = %q, want /home/user/project", p2.CurrentPath)
		}
	})

	// TC-PM005c: omitempty で PID=0, CurrentPath="" のペインもロード可能
	t.Run("zero values omitted in JSON are loaded as zero", func(t *testing.T) {
		sf := filepath.Join(t.TempDir(), "panes.json")
		m3 := NewManager(sf)
		m3.Register("/dev/ttys006") // PID=0, CurrentPath=""
		if err := m3.Save(); err != nil {
			t.Fatal(err)
		}
		m4 := NewManager(sf)
		if err := m4.Load(); err != nil {
			t.Fatal(err)
		}
		p4, err := m4.Get("0")
		if err != nil {
			t.Fatal(err)
		}
		if p4.PID != 0 {
			t.Errorf("PID = %d, want 0", p4.PID)
		}
		if p4.CurrentPath != "" {
			t.Errorf("CurrentPath = %q, want empty", p4.CurrentPath)
		}
	})
}

func TestManagerIndexOf(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "panes.json"))
	m.Register("/dev/ttys001")
	m.Register("/dev/ttys002")
	m.Register("/dev/ttys003")

	t.Run("first pane index is 0", func(t *testing.T) {
		idx := m.IndexOf("0")
		if idx != 0 {
			t.Errorf("IndexOf(0) = %d, want 0", idx)
		}
	})

	t.Run("second pane index is 1", func(t *testing.T) {
		idx := m.IndexOf("1")
		if idx != 1 {
			t.Errorf("IndexOf(1) = %d, want 1", idx)
		}
	})

	t.Run("third pane index is 2", func(t *testing.T) {
		idx := m.IndexOf("2")
		if idx != 2 {
			t.Errorf("IndexOf(2) = %d, want 2", idx)
		}
	})

	t.Run("nonexistent returns -1", func(t *testing.T) {
		idx := m.IndexOf("99")
		if idx != -1 {
			t.Errorf("IndexOf(99) = %d, want -1", idx)
		}
	})

	t.Run("after unregister index shifts", func(t *testing.T) {
		m.Unregister("0")
		idx := m.IndexOf("1")
		if idx != 0 {
			t.Errorf("IndexOf(1) after unregister = %d, want 0", idx)
		}
		idx = m.IndexOf("2")
		if idx != 1 {
			t.Errorf("IndexOf(2) after unregister = %d, want 1", idx)
		}
	})
}

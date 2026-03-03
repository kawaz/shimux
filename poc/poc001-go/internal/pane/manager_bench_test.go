package pane

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// newBenchManager creates a Manager for benchmarking (no file persistence).
func newBenchManager() *Manager {
	return NewManager(filepath.Join(os.TempDir(), "shimux-bench-panes.json"))
}

// --- Register benchmarks ---

func BenchmarkRegister(b *testing.B) {
	m := newBenchManager()
	for b.Loop() {
		m.Register("")
	}
}

// --- Unregister benchmarks ---

func benchmarkUnregister(b *testing.B, n int) {
	b.Helper()
	for b.Loop() {
		b.StopTimer()
		m := newBenchManager()
		ids := make([]string, n)
		for i := 0; i < n; i++ {
			p, _ := m.Register("")
			ids[i] = p.ID
		}
		b.StartTimer()
		for _, id := range ids {
			m.Unregister(id)
		}
	}
}

func BenchmarkUnregister_1(b *testing.B) {
	benchmarkUnregister(b, 1)
}

func BenchmarkUnregister_10(b *testing.B) {
	benchmarkUnregister(b, 10)
}

func BenchmarkUnregister_50(b *testing.B) {
	benchmarkUnregister(b, 50)
}

// --- Get benchmarks ---

func benchmarkGet(b *testing.B, n int) {
	b.Helper()
	m := newBenchManager()
	var targetID string
	for i := 0; i < n; i++ {
		p, _ := m.Register("")
		if i == n/2 {
			targetID = p.ID // target is the middle pane
		}
	}
	b.ResetTimer()
	for b.Loop() {
		m.Get(targetID)
	}
}

func BenchmarkGet_1(b *testing.B) {
	benchmarkGet(b, 1)
}

func BenchmarkGet_10(b *testing.B) {
	benchmarkGet(b, 10)
}

func BenchmarkGet_50(b *testing.B) {
	benchmarkGet(b, 50)
}

// --- List benchmarks ---

func benchmarkList(b *testing.B, n int) {
	b.Helper()
	m := newBenchManager()
	for i := 0; i < n; i++ {
		m.Register("")
	}
	b.ResetTimer()
	for b.Loop() {
		m.List()
	}
}

func BenchmarkList_1(b *testing.B) {
	benchmarkList(b, 1)
}

func BenchmarkList_10(b *testing.B) {
	benchmarkList(b, 10)
}

func BenchmarkList_50(b *testing.B) {
	benchmarkList(b, 50)
}

// --- GetActive benchmarks ---

func benchmarkGetActive(b *testing.B, n int) {
	b.Helper()
	m := newBenchManager()
	for i := 0; i < n; i++ {
		m.Register("")
	}
	b.ResetTimer()
	for b.Loop() {
		m.GetActive()
	}
}

func BenchmarkGetActive_1(b *testing.B) {
	benchmarkGetActive(b, 1)
}

func BenchmarkGetActive_10(b *testing.B) {
	benchmarkGetActive(b, 10)
}

func BenchmarkGetActive_50(b *testing.B) {
	benchmarkGetActive(b, 50)
}

// --- SetActive benchmarks ---

func benchmarkSetActive(b *testing.B, n int) {
	b.Helper()
	m := newBenchManager()
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		p, _ := m.Register("")
		ids[i] = p.ID
	}
	// Target the last registered pane
	targetID := ids[n-1]
	b.ResetTimer()
	for b.Loop() {
		m.SetActive(targetID)
	}
}

func BenchmarkSetActive_1(b *testing.B) {
	benchmarkSetActive(b, 1)
}

func BenchmarkSetActive_10(b *testing.B) {
	benchmarkSetActive(b, 10)
}

func BenchmarkSetActive_50(b *testing.B) {
	benchmarkSetActive(b, 50)
}

// --- Save/Load benchmarks ---

func benchmarkSave(b *testing.B, n int) {
	b.Helper()
	tmpDir := b.TempDir()
	m := NewManager(filepath.Join(tmpDir, "panes.json"))
	for i := 0; i < n; i++ {
		m.Register(fmt.Sprintf("/dev/pts/%d", i))
	}
	b.ResetTimer()
	for b.Loop() {
		if err := m.Save(); err != nil {
			b.Fatalf("save failed: %v", err)
		}
	}
}

func BenchmarkSave_1(b *testing.B) {
	benchmarkSave(b, 1)
}

func BenchmarkSave_10(b *testing.B) {
	benchmarkSave(b, 10)
}

func BenchmarkSave_50(b *testing.B) {
	benchmarkSave(b, 50)
}

func benchmarkLoad(b *testing.B, n int) {
	b.Helper()
	tmpDir := b.TempDir()
	stateFile := filepath.Join(tmpDir, "panes.json")
	// Create and save state
	m := NewManager(stateFile)
	for i := 0; i < n; i++ {
		m.Register(fmt.Sprintf("/dev/pts/%d", i))
	}
	if err := m.Save(); err != nil {
		b.Fatalf("save failed: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		loader := NewManager(stateFile)
		if err := loader.Load(); err != nil {
			b.Fatalf("load failed: %v", err)
		}
	}
}

func BenchmarkLoad_1(b *testing.B) {
	benchmarkLoad(b, 1)
}

func BenchmarkLoad_10(b *testing.B) {
	benchmarkLoad(b, 10)
}

func BenchmarkLoad_50(b *testing.B) {
	benchmarkLoad(b, 50)
}

// --- OrderedList benchmarks ---

func benchmarkOrderedList(b *testing.B, n int) {
	b.Helper()
	m := newBenchManager()
	for i := 0; i < n; i++ {
		m.Register("")
	}
	b.ResetTimer()
	for b.Loop() {
		m.OrderedList()
	}
}

func BenchmarkOrderedList_1(b *testing.B) {
	benchmarkOrderedList(b, 1)
}

func BenchmarkOrderedList_10(b *testing.B) {
	benchmarkOrderedList(b, 10)
}

func BenchmarkOrderedList_50(b *testing.B) {
	benchmarkOrderedList(b, 50)
}

package pane

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

// TestStress_ConcurrentRegisterUnregister は10 goroutineが同時にRegister→Unregisterを
// 繰り返し、IDの一意性とデータ整合性を検証する。
func TestStress_ConcurrentRegisterUnregister(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "panes.json"))

	const numGoroutines = 10
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	barrier := make(chan struct{})

	// 各goroutineが登録したIDを収集してユニーク性を検証
	idsCh := make(chan string, numGoroutines*opsPerGoroutine)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			<-barrier // 一斉スタート
			for i := 0; i < opsPerGoroutine; i++ {
				p, err := m.Register(fmt.Sprintf("/dev/ttys-g%d-i%d", gID, i))
				if err != nil {
					t.Errorf("goroutine %d: Register error: %v", gID, err)
					return
				}
				idsCh <- p.ID
				// 登録後すぐにUnregister
				if err := m.Unregister(p.ID); err != nil {
					t.Errorf("goroutine %d: Unregister(%q) error: %v", gID, p.ID, err)
					return
				}
			}
		}(g)
	}

	close(barrier)
	wg.Wait()
	close(idsCh)

	// IDの一意性を検証
	seen := make(map[string]bool)
	for id := range idsCh {
		if seen[id] {
			t.Errorf("duplicate ID detected: %q", id)
		}
		seen[id] = true
	}

	totalExpected := numGoroutines * opsPerGoroutine
	if len(seen) != totalExpected {
		t.Errorf("unique IDs = %d, want %d", len(seen), totalExpected)
	}

	// 全てUnregisterされたのでペインは0
	panes := m.List()
	if len(panes) != 0 {
		t.Errorf("remaining panes = %d, want 0", len(panes))
	}
}

// TestStress_ConcurrentReadWrite はReader goroutineがGet/Listを連続実行中に
// Writer goroutineがRegister/Unregisterを行い、パニックやdata raceが発生しないことを検証する。
func TestStress_ConcurrentReadWrite(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "panes.json"))

	// 事前にいくつかペインを登録しておく
	for i := 0; i < 5; i++ {
		m.Register(fmt.Sprintf("/dev/ttys-init%d", i))
	}

	const numReaders = 5
	const numWriters = 5
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	barrier := make(chan struct{})

	// Reader goroutines
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(rID int) {
			defer wg.Done()
			<-barrier
			for i := 0; i < opsPerGoroutine; i++ {
				_ = m.List()
				_ = m.OrderedList()
				_ = m.GetActive()
				// Getは存在しないIDでもエラーを返すだけでパニックしないこと
				_, _ = m.Get(fmt.Sprintf("%d", i%10))
			}
		}(r)
	}

	// Writer goroutines
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			<-barrier
			for i := 0; i < opsPerGoroutine; i++ {
				p, err := m.Register(fmt.Sprintf("/dev/ttys-w%d-i%d", wID, i))
				if err != nil {
					t.Errorf("writer %d: Register error: %v", wID, err)
					return
				}
				// 半分はすぐUnregister
				if i%2 == 0 {
					m.Unregister(p.ID)
				}
			}
		}(w)
	}

	close(barrier)
	wg.Wait()

	// パニックしなければ成功。ペイン数の整合性もチェック
	panes := m.List()
	order := m.OrderedList()
	if len(panes) != len(order) {
		t.Errorf("List() count = %d, OrderedList() count = %d, should match", len(panes), len(order))
	}
}

// TestStress_ConcurrentSetActive は10 goroutineが別々のペインを同時にSetActiveし、
// 最終的にActiveなペインが1つだけであることを検証する。
func TestStress_ConcurrentSetActive(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "panes.json"))

	const numPanes = 10
	paneIDs := make([]string, numPanes)
	for i := 0; i < numPanes; i++ {
		p, err := m.Register(fmt.Sprintf("/dev/ttys%03d", i))
		if err != nil {
			t.Fatal(err)
		}
		paneIDs[i] = p.ID
	}

	var wg sync.WaitGroup
	barrier := make(chan struct{})

	const rounds = 100
	for g := 0; g < numPanes; g++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			<-barrier
			for r := 0; r < rounds; r++ {
				_ = m.SetActive(paneIDs[gID])
			}
		}(g)
	}

	close(barrier)
	wg.Wait()

	// 最終状態: ちょうど1つだけがactive
	activeCount := 0
	for _, p := range m.List() {
		if p.Active {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("active pane count = %d, want 1", activeCount)
	}
}

// TestStress_ConcurrentSaveLoad はSaveとLoadを同時実行し、
// JSONファイルが壊れずデータ整合性が保たれることを検証する。
func TestStress_ConcurrentSaveLoad(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "panes.json")
	m := NewManager(stateFile)

	// 初期ペインを登録してSave
	for i := 0; i < 5; i++ {
		m.Register(fmt.Sprintf("/dev/ttys%03d", i))
	}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}

	const numGoroutines = 10
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	barrier := make(chan struct{})

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			<-barrier
			for i := 0; i < opsPerGoroutine; i++ {
				if gID%2 == 0 {
					// Save goroutines
					if err := m.Save(); err != nil {
						t.Errorf("goroutine %d: Save error: %v", gID, err)
						return
					}
				} else {
					// Load goroutines
					m2 := NewManager(stateFile)
					if err := m2.Load(); err != nil {
						// ファイルが書き込み中でエラーになることは許容するが、
						// パニックやdata raceは許容しない
						continue
					}
					// Load成功した場合、JSONが有効であることを確認
					panes := m2.List()
					if len(panes) == 0 {
						t.Errorf("goroutine %d: loaded 0 panes, expected > 0", gID)
					}
				}
			}
		}(g)
	}

	close(barrier)
	wg.Wait()

	// 最終的にSave/Loadが正しく動作することを確認
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	m2 := NewManager(stateFile)
	if err := m2.Load(); err != nil {
		t.Fatal(err)
	}
	panes := m2.List()
	if len(panes) == 0 {
		t.Error("final load returned 0 panes")
	}
}

// TestStress_LargeScaleOperations は100ペインを登録して同時操作し、
// 大量データでの整合性を検証する。
func TestStress_LargeScaleOperations(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "panes.json"))

	const numPanes = 100
	paneIDs := make([]string, numPanes)

	// 100ペインを順次登録
	for i := 0; i < numPanes; i++ {
		p, err := m.Register(fmt.Sprintf("/dev/ttys%03d", i))
		if err != nil {
			t.Fatal(err)
		}
		paneIDs[i] = p.ID
	}

	// 全ペインが登録されていることを確認
	panes := m.List()
	if len(panes) != numPanes {
		t.Fatalf("registered panes = %d, want %d", len(panes), numPanes)
	}

	var wg sync.WaitGroup
	barrier := make(chan struct{})

	// 同時に様々な操作を実行
	const numWorkers = 20

	// Get/List workers
	for w := 0; w < numWorkers/2; w++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			<-barrier
			for i := 0; i < 100; i++ {
				idx := (wID*100 + i) % numPanes
				_, _ = m.Get(paneIDs[idx])
				_ = m.List()
				_ = m.GetActive()
			}
		}(w)
	}

	// SetActive workers
	for w := numWorkers / 2; w < numWorkers; w++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			<-barrier
			for i := 0; i < 100; i++ {
				idx := (wID*100 + i) % numPanes
				_ = m.SetActive(paneIDs[idx])
			}
		}(w)
	}

	close(barrier)
	wg.Wait()

	// 最終整合性チェック
	finalPanes := m.List()
	if len(finalPanes) != numPanes {
		t.Errorf("final pane count = %d, want %d", len(finalPanes), numPanes)
	}

	activeCount := 0
	for _, p := range finalPanes {
		if p.Active {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("active pane count = %d, want 1", activeCount)
	}
}

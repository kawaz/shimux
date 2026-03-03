package tmux

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kawaz/shimux/internal/ghostty"
	"github.com/kawaz/shimux/internal/pane"
)

// threadSafeMock はスレッドセーフなController実装。
// ghosttytest.MockController はスライスへの非同期appendが unsafe なため、
// ストレステスト用にアトミックカウンタで呼び出し回数を記録する。
type threadSafeMock struct {
	splitCount atomic.Int64
	closeCount atomic.Int64
	gotoCount  atomic.Int64
}

func (m *threadSafeMock) NewWindow() error { return nil }
func (m *threadSafeMock) NewTab() error    { return nil }
func (m *threadSafeMock) NewSplit(_ ghostty.SplitDirection) error {
	m.splitCount.Add(1)
	return nil
}
func (m *threadSafeMock) GotoSplit(_ ghostty.GotoDirection) error {
	m.gotoCount.Add(1)
	return nil
}
func (m *threadSafeMock) CloseSurface() error {
	m.closeCount.Add(1)
	return nil
}

// newStressTestContext はストレステスト用のCommandContextを作成する。
// Stdout/Stderrは各goroutineで独立に使うため、ここでは共有用のダミーを設定。
// 実際のテストでは各goroutineが独自のBufferを使う。
func newStressTestContext(session string, mock *threadSafeMock) *CommandContext {
	pm := pane.NewWithDir(filepath.Join(os.TempDir(), fmt.Sprintf("shimux-stress-%d", os.Getpid())))
	return &CommandContext{
		Controller:  mock,
		PaneManager: pm,
		Session:     session,
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
	}
}

// TestStress_ConcurrentExecute は10 goroutineが異なるコマンドを同時実行し、
// パニックやdata raceが発生しないことを検証する。
func TestStress_ConcurrentExecute(t *testing.T) {
	mock := &threadSafeMock{}
	pm := pane.NewManager(filepath.Join(t.TempDir(), "panes.json"))

	// 初期ペインを登録（list-panes, display-message用）
	for i := 0; i < 3; i++ {
		pm.Register(fmt.Sprintf("/dev/ttys%03d", i))
	}

	commands := []*ParseResult{
		{Command: "has-session", Args: []string{"-t", "test-session"}},
		{Command: "switch-client"},
		{Command: "new-session"},
		{Command: "show-options", Args: []string{"-g", "prefix"}},
		{Command: "list-panes"},
		{Command: "list-panes", Args: []string{"-F", "#{pane_id}:#{pane_active}"}},
		{Command: "display-message", Args: []string{"-p", "#{session_name}"}},
		{Command: "display-message", Args: []string{"-p", "#{pane_id}"}},
		{Command: "select-pane", Args: []string{"-t", "%0"}},
		// send-keysはソケット接続が必要なので除外
	}

	const numGoroutines = 10
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	barrier := make(chan struct{})

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			// 各goroutineは独自のStdout/Stderrを持つ
			ctx := &CommandContext{
				Controller:  mock,
				PaneManager: pm,
				Session:     "test-session",
				Stdout:      &bytes.Buffer{},
				Stderr:      &bytes.Buffer{},
			}
			<-barrier
			for i := 0; i < opsPerGoroutine; i++ {
				cmd := commands[(gID*opsPerGoroutine+i)%len(commands)]
				_, err := Execute(ctx, cmd)
				// コマンド実行自体はエラーを返す可能性がある（has-sessionのexit 1等）が、
				// パニックしないことが重要
				_ = err
			}
		}(g)
	}

	close(barrier)
	wg.Wait()
	// パニックしなければ成功
}

// TestStress_SplitWindowListPanesRace はsplit-window中にlist-panesが実行される
// 競合状態をテストし、data raceが発生しないことを検証する。
func TestStress_SplitWindowListPanesRace(t *testing.T) {
	mock := &threadSafeMock{}
	pm := pane.NewManager(filepath.Join(t.TempDir(), "panes.json"))

	const numSplitters = 5
	const numListers = 5
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	barrier := make(chan struct{})

	// split-window goroutines
	for s := 0; s < numSplitters; s++ {
		wg.Add(1)
		go func(sID int) {
			defer wg.Done()
			ctx := &CommandContext{
				Controller:  mock,
				PaneManager: pm,
				Session:     "stress-session",
				Stdout:      &bytes.Buffer{},
				Stderr:      &bytes.Buffer{},
			}
			<-barrier
			for i := 0; i < opsPerGoroutine; i++ {
				result := &ParseResult{
					Command: "split-window",
					Args:    []string{"-P", "-F", "#{pane_id}"},
				}
				_, err := Execute(ctx, result)
				if err != nil {
					t.Errorf("splitter %d: %v", sID, err)
					return
				}
			}
		}(s)
	}

	// list-panes goroutines
	for l := 0; l < numListers; l++ {
		wg.Add(1)
		go func(lID int) {
			defer wg.Done()
			ctx := &CommandContext{
				Controller:  mock,
				PaneManager: pm,
				Session:     "stress-session",
				Stdout:      &bytes.Buffer{},
				Stderr:      &bytes.Buffer{},
			}
			<-barrier
			for i := 0; i < opsPerGoroutine; i++ {
				result := &ParseResult{
					Command: "list-panes",
					Args:    []string{"-F", "#{pane_id}:#{pane_active}"},
				}
				_, err := Execute(ctx, result)
				if err != nil {
					t.Errorf("lister %d: %v", lID, err)
					return
				}
			}
		}(l)
	}

	close(barrier)
	wg.Wait()

	// split-windowで登録されたペイン数を確認
	totalSplits := mock.splitCount.Load()
	expectedSplits := int64(numSplitters * opsPerGoroutine)
	if totalSplits != expectedSplits {
		t.Errorf("total splits = %d, want %d", totalSplits, expectedSplits)
	}

	// ペイン数は split 回数と一致するはず
	panes := pm.List()
	if int64(len(panes)) != expectedSplits {
		t.Errorf("registered panes = %d, want %d", len(panes), expectedSplits)
	}
}

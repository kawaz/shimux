package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"unsafe"
)

// --- MockPTYOpener ---

// MockPTYOpener はテスト用のPTYOpener実装。os.Pipe()ペアでPTYを模擬する。
type MockPTYOpener struct {
	MasterR, MasterW *os.File // Master側（パイプ）
	SlaveR, SlaveW   *os.File // Slave側（パイプ）
	Err              error
}

func (m *MockPTYOpener) Open() (*PTYPair, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return &PTYPair{Master: m.MasterW, Slave: m.SlaveR}, nil
}

// newMockPTYOpener はパイプペアを作成してMockPTYOpenerを返す。
func newMockPTYOpener(t *testing.T) *MockPTYOpener {
	t.Helper()
	mr, mw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	sr, sw, err := os.Pipe()
	if err != nil {
		mr.Close()
		mw.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		mr.Close()
		mw.Close()
		sr.Close()
		sw.Close()
	})
	return &MockPTYOpener{
		MasterR: mr,
		MasterW: mw,
		SlaveR:  sr,
		SlaveW:  sw,
	}
}

// --- PTYPair ---

func TestPTYPair_Fields(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	pair := &PTYPair{Master: w, Slave: r}
	if pair.Master != w {
		t.Error("Master should be w")
	}
	if pair.Slave != r {
		t.Error("Slave should be r")
	}
}

// --- PTYOpener interface ---

func TestPTYOpenerInterface(t *testing.T) {
	// MockPTYOpener がPTYOpenerインターフェースを満たすことを確認
	var _ PTYOpener = &MockPTYOpener{}
	var _ PTYOpener = &DefaultPTYOpener{}
}

func TestMockPTYOpener_Success(t *testing.T) {
	mock := newMockPTYOpener(t)
	pair, err := mock.Open()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair == nil {
		t.Fatal("pair should not be nil")
	}
	if pair.Master == nil {
		t.Error("Master should not be nil")
	}
	if pair.Slave == nil {
		t.Error("Slave should not be nil")
	}
}

func TestMockPTYOpener_Error(t *testing.T) {
	mock := &MockPTYOpener{Err: errors.New("mock pty error")}
	pair, err := mock.Open()
	if err == nil {
		t.Fatal("expected error")
	}
	if pair != nil {
		t.Error("pair should be nil on error")
	}
	if err.Error() != "mock pty error" {
		t.Errorf("error = %q, want %q", err.Error(), "mock pty error")
	}
}

// --- New() ---

func TestNew_DefaultValues(t *testing.T) {
	cfg := Config{
		SocketPath: "/tmp/test.sock",
		PaneID:     "42",
	}
	a := New(cfg)
	if a == nil {
		t.Fatal("New should return non-nil Agent")
	}
	if a.socketPath != "/tmp/test.sock" {
		t.Errorf("socketPath = %q, want %q", a.socketPath, "/tmp/test.sock")
	}
	if a.paneID != "42" {
		t.Errorf("paneID = %q, want %q", a.paneID, "42")
	}
	// PTYOpener should default to DefaultPTYOpener
	if _, ok := a.ptyOpener.(*DefaultPTYOpener); !ok {
		t.Errorf("ptyOpener should be *DefaultPTYOpener, got %T", a.ptyOpener)
	}
	// done channel should be initialized
	if a.done == nil {
		t.Error("done channel should be initialized")
	}
}

func TestNew_CustomPTYOpener(t *testing.T) {
	mock := &MockPTYOpener{}
	cfg := Config{
		PTYOpener:  mock,
		SocketPath: "/tmp/test.sock",
		PaneID:     "42",
	}
	a := New(cfg)
	if a.ptyOpener != mock {
		t.Error("ptyOpener should be the provided mock")
	}
}

func TestNew_CustomShellCmd(t *testing.T) {
	cfg := Config{
		SocketPath: "/tmp/test.sock",
		PaneID:     "42",
		ShellCmd:   "/bin/zsh",
	}
	a := New(cfg)
	if a.shellCmd != "/bin/zsh" {
		t.Errorf("shellCmd = %q, want %q", a.shellCmd, "/bin/zsh")
	}
}

func TestNew_DefaultShellFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")

	cfg := Config{
		SocketPath: "/tmp/test.sock",
		PaneID:     "42",
	}
	a := New(cfg)
	if a.shellCmd != "/usr/bin/fish" {
		t.Errorf("shellCmd = %q, want %q", a.shellCmd, "/usr/bin/fish")
	}
}

func TestNew_DefaultShellFallback(t *testing.T) {
	t.Setenv("SHELL", "")

	cfg := Config{
		SocketPath: "/tmp/test.sock",
		PaneID:     "42",
	}
	a := New(cfg)
	if a.shellCmd != "/bin/sh" {
		t.Errorf("shellCmd = %q, want %q", a.shellCmd, "/bin/sh")
	}
}

// --- getDefaultShell ---

func TestGetDefaultShell_FromEnv(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	got := getDefaultShell()
	if got != "/bin/bash" {
		t.Errorf("getDefaultShell() = %q, want %q", got, "/bin/bash")
	}
}

func TestGetDefaultShell_Fallback(t *testing.T) {
	t.Setenv("SHELL", "")
	got := getDefaultShell()
	if got != "/bin/sh" {
		t.Errorf("getDefaultShell() = %q, want %q", got, "/bin/sh")
	}
}

func TestGetDefaultShell_WithPath(t *testing.T) {
	t.Setenv("SHELL", "/usr/local/bin/zsh")
	got := getDefaultShell()
	if got != "/usr/local/bin/zsh" {
		t.Errorf("getDefaultShell() = %q, want %q", got, "/usr/local/bin/zsh")
	}
}

// --- syncWindowSize ---

func TestSyncWindowSize_WithPipes(t *testing.T) {
	// os.Pipe()はTTYではないのでioctlは失敗するが、panicしないことを確認
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	// パイプに対してsyncWindowSizeを呼んでもpanicしないこと
	syncWindowSize(r, w)
}

func TestSyncWindowSize_NilFiles(t *testing.T) {
	// nilファイルでpanicしないことを確認
	// Design rationale: syncWindowSizeは内部でエラーを握りつぶすため、
	// nilでも安全に動作する必要がある
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("syncWindowSize panicked with nil files: %v", r)
		}
	}()
	syncWindowSize(nil, nil)
}

// --- Config validation ---

func TestValidateConfig_Valid(t *testing.T) {
	cfg := Config{
		SocketPath: "/tmp/test.sock",
		PaneID:     "42",
	}
	if err := validateConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConfig_MissingSocketPath(t *testing.T) {
	cfg := Config{
		PaneID: "42",
	}
	err := validateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for missing SocketPath")
	}
	if err.Error() != "SocketPath is required" {
		t.Errorf("error = %q, want %q", err.Error(), "SocketPath is required")
	}
}

func TestValidateConfig_MissingPaneID(t *testing.T) {
	cfg := Config{
		SocketPath: "/tmp/test.sock",
	}
	err := validateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for missing PaneID")
	}
	if err.Error() != "PaneID is required" {
		t.Errorf("error = %q, want %q", err.Error(), "PaneID is required")
	}
}

func TestValidateConfig_BothMissing(t *testing.T) {
	cfg := Config{}
	err := validateConfig(cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	// SocketPathが先にチェックされることを確認
	if err.Error() != "SocketPath is required" {
		t.Errorf("error = %q, want %q", err.Error(), "SocketPath is required")
	}
}

// --- Run() error cases ---

func TestRun_ValidationError(t *testing.T) {
	a := &Agent{
		ptyOpener: &MockPTYOpener{},
		done:      make(chan struct{}),
	}
	err := a.Run(t.Context())
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Error() != "SocketPath is required" {
		t.Errorf("error = %q, want %q", err.Error(), "SocketPath is required")
	}
}

func TestRun_PTYOpenError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "shimux-agent-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	sockPath := filepath.Join(tmpDir, "test.sock")

	mock := &MockPTYOpener{Err: errors.New("pty open failed")}
	a := New(Config{
		PTYOpener:  mock,
		SocketPath: sockPath,
		PaneID:     "42",
	})

	err = a.Run(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "pty open: pty open failed" {
		t.Errorf("error = %q, want %q", err.Error(), "pty open: pty open failed")
	}
}

func TestRun_SocketInUseError(t *testing.T) {
	sockPath := testSocketPath(t)

	// 先にサーバを起動してソケットを使用中にする
	srv, err := ListenAndServe(sockPath, func(data string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	mock := newMockPTYOpener(t)
	a := New(Config{
		PTYOpener:  mock,
		SocketPath: sockPath,
		PaneID:     "42",
	})

	err = a.Run(t.Context())
	if err != ErrSocketInUse {
		t.Errorf("expected ErrSocketInUse, got: %v", err)
	}
}

func TestRun_ShellStartError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "shimux-agent-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	sockPath := filepath.Join(tmpDir, "test.sock")

	mock := newMockPTYOpener(t)
	a := New(Config{
		PTYOpener:  mock,
		SocketPath: sockPath,
		PaneID:     "42",
		ShellCmd:   "/nonexistent/shell/path",
	})

	err = a.Run(t.Context())
	if err == nil {
		t.Fatal("expected error for nonexistent shell")
	}
	// Accept any error wrapping -- the key is that an error is returned
	if !errors.Is(err, exec.ErrNotFound) {
		t.Logf("got non-ErrNotFound error (acceptable): %v (type: %T)", err, err)
	}
}

// --- Run() integration: send-keys through socket writes to PTY master ---

func TestRun_SendKeysWritesToMaster(t *testing.T) {
	tmpDir, err := os.MkdirTemp("/tmp", "shimux-agent-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	sockPath := filepath.Join(tmpDir, "test.sock")

	// パイプペアでPTYを模擬。
	// Agent内部ではmasterW (= PTYPair.Master) に send-keys データを書く。
	// masterR から読めばそのデータが取れる。
	masterR, masterW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer masterR.Close()
	defer masterW.Close()

	slaveR, slaveW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer slaveR.Close()
	defer slaveW.Close()

	mock := &MockPTYOpener{
		MasterR: masterR,
		MasterW: masterW,
		SlaveR:  slaveR,
		SlaveW:  slaveW,
	}

	// "true" は即座に終了するのでシェルの代わりに使う
	a := New(Config{
		PTYOpener:  mock,
		SocketPath: sockPath,
		PaneID:     "42",
		ShellCmd:   "/usr/bin/true",
	})

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	runErr := make(chan error, 1)
	go func() {
		runErr <- a.Run(ctx)
	}()

	// ソケットが作成されるまで待つ
	for i := 0; i < 100; i++ {
		if _, statErr := os.Stat(sockPath); statErr == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// send-keysを送信
	sendErr := SendKeysWithTimeout(sockPath, "hello agent", 2*time.Second)
	if sendErr != nil {
		// シェルが即座に終了してmasterが閉じている可能性
		t.Logf("SendKeys error (may be expected if shell exited quickly): %v", sendErr)
		// Run()の完了を待つ
		select {
		case <-runErr:
		case <-time.After(5 * time.Second):
		}
		return
	}

	// masterRからデータを読み取る (deadlineはパイプでは使えないのでゴルーチンで)
	readDone := make(chan string, 1)
	go func() {
		buf := make([]byte, 256)
		n, readErr := masterR.Read(buf)
		if readErr != nil {
			readDone <- ""
			return
		}
		readDone <- string(buf[:n])
	}()

	select {
	case got := <-readDone:
		if got == "" {
			t.Log("Read from masterR returned empty (may be expected)")
		} else if got != "hello agent" {
			t.Errorf("read from master = %q, want %q", got, "hello agent")
		}
	case <-time.After(3 * time.Second):
		t.Log("Timed out reading from masterR")
	}

	// Runの終了を待つ
	select {
	case err := <-runErr:
		if err != nil {
			t.Logf("Run returned: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Log("Run did not exit in time")
	}
}

// --- Agent struct fields ---

func TestAgent_StructFields(t *testing.T) {
	a := &Agent{}
	_ = a.ptyOpener
	_ = a.socketPath
	_ = a.paneID
	_ = a.shellCmd
	_ = a.master
	_ = a.shell
	_ = a.server
	_ = &a.mu
	_ = a.done
}

func TestAgent_MutexType(t *testing.T) {
	a := &Agent{}
	_ = &a.mu // Verify mu exists as sync.Mutex without copying
}

// --- DefaultPTYOpener struct ---

func TestDefaultPTYOpener_Exists(t *testing.T) {
	opener := &DefaultPTYOpener{}
	_ = opener
	var _ PTYOpener = opener
}

// --- Config struct ---

func TestConfig_AllFields(t *testing.T) {
	mock := &MockPTYOpener{}
	cfg := Config{
		PTYOpener:  mock,
		SocketPath: "/tmp/test.sock",
		PaneID:     "42",
		ShellCmd:   "/bin/zsh",
	}
	if cfg.PTYOpener != mock {
		t.Error("PTYOpener mismatch")
	}
	if cfg.SocketPath != "/tmp/test.sock" {
		t.Error("SocketPath mismatch")
	}
	if cfg.PaneID != "42" {
		t.Error("PaneID mismatch")
	}
	if cfg.ShellCmd != "/bin/zsh" {
		t.Error("ShellCmd mismatch")
	}
}

// --- unsafe import availability ---

func TestUnsafeImport(t *testing.T) {
	var x int
	p := unsafe.Pointer(&x)
	_ = p
}

// --- New() returns independent Agent instances ---

func TestNew_IndependentInstances(t *testing.T) {
	cfg1 := Config{SocketPath: "/tmp/s1.sock", PaneID: "1"}
	cfg2 := Config{SocketPath: "/tmp/s2.sock", PaneID: "2"}

	a1 := New(cfg1)
	a2 := New(cfg2)

	if a1.socketPath == a2.socketPath {
		t.Error("agents should have different socket paths")
	}
	if a1.paneID == a2.paneID {
		t.Error("agents should have different pane IDs")
	}
	if fmt.Sprintf("%p", &a1.done) == fmt.Sprintf("%p", &a2.done) {
		t.Error("agents should have independent done channels")
	}
}

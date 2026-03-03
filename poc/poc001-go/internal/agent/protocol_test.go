package agent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testSocketPath returns a short socket path inside /tmp to avoid macOS
// sun_path (104 byte) limitation. t.TempDir() produces very long paths.
func testSocketPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "shimux-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "t.sock")
}

// --- SocketDir / SocketPath / SafeSocketPath ---

func TestSocketDir(t *testing.T) {
	dir := SocketDir("mysession")
	want := filepath.Join(os.TempDir(), "shimux", "mysession")
	if dir != want {
		t.Errorf("SocketDir(%q) = %q, want %q", "mysession", dir, want)
	}
}

func TestSocketPath(t *testing.T) {
	p := SocketPath("mysession", "42")
	want := filepath.Join(os.TempDir(), "shimux", "mysession", "pane-42.sock")
	if p != want {
		t.Errorf("SocketPath = %q, want %q", p, want)
	}
}

func TestSafeSocketPath_Short(t *testing.T) {
	// Short session name should not be hashed.
	p := SafeSocketPath("sess", "0")
	want := SocketPath("sess", "0")
	if p != want {
		t.Errorf("SafeSocketPath short = %q, want %q", p, want)
	}
}

func TestSafeSocketPath_Long(t *testing.T) {
	// Create a session name long enough to exceed 104 bytes.
	longSession := strings.Repeat("a", 200)
	p := SafeSocketPath(longSession, "0")

	if len(p) > 104 {
		t.Errorf("SafeSocketPath should be <= 104 bytes, got %d: %q", len(p), p)
	}

	// Verify hashed session name is used.
	h := sha256.Sum256([]byte(longSession))
	hashed := fmt.Sprintf("%x", h)[:16]
	wantPath := SocketPath(hashed, "0")
	if p != wantPath {
		t.Errorf("SafeSocketPath hashed = %q, want %q", p, wantPath)
	}
}

func TestSafeSocketPath_BoundaryExact104(t *testing.T) {
	// Find the exact boundary: build paths with increasing session name length,
	// use unhashed path when <= 104, hashed when > 104.
	base := filepath.Join(os.TempDir(), "shimux") + string(filepath.Separator)
	suffix := string(filepath.Separator) + "pane-0.sock"
	maxSessionLen := 104 - len(base) - len(suffix)

	// Session name of exactly maxSessionLen should NOT be hashed.
	exactSession := strings.Repeat("x", maxSessionLen)
	p := SafeSocketPath(exactSession, "0")
	if p != SocketPath(exactSession, "0") {
		t.Errorf("session len %d should not be hashed", maxSessionLen)
	}

	// Session name of maxSessionLen+1 should be hashed.
	overSession := strings.Repeat("x", maxSessionLen+1)
	p2 := SafeSocketPath(overSession, "0")
	if p2 == SocketPath(overSession, "0") {
		t.Errorf("session len %d should be hashed", maxSessionLen+1)
	}
	if len(p2) > 104 {
		t.Errorf("hashed path should be <= 104, got %d", len(p2))
	}
}

// --- JSON encode/decode ---

func TestSendKeysRequestJSON(t *testing.T) {
	req := SendKeysRequest{Data: "hello\nworld"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var decoded SendKeysRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Data != req.Data {
		t.Errorf("decoded data = %q, want %q", decoded.Data, req.Data)
	}
}

func TestSendKeysResponseJSON_OK(t *testing.T) {
	resp := SendKeysResponse{OK: true}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var decoded SendKeysResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.OK {
		t.Error("decoded OK should be true")
	}
	if decoded.Error != "" {
		t.Errorf("decoded Error should be empty, got %q", decoded.Error)
	}
}

func TestSendKeysResponseJSON_Error(t *testing.T) {
	resp := SendKeysResponse{OK: false, Error: "some error"}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var decoded SendKeysResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.OK {
		t.Error("decoded OK should be false")
	}
	if decoded.Error != "some error" {
		t.Errorf("decoded Error = %q, want %q", decoded.Error, "some error")
	}
}

func TestSendKeysResponseJSON_OmitEmpty(t *testing.T) {
	resp := SendKeysResponse{OK: true}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"error"`) {
		t.Errorf("JSON should omit error field when empty: %s", data)
	}
}

// --- Client-Server communication ---

func TestClientServerBasic(t *testing.T) {
	sockPath := testSocketPath(t)

	var mu sync.Mutex
	var received string
	srv, err := ListenAndServe(sockPath, func(data string) error {
		mu.Lock()
		received = data
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	if err := SendKeysWithTimeout(sockPath, "hello tmux", 2*time.Second); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	got := received
	mu.Unlock()
	if got != "hello tmux" {
		t.Errorf("received = %q, want %q", got, "hello tmux")
	}
}

func TestClientServerHandlerError(t *testing.T) {
	sockPath := testSocketPath(t)

	srv, err := ListenAndServe(sockPath, func(data string) error {
		return fmt.Errorf("handler failed")
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// Use SendKeysWithTimeout (single attempt) to avoid retry delays.
	err = SendKeysWithTimeout(sockPath, "data", 2*time.Second)
	if err == nil {
		t.Fatal("expected error from handler")
	}
	if !strings.Contains(err.Error(), "handler failed") {
		t.Errorf("error should contain 'handler failed', got: %v", err)
	}
}

func TestClientServerMultipleRequests(t *testing.T) {
	sockPath := testSocketPath(t)

	var mu sync.Mutex
	var messages []string
	srv, err := ListenAndServe(sockPath, func(data string) error {
		mu.Lock()
		messages = append(messages, data)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("msg-%d", i)
		if err := SendKeysWithTimeout(sockPath, msg, 2*time.Second); err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(messages) != 5 {
		t.Errorf("got %d messages, want 5", len(messages))
	}
}

func TestServerAddr(t *testing.T) {
	sockPath := testSocketPath(t)
	srv, err := ListenAndServe(sockPath, func(data string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	if srv.Addr() != sockPath {
		t.Errorf("Addr() = %q, want %q", srv.Addr(), sockPath)
	}
}

func TestServerClose(t *testing.T) {
	sockPath := testSocketPath(t)
	srv, err := ListenAndServe(sockPath, func(data string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	if err := srv.Close(); err != nil {
		t.Fatal(err)
	}

	// After close, connection should fail.
	err = SendKeysWithTimeout(sockPath, "data", 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected error after server close")
	}
}

// --- Concurrent connection limit (max 16) ---

func TestConcurrentConnectionLimit(t *testing.T) {
	sockPath := testSocketPath(t)

	// Handler that blocks until we signal.
	var releaseOnce sync.Once
	release := make(chan struct{})
	var activeCount atomic.Int32

	srv, err := ListenAndServe(sockPath, func(data string) error {
		activeCount.Add(1)
		<-release
		activeCount.Add(-1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	defer releaseOnce.Do(func() { close(release) })

	// Launch 20 concurrent requests.
	const numClients = 20
	var wg sync.WaitGroup
	errs := make([]error, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = SendKeysWithTimeout(sockPath, fmt.Sprintf("data-%d", idx), 5*time.Second)
		}(i)
	}

	// Wait a bit for connections to be established.
	time.Sleep(500 * time.Millisecond)

	peak := activeCount.Load()
	if peak > 16 {
		t.Errorf("peak active handlers = %d, want <= 16", peak)
	}

	// Release all handlers.
	releaseOnce.Do(func() { close(release) })
	wg.Wait()
}

// --- Request size limit (1MB) ---

func TestRequestSizeLimit(t *testing.T) {
	sockPath := testSocketPath(t)

	srv, err := ListenAndServe(sockPath, func(data string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// Send data larger than 1MB. The JSON encoding adds overhead, but the
	// server uses LimitReader on the entire request body.
	bigData := strings.Repeat("A", 2*1024*1024) // 2MB
	err = SendKeysWithTimeout(sockPath, bigData, 5*time.Second)
	if err == nil {
		t.Fatal("expected error for oversized request")
	}
}

// --- Connection timeout ---

func TestConnectionTimeout(t *testing.T) {
	sockPath := testSocketPath(t)

	// Handler that never returns (blocks forever).
	block := make(chan struct{})

	srv, err := ListenAndServe(sockPath, func(data string) error {
		<-block
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// LIFO順: close(block) → srv.Close() の順で実行される。
	// ハンドラのブロック解除を先に行い、Server.Close()のwg.Wait()が完了できるようにする。
	defer srv.Close()
	defer close(block)

	// Use a very short timeout.
	start := time.Now()
	err = SendKeysWithTimeout(sockPath, "data", 500*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	// Should complete roughly within the timeout, not hang for 5 seconds.
	if elapsed > 3*time.Second {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

// --- SendKeys retry behavior ---

func TestSendKeysRetry(t *testing.T) {
	sockPath := testSocketPath(t)

	// Start server after a short delay to test retry logic.
	var srvMu sync.Mutex
	var srv *Server
	go func() {
		time.Sleep(300 * time.Millisecond)
		s, err := ListenAndServe(sockPath, func(data string) error { return nil })
		if err != nil {
			return
		}
		srvMu.Lock()
		srv = s
		srvMu.Unlock()
	}()
	t.Cleanup(func() {
		srvMu.Lock()
		if srv != nil {
			srv.Close()
		}
		srvMu.Unlock()
	})

	// SendKeys should retry and eventually succeed.
	if err := SendKeys(sockPath, "retry-test"); err != nil {
		t.Fatalf("SendKeys with retry failed: %v", err)
	}
}

// --- Stale socket cleanup ---

func TestCleanStaleSocket_NoFile(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "nonexistent.sock")
	err := CleanStaleSocket(sockPath)
	if err != nil {
		t.Fatalf("CleanStaleSocket on nonexistent file should return nil, got: %v", err)
	}
}

func TestCleanStaleSocket_StaleSocket(t *testing.T) {
	sockPath := testSocketPath(t)

	// Create a stale socket file by using syscall.Socket + bind directly,
	// so we can close the fd without auto-removing the file.
	// Alternatively, just create a unix socket file that nobody listens on.
	//
	// On macOS, net.Listen("unix", ...) + Close() removes the file.
	// Use a lower-level approach: create a listener, get the fd, and let
	// it leak (the file stays), or simply create a regular file at that path
	// and rely on CleanStaleSocket's "dial fails -> remove" logic.
	//
	// The simplest reliable approach: create a regular file. CleanStaleSocket
	// will fail to dial it (not a socket), treat it as stale, and remove it.
	if err := os.WriteFile(sockPath, []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	// Socket file should exist.
	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		t.Fatal("expected stale socket file to exist")
	}

	// CleanStaleSocket should detect it as stale and remove it.
	if err := CleanStaleSocket(sockPath); err != nil {
		t.Fatalf("CleanStaleSocket: %v", err)
	}

	// File should be removed.
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Fatal("stale socket file should have been removed")
	}
}

func TestCleanStaleSocket_ActiveSocket(t *testing.T) {
	sockPath := testSocketPath(t)

	// Start a real server.
	srv, err := ListenAndServe(sockPath, func(data string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// CleanStaleSocket should return ErrSocketInUse.
	err = CleanStaleSocket(sockPath)
	if err != ErrSocketInUse {
		t.Errorf("expected ErrSocketInUse, got: %v", err)
	}

	// Socket file should still exist.
	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		t.Fatal("active socket file should still exist")
	}
}

// --- Socket directory creation and permissions ---

func TestListenAndServe_CreatesDirectory(t *testing.T) {
	base := testSocketPath(t)
	dir := filepath.Join(filepath.Dir(base), "sub", "dir")
	sockPath := filepath.Join(dir, "t.sock")

	srv, err := ListenAndServe(sockPath, func(data string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("should be a directory")
	}
	// Check directory permissions (0700).
	perm := info.Mode().Perm()
	if perm != 0700 {
		t.Errorf("directory permission = %o, want 0700", perm)
	}
}

// --- Empty data ---

func TestSendKeysEmptyData(t *testing.T) {
	sockPath := testSocketPath(t)
	var mu sync.Mutex
	var received string
	var called bool
	srv, err := ListenAndServe(sockPath, func(data string) error {
		mu.Lock()
		received = data
		called = true
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	if err := SendKeysWithTimeout(sockPath, "", 2*time.Second); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	gotCalled := called
	gotReceived := received
	mu.Unlock()
	if !gotCalled {
		t.Fatal("handler was not called")
	}
	if gotReceived != "" {
		t.Errorf("received = %q, want empty", gotReceived)
	}
}

// --- ErrSocketInUse sentinel ---

func TestErrSocketInUse(t *testing.T) {
	if ErrSocketInUse.Error() != "socket is already in use by another process" {
		t.Errorf("ErrSocketInUse message = %q", ErrSocketInUse.Error())
	}
}

// --- SendKeysWithTimeout client function ---

func TestSendKeysWithTimeout_Success(t *testing.T) {
	sockPath := testSocketPath(t)
	srv, err := ListenAndServe(sockPath, func(data string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	if err := SendKeysWithTimeout(sockPath, "hello", 2*time.Second); err != nil {
		t.Fatal(err)
	}
}

// --- Server double close ---

func TestServerDoubleClose(t *testing.T) {
	sockPath := testSocketPath(t)
	srv, err := ListenAndServe(sockPath, func(data string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	if err := srv.Close(); err != nil {
		t.Fatal(err)
	}
	// Second close should not panic.
	_ = srv.Close()
}

// --- Helper for raw connection to test LimitReader ---

func TestRequestSizeLimitRaw(t *testing.T) {
	sockPath := testSocketPath(t)

	srv, err := ListenAndServe(sockPath, func(data string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// Connect directly and send a huge JSON payload.
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Build a JSON payload > 1MB.
	bigData := strings.Repeat("A", 2*1024*1024)
	req := SendKeysRequest{Data: bigData}
	data, _ := json.Marshal(req)

	conn.Write(data)
	// Signal end of write.
	if uc, ok := conn.(*net.UnixConn); ok {
		uc.CloseWrite()
	}

	// Read response - should be an error or connection closed.
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		// Connection reset or error is acceptable.
		return
	}
	if n > 0 {
		var resp SendKeysResponse
		if err := json.Unmarshal(buf[:n], &resp); err == nil {
			if resp.OK {
				t.Error("oversized request should not succeed")
			}
		}
	}
}

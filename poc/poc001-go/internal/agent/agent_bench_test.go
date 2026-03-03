package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- SafeSocketPath benchmarks ---

func BenchmarkSafeSocketPath_Short(b *testing.B) {
	for b.Loop() {
		SafeSocketPath("short", "0")
	}
}

func BenchmarkSafeSocketPath_Long(b *testing.B) {
	// Long session name that will exceed 104-byte sun_path limit
	longSession := strings.Repeat("a", 100)
	for b.Loop() {
		SafeSocketPath(longSession, "0")
	}
}

func BenchmarkSafeSocketPath_WithHash(b *testing.B) {
	// Session name long enough to trigger SHA-256 hashing
	longSession := strings.Repeat("very-long-session-name-", 10)
	for b.Loop() {
		SafeSocketPath(longSession, "42")
	}
}

func BenchmarkSocketPath(b *testing.B) {
	for b.Loop() {
		SocketPath("my-session", "0")
	}
}

func BenchmarkSocketDir(b *testing.B) {
	for b.Loop() {
		SocketDir("my-session")
	}
}

// benchSocketDir creates a short temporary directory for Unix sockets.
// Design rationale: b.TempDir() produces paths that exceed macOS's 104-byte
// sun_path limit, so we use /tmp directly with a short name.
func benchSocketDir(b *testing.B, name string) string {
	b.Helper()
	dir := filepath.Join("/tmp", "shimux-b-"+name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		b.Fatalf("mkdir: %v", err)
	}
	b.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// --- SendKeysWithTimeout round-trip benchmarks ---

func BenchmarkSendKeysWithTimeout_RoundTrip(b *testing.B) {
	dir := benchSocketDir(b, "rt")
	socketPath := filepath.Join(dir, "b.sock")

	srv, err := ListenAndServe(socketPath, func(data string) error {
		return nil // no-op handler
	})
	if err != nil {
		b.Fatalf("failed to start server: %v", err)
	}
	defer srv.Close()

	b.ResetTimer()
	for b.Loop() {
		if err := SendKeysWithTimeout(socketPath, "hello", connTimeout); err != nil {
			b.Fatalf("send failed: %v", err)
		}
	}
}

func BenchmarkSendKeysWithTimeout_LargePayload(b *testing.B) {
	dir := benchSocketDir(b, "lg")
	socketPath := filepath.Join(dir, "b.sock")

	srv, err := ListenAndServe(socketPath, func(data string) error {
		return nil
	})
	if err != nil {
		b.Fatalf("failed to start server: %v", err)
	}
	defer srv.Close()

	// 64KB payload
	largeData := strings.Repeat("x", 64*1024)
	b.ResetTimer()
	for b.Loop() {
		if err := SendKeysWithTimeout(socketPath, largeData, connTimeout); err != nil {
			b.Fatalf("send failed: %v", err)
		}
	}
}

func BenchmarkSendKeysWithTimeout_SmallPayload(b *testing.B) {
	dir := benchSocketDir(b, "sm")
	socketPath := filepath.Join(dir, "b.sock")

	srv, err := ListenAndServe(socketPath, func(data string) error {
		return nil
	})
	if err != nil {
		b.Fatalf("failed to start server: %v", err)
	}
	defer srv.Close()

	b.ResetTimer()
	for b.Loop() {
		if err := SendKeysWithTimeout(socketPath, "a", connTimeout); err != nil {
			b.Fatalf("send failed: %v", err)
		}
	}
}

// --- CleanStaleSocket benchmarks ---

func BenchmarkCleanStaleSocket_NonExistent(b *testing.B) {
	path := filepath.Join(os.TempDir(), "shimux-bench-nonexistent.sock")
	for b.Loop() {
		CleanStaleSocket(path)
	}
}

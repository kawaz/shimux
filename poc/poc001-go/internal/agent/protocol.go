package agent

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// SendKeysRequest はsend-keysリクエスト。
type SendKeysRequest struct {
	Data string `json:"data"` // PTYに書き込むテキスト
}

// SendKeysResponse はsend-keysレスポンス。
type SendKeysResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// ErrSocketInUse は別プロセスがソケットを使用中であることを示す。
var ErrSocketInUse = errors.New("socket is already in use by another process")

const (
	maxSunPath        = 104     // macOS の sun_path 制限
	maxRequestSize    = 1 << 20 // 1MB
	maxConcurrent     = 16
	connTimeout       = 5 * time.Second
	retryInitInterval = 10 * time.Millisecond
	retryMaxInterval  = 200 * time.Millisecond
	retryMaxAttempts  = 10
	staleDialTimeout  = 500 * time.Millisecond
)

// SocketDir はセッションのソケットディレクトリパスを返す。
// パス: <tmpdir>/shimux/<session>/
func SocketDir(session string) string {
	return filepath.Join(os.TempDir(), "shimux", session)
}

// SocketPath はペインのソケットパスを返す。
// パス: <tmpdir>/shimux/<session>/pane-<id>.sock
func SocketPath(session string, paneID string) string {
	return filepath.Join(SocketDir(session), "pane-"+paneID+".sock")
}

// SafeSocketPath はパス長制限を考慮したソケットパスを返す。
// macOS の sun_path 制限は 104バイト。
// パス長が104バイトを超える場合、セッション名をSHA-256の先頭16文字にハッシュ化。
func SafeSocketPath(session string, paneID string) string {
	p := SocketPath(session, paneID)
	if len(p) <= maxSunPath {
		return p
	}
	h := sha256.Sum256([]byte(session))
	hashed := fmt.Sprintf("%x", h)[:16]
	return SocketPath(hashed, paneID)
}

// Server はUnixソケットサーバ。
type Server struct {
	listener  net.Listener
	handler   func(data string) error
	sem       chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// ListenAndServe はUnixソケットサーバを起動する。
// handler はsend-keysリクエストを処理する関数。
// 同時接続数: 最大16、リクエストサイズ: 最大1MB、接続タイムアウト: 5秒。
func ListenAndServe(socketPath string, handler func(data string) error) (*Server, error) {
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create socket directory: %w", err)
	}

	// umask(0077) でソケット作成し、0600 パーミッションを保証。
	oldUmask := syscall.Umask(0077)
	ln, err := net.Listen("unix", socketPath)
	syscall.Umask(oldUmask)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", socketPath, err)
	}

	srv := &Server{
		listener: ln,
		handler:  handler,
		sem:      make(chan struct{}, maxConcurrent),
	}

	srv.wg.Add(1)
	go srv.acceptLoop()

	return srv, nil
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Listener closed.
			return
		}
		s.sem <- struct{}{} // Acquire semaphore slot.
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() { <-s.sem }() // Release semaphore slot.
			s.handleConn(conn)
		}()
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	// 接続タイムアウト設定。
	conn.SetDeadline(time.Now().Add(connTimeout))

	// ストリーミングDecoder + LimitReader でメモリ効率よくサイズ制限。
	// maxRequestSize+1 バイトまで読み、Decode後に残りバイト数が0ならサイズ超過。
	lr := &io.LimitedReader{R: conn, N: maxRequestSize + 1}
	var req SendKeysRequest
	if err := json.NewDecoder(lr).Decode(&req); err != nil {
		msg := "invalid request: " + err.Error()
		if lr.N == 0 {
			msg = "request too large"
		}
		resp := SendKeysResponse{OK: false, Error: msg}
		json.NewEncoder(conn).Encode(resp)
		return
	}

	if err := s.handler(req.Data); err != nil {
		resp := SendKeysResponse{OK: false, Error: err.Error()}
		json.NewEncoder(conn).Encode(resp)
		return
	}

	resp := SendKeysResponse{OK: true}
	json.NewEncoder(conn).Encode(resp)
}

// Close はサーバを停止し、in-flightハンドラの完了を待つ。
func (s *Server) Close() error {
	var err error
	s.closeOnce.Do(func() {
		err = s.listener.Close()
	})
	s.wg.Wait()
	return err
}

// Addr はソケットアドレスを返す。
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

// SendKeys はソケット経由でsend-keysを送信する。
// リトライ: 指数バックオフ（初期10ms、倍率2x、上限200ms）x 最大10回。
func SendKeys(socketPath string, data string) error {
	var lastErr error
	interval := retryInitInterval
	for i := 0; i < retryMaxAttempts; i++ {
		err := SendKeysWithTimeout(socketPath, data, connTimeout)
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(interval)
		interval *= 2
		if interval > retryMaxInterval {
			interval = retryMaxInterval
		}
	}
	return fmt.Errorf("send-keys failed after %d retries: %w", retryMaxAttempts, lastErr)
}

// SendKeysWithTimeout はタイムアウト付きで送信する。
func SendKeysWithTimeout(socketPath string, data string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", socketPath, err)
	}
	defer conn.Close()

	// タイムアウト設定。
	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}

	req := SendKeysRequest{Data: data}
	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	if _, err := conn.Write(reqData); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	// クライアント側から書き込み終了を通知して、サーバのDecoderがEOFを検出できるようにする。
	if uc, ok := conn.(*net.UnixConn); ok {
		uc.CloseWrite()
	}

	var resp SendKeysResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if !resp.OK {
		return fmt.Errorf("server error: %s", resp.Error)
	}
	return nil
}

// CleanStaleSocket は古いソケットファイルを検出・削除する。
// - net.DialTimeout() で生死確認（500ms）
// - 接続失敗 -> 古いソケット削除して nil を返す
// - 接続成功 -> 別プロセス稼働中のため ErrSocketInUse を返す
func CleanStaleSocket(socketPath string) error {
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil
	}

	conn, err := net.DialTimeout("unix", socketPath, staleDialTimeout)
	if err != nil {
		// 接続失敗 = stale socket。削除する。
		if removeErr := os.Remove(socketPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("remove stale socket: %w", removeErr)
		}
		return nil
	}
	// 接続成功 = 別プロセスが使用中。
	conn.Close()
	return ErrSocketInUse
}

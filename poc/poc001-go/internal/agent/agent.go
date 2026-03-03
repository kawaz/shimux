package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"unsafe"
)

// PTYPair はPTYのマスター/スレーブペア。
type PTYPair struct {
	Master *os.File
	Slave  *os.File
}

// PTYOpener はPTYペアを開くインターフェース（テスト用に差し替え可能）。
type PTYOpener interface {
	Open() (*PTYPair, error)
}

// DefaultPTYOpener は実際のPTYを開く実装。
type DefaultPTYOpener struct{}

// Open は /dev/ptmx を使って実際のPTYペアを開く。
// macOS: posix_openpt + grantpt + unlockpt + ptsname 相当の操作を行う。
func (d *DefaultPTYOpener) Open() (*PTYPair, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/ptmx: %w", err)
	}

	// grantpt: macOS では ioctl(fd, TIOCPTYGRANT, 0)
	if err := ioctl(master.Fd(), syscall.TIOCPTYGRANT, 0); err != nil {
		master.Close()
		return nil, fmt.Errorf("grantpt: %w", err)
	}

	// unlockpt: macOS では ioctl(fd, TIOCPTYUNLK, 0)
	if err := ioctl(master.Fd(), syscall.TIOCPTYUNLK, 0); err != nil {
		master.Close()
		return nil, fmt.Errorf("unlockpt: %w", err)
	}

	// ptsname: macOS では ioctl(fd, TIOCPTYGNAME, buf)
	var buf [128]byte
	if err := ioctl(master.Fd(), syscall.TIOCPTYGNAME, uintptr(unsafe.Pointer(&buf[0]))); err != nil {
		master.Close()
		return nil, fmt.Errorf("ptsname: %w", err)
	}

	// buf はnull終端文字列
	slaveName := ""
	for i, b := range buf {
		if b == 0 {
			slaveName = string(buf[:i])
			break
		}
	}
	if slaveName == "" {
		master.Close()
		return nil, fmt.Errorf("ptsname: empty slave name")
	}

	slave, err := os.OpenFile(slaveName, os.O_RDWR, 0)
	if err != nil {
		master.Close()
		return nil, fmt.Errorf("open slave %s: %w", slaveName, err)
	}

	return &PTYPair{Master: master, Slave: slave}, nil
}

// ioctl は syscall.Syscall を使って ioctl を実行する。
func ioctl(fd uintptr, req uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}

// Agent はPTYプロキシ。
type Agent struct {
	ptyOpener  PTYOpener
	socketPath string
	paneID     string
	shellCmd   string // デフォルト: 環境変数SHELLまたは/bin/sh

	master *os.File
	shell  *exec.Cmd
	server *Server // protocol.goのServer
	mu     sync.Mutex
	done   chan struct{}
}

// Config はAgent設定。
type Config struct {
	PTYOpener  PTYOpener // nil -> DefaultPTYOpener
	SocketPath string    // 必須
	PaneID     string    // 必須
	ShellCmd   string    // 空 -> SHELL環境変数 -> /bin/sh
}

// New はAgentを作成する。
func New(cfg Config) *Agent {
	opener := cfg.PTYOpener
	if opener == nil {
		opener = &DefaultPTYOpener{}
	}

	shell := cfg.ShellCmd
	if shell == "" {
		shell = getDefaultShell()
	}

	return &Agent{
		ptyOpener:  opener,
		socketPath: cfg.SocketPath,
		paneID:     cfg.PaneID,
		shellCmd:   shell,
		done:       make(chan struct{}),
	}
}

// getDefaultShell はデフォルトシェルパスを返す。
// SHELL環境変数 -> /bin/sh の順でフォールバック。
func getDefaultShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}

// validateConfig はConfig値をバリデーションする。
func validateConfig(cfg Config) error {
	if cfg.SocketPath == "" {
		return fmt.Errorf("SocketPath is required")
	}
	if cfg.PaneID == "" {
		return fmt.Errorf("PaneID is required")
	}
	return nil
}

// syncWindowSize は from の端末ウィンドウサイズを to にコピーする。
// from/to が nil またはTTYでない場合はエラーを無視する。
func syncWindowSize(from, to *os.File) {
	if from == nil || to == nil {
		return
	}

	var ws [4]uint16 // rows, cols, xpixel, ypixel
	// TIOCGWINSZ: 端末のウィンドウサイズを取得
	if err := ioctl(from.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws[0]))); err != nil {
		return // fromがTTYでない場合は無視
	}
	// TIOCSWINSZ: 端末のウィンドウサイズを設定
	_ = ioctl(to.Fd(), syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&ws[0])))
}

// Run はAgentのメイン処理を実行する。
func (a *Agent) Run(ctx context.Context) error {
	// バリデーション
	if err := validateConfig(Config{SocketPath: a.socketPath, PaneID: a.paneID}); err != nil {
		return err
	}

	// 1. Stale socket cleanup
	if err := CleanStaleSocket(a.socketPath); err != nil {
		return err // ErrSocketInUse の場合は別プロセス稼働中
	}

	// 2. PTY作成
	pty, err := a.ptyOpener.Open()
	if err != nil {
		return fmt.Errorf("pty open: %w", err)
	}
	// Design rationale: defer pty.Master.Close() を置かない。
	// Master は266行付近で明示的にCloseする。二重Closeを避けるため defer は使わない。

	a.mu.Lock()
	a.master = pty.Master
	a.mu.Unlock()

	// 3. ウィンドウサイズ同期
	syncWindowSize(os.Stdin, pty.Master)

	// 4. シェル起動
	shell := exec.CommandContext(ctx, a.shellCmd)
	shell.Stdin = pty.Slave
	shell.Stdout = pty.Slave
	shell.Stderr = pty.Slave
	shell.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    int(pty.Slave.Fd()), // macOS: 親プロセスのfd番号
	}
	if err := shell.Start(); err != nil {
		return fmt.Errorf("shell start: %w", err)
	}

	a.mu.Lock()
	a.shell = shell
	a.mu.Unlock()

	// 5. 親プロセス側slave fdクローズ
	pty.Slave.Close()

	// 6. Unixソケットサーバ開始
	server, err := ListenAndServe(a.socketPath, func(data string) error {
		a.mu.Lock()
		m := a.master
		a.mu.Unlock()
		if m == nil {
			return fmt.Errorf("master is closed")
		}
		_, err := m.Write([]byte(data))
		return err
	})
	if err != nil {
		// シェルプロセスを終了させる
		shell.Process.Kill()
		shell.Wait()
		return fmt.Errorf("listen: %w", err)
	}

	a.mu.Lock()
	a.server = server
	a.mu.Unlock()
	defer server.Close()

	// 7. I/Oプロキシ
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		// Design rationale: シェル終了後、Master.Close() により Write が EBADF を返し、
		// io.Copy がエラーで脱出する。stdin 側の Read がブロックしていても、
		// Master 側の Write エラーで安全に終了するため、追加のキャンセル機構は不要。
		io.Copy(pty.Master, os.Stdin)
	}()
	go func() {
		defer wg.Done()
		// Design rationale: Master.Close() により Read が EBADF/EIO を返し、
		// io.Copy が終了する。
		io.Copy(os.Stdout, pty.Master)
	}()

	// 8. SIGWINCH転送
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			syncWindowSize(os.Stdin, pty.Master)
		}
	}()

	// 9. シェル終了を待つ
	shellErr := shell.Wait()

	// 10. シャットダウン
	signal.Stop(sigCh)
	close(sigCh)
	server.Close()

	// masterを閉じてI/Oプロキシを停止
	pty.Master.Close()
	a.mu.Lock()
	a.master = nil
	a.mu.Unlock()

	wg.Wait()
	close(a.done)

	return shellErr
}

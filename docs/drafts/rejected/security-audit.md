# PTY Proxy Security Audit Report

**Date**: 2026-02-23
**Scope**: gmux PTY proxy architecture (desk audit, no runtime testing)
**Auditor**: Automated security review
**Commit**: wip-redesign branch (main HEAD: 7b62385)

---

## Executive Summary

gmux implements a PTY proxy architecture where `gmux-agent` creates an internal PTY pair, launches a shell on the slave side, and accepts `send-keys` commands via a Unix domain socket. This architecture is used to allow Claude Code (and other tmux-dependent tools) to inject keystrokes into Ghostty terminal panes.

The overall security posture is **reasonable for a single-user desktop tool**. The codebase demonstrates awareness of key security concerns: socket permissions are enforced via `umask(0077)`, directory permissions are set to `0700`, request size is capped at 1MB, and concurrent connections are limited. However, several areas warrant attention before production use, particularly around `/tmp` directory usage, state file permissions, and the absence of input validation on data written to the PTY master.

**Finding summary**:
- Critical: 0
- High: 1
- Medium: 3
- Low: 3
- Info: 2

---

## Findings

### [High] H-1: Symlink attack on `/tmp/gmux/` directory (TOCTOU)

- **File**: `internal/agent/protocol.go:46-47`, `internal/agent/protocol.go:81-84`
- **Risk**: An attacker with local access could pre-create a symlink at `/tmp/gmux/<session>/` pointing to an arbitrary directory before `gmux-agent` starts. When `MkdirAll(dir, 0700)` is called, if `/tmp/gmux/` already exists as a symlink, the socket file could be created in an attacker-controlled location. Conversely, an attacker could place a rogue socket at the expected path, causing `CleanStaleSocket` to connect to a malicious server.
- **Current state**: `SocketDir()` uses `os.TempDir()` (typically `/tmp` on macOS) to construct the path `/tmp/gmux/<session>/`. `MkdirAll` does not verify that intermediate directories are not symlinks. There is no sticky-bit or ownership check on `/tmp/gmux/`.
- **Scenario**: On a multi-user macOS system (shared workstation, CI server), another user creates `/tmp/gmux` as a symlink to `/tmp/attacker-controlled/`. When the victim's `gmux-agent` starts, it creates the socket inside the attacker's directory, which the attacker can then connect to and inject arbitrary keystrokes.
- **Recommended fix**:
  1. Use `$XDG_RUNTIME_DIR` (Linux) or `$TMPDIR` (macOS, per-user) instead of `os.TempDir()`. On macOS, `$TMPDIR` points to `/var/folders/xx/.../.../T/` which is per-user and not shared. `os.TempDir()` on macOS already returns `$TMPDIR` if set, but the function's documentation notes it may fall back to `/tmp`. Explicitly prefer `$TMPDIR` with a per-user fallback.
  2. After `MkdirAll`, verify that the created directory and its parents are owned by the current user and are not symlinks. Use `os.Lstat` instead of `os.Stat` for this check.
  3. Consider using `os.MkdirTemp` for the session-level directory to add randomness.

```go
// Current (protocol.go:46)
func SocketDir(session string) string {
    return filepath.Join(os.TempDir(), "gmux", session)
}

// Recommended: explicit per-user directory
func SocketDir(session string) string {
    // macOS: $TMPDIR is per-user (/var/folders/...)
    // Linux: $XDG_RUNTIME_DIR is per-user (/run/user/<uid>/)
    base := os.Getenv("XDG_RUNTIME_DIR")
    if base == "" {
        base = os.TempDir() // macOS $TMPDIR is already per-user
    }
    return filepath.Join(base, "gmux", session)
}
```

---

### [Medium] M-1: No input validation on PTY write data (terminal escape injection)

- **File**: `internal/agent/agent.go:215-224`, `internal/agent/protocol.go:144`
- **Risk**: The `send-keys` handler writes the received `data` string directly to `master.Write([]byte(data))` without any validation or sanitization. A malicious (or compromised) tmux client can send arbitrary terminal escape sequences including:
  - **OSC sequences**: `\x1b]0;malicious-title\x07` to change the terminal title (used in social engineering attacks to mislead the user about what is running)
  - **CSI sequences**: Cursor manipulation, screen clearing, or overwriting visible output
  - **OSC 52**: Clipboard manipulation (`\x1b]52;c;<base64-data>\x07`) to silently modify clipboard contents
  - **Bracketed paste abuse**: Sequences that break out of bracketed paste mode
- **Current state**: The `SendKeysRequest.Data` field is a raw string with no filtering. The 1MB size limit (`maxRequestSize`) prevents memory exhaustion but does not address content-level concerns.
- **Mitigating factors**: In the expected use case, the only client is `gmux` itself (invoked by Claude Code), so the attack surface requires compromising the Claude Code process or having local access to the socket file. This is a defense-in-depth concern rather than a direct vulnerability.
- **Recommended fix**:
  1. **Option A (strict)**: Reject or strip sequences that are not printable text or known-safe control characters (`\r`, `\n`, `\t`, `\x1b[A-D` for arrow keys, etc.). This would be a whitelist approach.
  2. **Option B (pragmatic)**: Add an optional `--raw` mode to send-keys and default to filtering dangerous escape sequences (OSC 52, OSC 0, etc.). This matches the defense-in-depth principle.
  3. **Option C (document)**: Accept the risk with a design rationale comment noting that the socket is protected by file permissions and the only expected client is gmux itself.

---

### [Medium] M-2: State file (`panes.json`) created without explicit permission control

- **File**: `internal/pane/manager.go:188-209`
- **Risk**: `os.CreateTemp(dir, "panes-*.json.tmp")` creates the temp file with the default permission of `0600` (Go's `os.CreateTemp` behavior), which is secure. However, the state directory `~/.local/state/gmux/` is created implicitly (if the caller creates it) and its permissions are not explicitly enforced by the `Manager`. If the directory has overly permissive permissions (e.g., `0755`), other users can read pane state information.
- **Current state**: The `DefaultStateDir()` returns `~/.local/state/gmux/` but `NewManager`/`NewWithDir` do not create the directory. The directory creation is left to the `Save()` method's `os.CreateTemp` call, which will fail if the directory does not exist. The caller (`cmd/gmux/main.go`) does not explicitly create or permission-check this directory either.
- **Data exposed**: Pane IDs, TTY paths, active pane state, window dimensions. This is low-sensitivity information but could aid reconnaissance.
- **Recommended fix**: In `Save()`, before `CreateTemp`, ensure the directory exists with `os.MkdirAll(dir, 0700)` and verify ownership/permissions.

---

### [Medium] M-3: Race condition in `CleanStaleSocket` (TOCTOU)

- **File**: `internal/agent/protocol.go:237-253`
- **Risk**: There is a time-of-check-to-time-of-use gap between the `os.Stat` check, the `net.DialTimeout` connection attempt, and the `os.Remove` removal. Between the connection failure and the removal, another legitimate `gmux-agent` process could have started and bound to that socket path. The `Remove` would then delete the live socket.
- **Current state**: The sequence is: (1) Stat check -> (2) Dial attempt -> (3) If dial fails, remove. Another process starting between steps 2 and 3 would lose its socket.
- **Mitigating factors**: This race window is very small (sub-millisecond) and requires a precise interleaving of two `gmux-agent` processes starting for the same pane ID simultaneously. In practice, this is unlikely because each pane ID maps to a single Ghostty pane.
- **Recommended fix**: Use `flock` on a lockfile adjacent to the socket, or use `SO_REUSEADDR` semantics. Alternatively, document the limitation with a design rationale comment noting the single-pane-ID invariant.

---

### [Low] L-1: Wrapper temp directory permissions inherit default umask

- **File**: `internal/wrapper/wrapper.go:70-71`
- **Risk**: `os.MkdirTemp("", "gmux-wrapper-")` creates a temporary directory with permissions `0700` (Go's default for MkdirTemp, which applies `0700` regardless of umask). The symlink inside (`tmux -> gmux`) is accessible only to the current user. This is secure.
- **Current state**: The implementation is correct. Go's `os.MkdirTemp` uses `0700` unconditionally.
- **Note**: No action required. Documented for completeness.

---

### [Low] L-2: Session name predictability in socket path

- **File**: `internal/agent/protocol.go:51-53`, `internal/wrapper/wrapper.go:47-48`
- **Risk**: The socket path is `<tmpdir>/gmux/<session>/pane-<id>.sock`. The session name is derived from the PID (`gmux-<pid>`) or from the repository/worktree name. Both are predictable. Combined with H-1 (shared `/tmp`), an attacker who knows the victim's session name can pre-create the socket directory.
- **Current state**: `GenerateSessionName` in wrapper.go sanitizes the name (`[^a-zA-Z0-9-]` -> `_`) but does not add randomness. The default session name `gmux-<pid>` uses the PID which is observable by other users on the system (`ps`).
- **Mitigating factors**: On macOS with `$TMPDIR` set (the typical case), the base path is per-user and not accessible to other users. The risk materializes only when `$TMPDIR` is unset or when using a shared `/tmp`.
- **Recommended fix**: If H-1 is fixed (using per-user runtime directory), this becomes informational. Otherwise, add a random suffix to the session directory name.

---

### [Low] L-3: Umask restoration is not thread-safe

- **File**: `internal/agent/protocol.go:87-89`
- **Risk**: `syscall.Umask` is process-global. If multiple goroutines call `ListenAndServe` concurrently (e.g., during tests or in a hypothetical multi-pane scenario within the same process), the umask save/restore sequence is not atomic and could result in a goroutine creating files with an unexpected umask.
- **Current state**:
  ```go
  oldUmask := syscall.Umask(0077)
  ln, err := net.Listen("unix", socketPath)
  syscall.Umask(oldUmask)
  ```
  There is no mutex protecting this sequence.
- **Mitigating factors**: In production, each `gmux-agent` process serves exactly one pane and calls `ListenAndServe` exactly once. The race is theoretically possible in test scenarios only.
- **Recommended fix**: Wrap the umask manipulation in a package-level `sync.Mutex`, or use a different approach such as `chmod` after socket creation.

```go
var umaskMu sync.Mutex

func listenUnixSocket(path string) (net.Listener, error) {
    umaskMu.Lock()
    old := syscall.Umask(0077)
    ln, err := net.Listen("unix", path)
    syscall.Umask(old)
    umaskMu.Unlock()
    return ln, err
}
```

---

### [Info] I-1: Shell command injection via send-keys is by design

- **File**: `internal/tmux/keys.go:59-69`, `internal/tmux/commands.go:150-181`
- **Risk**: The `send-keys` command can inject arbitrary shell commands (e.g., `"; rm -rf /"`) into the target pane's PTY. This is the intended behavior -- it is functionally equivalent to `tmux send-keys`.
- **Current state**: `BuildSendKeysData` concatenates arguments and writes them to the PTY. In non-literal mode, special key names are expanded to escape sequences. There is no shell-level sanitization because the data is written to a PTY, not parsed by a shell interpreter. The shell running in the PTY will interpret the input as if typed by the user.
- **Analysis**: This is not a vulnerability but a design characteristic. The same risk exists in tmux itself. The threat model assumes that only trusted processes (Claude Code) have access to the send-keys mechanism, enforced by socket file permissions. If an attacker can already write to the socket, they can already execute arbitrary commands in the user's shell.
- **Recommendation**: No code change needed. The design rationale is sound. Consider adding a brief comment in `handleSendKeys` noting the trust boundary assumption.

---

### [Info] I-2: Environment variable trust boundary

- **File**: `internal/wrapper/wrapper.go:84-92`, `cmd/gmux-agent/main.go:28-40`
- **Risk**: `GMUX_SESSION`, `GMUX_PANE_ID`, and `GMUX` environment variables are set by the wrapper and read by `gmux-agent` and `gmux` (tmux mode). A process that can modify the child's environment can redirect send-keys to a different pane or session.
- **Current state**: Environment variables are inherited from the parent process. There is no cryptographic binding or signing of these values. `gmux-agent` trusts `GMUX_PANE_ID` and `GMUX_SESSION` at face value.
- **Analysis**: This matches the tmux threat model. tmux itself uses `$TMUX` and `$TMUX_PANE` with the same trust assumptions. The environment is controlled by the parent process, and if the parent is compromised, the child's environment is the least of the concerns. The session name feeds into `SafeSocketPath`, which uses SHA-256 hashing for long names -- this is purely a path-length optimization, not a security measure.
- **Recommendation**: No code change needed. This is an inherent characteristic of the Unix process model.

---

## Recommended Actions (Priority Order)

| Priority | Finding | Action | Effort |
|----------|---------|--------|--------|
| 1 | H-1 | Use per-user runtime directory (`$TMPDIR`/`$XDG_RUNTIME_DIR`) instead of shared `/tmp`. Verify directory ownership after creation. | Medium |
| 2 | M-1 | Add defense-in-depth comment or implement basic escape sequence filtering for PTY writes. | Low-Medium |
| 3 | M-2 | Ensure state directory is created with `0700` permissions in `Save()`. | Low |
| 4 | M-3 | Add design rationale comment about single-pane-ID invariant, or implement flock-based locking. | Low |
| 5 | L-3 | Add mutex around umask manipulation. | Trivial |
| 6 | L-2 | Resolves automatically if H-1 is fixed. | N/A |

---

## Conclusion

The gmux PTY proxy architecture follows standard Unix security practices for a single-user desktop tool. The most significant finding (H-1) relates to the use of a potentially shared `/tmp` directory for Unix sockets, which could enable symlink attacks on multi-user systems. On typical macOS configurations where `$TMPDIR` is per-user, this risk is substantially mitigated -- `os.TempDir()` returns `$TMPDIR` when set, which is the norm on macOS. The risk surfaces when `$TMPDIR` is unset or in non-standard configurations.

The socket permission model (umask 0077, directory 0700) is correctly implemented and prevents unauthorized local users from connecting to the send-keys socket under normal conditions.

The absence of input validation on PTY writes (M-1) is a defense-in-depth gap rather than a direct vulnerability, since the socket access control is the primary security boundary. Implementing basic filtering of dangerous terminal escape sequences would strengthen the security posture.

The state management (panes.json) and wrapper mechanisms are adequate for the intended use case, with minor improvements possible around explicit permission enforcement.

**Overall assessment**: Suitable for single-user desktop use. Before deploying in shared environments (CI servers, multi-user workstations), H-1 should be addressed.

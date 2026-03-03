// Event loop runner for tty-overlay.
// Manages PTY child process, stdin/stdout forwarding, and overlay rendering.

///|
pub struct Runner {
  pty_handle : @ffi.PtyHandle
  master_fd : Int
  child_pid : Int
  tty_handle : @ffi.TtyHandle?
  overlay : @overlay.OverlayState
  config : @args.Config
  content_source : @args.ContentSource
  interval_ms : Int
  mut next_update_ms : Int64 // next content-command refresh time
  mut stdin_open : Bool
  mut prev_rows : Int
  mut prev_cols : Int
}

/// Create a new Runner from a parsed Config.
/// Sets up PTY, spawns child process, configures terminal, and initializes overlay.
pub fn Runner::new(config : @args.Config) -> Runner raise {
  // 1. Get terminal size (fallback to 24x80)
  let (rows, cols) = match @ffi.tty_size(0) {
    Some((r, c)) => (r, c)
    None => (24, 80)
  }
  // 2. Open PTY pair
  let pty_handle = match @ffi.pty_open(cols, rows) {
    Some(h) => h
    None => raise Failure::Failure("pty_open failed")
  }
  // 3. Spawn child process using argv
  let child_pid = match @ffi.pty_spawn_argv(pty_handle, config.command) {
    Some(pid) => pid
    None => {
      let _ = @ffi.pty_close(pty_handle)
      raise Failure::Failure("pty_spawn_argv failed")
    }
  }
  let master_fd = @ffi.pty_master_fd(pty_handle)
  // 4. Setup signals: ignore SIGPIPE, forward SIGWINCH to PTY
  let _ = @ffi.sig_ignore(@ffi.SIGPIPE)
  let _ = @ffi.sig_setup_winch(master_fd)
  // 5. Setup TTY raw mode if stdin is a terminal
  let tty_handle = if @ffi.tty_is_tty(0) {
    match @ffi.tty_save(0) {
      Some(handle) => {
        let _ = @ffi.tty_set_raw(0)
        Some(handle)
      }
      None => None
    }
  } else {
    None
  }
  // 6. Initialize overlay text from content source
  let initial_text = match config.content_source {
    Static(text) => text
    Command(cmd) => strip_trailing_newline(@ffi.exec_capture(cmd))
  }
  let overlay = @overlay.OverlayState::new(
    initial_text,
    rows,
    cols,
    config_width=config.width,
    config_height=config.height,
    vertical=config.vertical,
    horizontal=config.horizontal,
    fg_color=config.fg_color,
    bg_color=config.bg_color,
    position_mode=config.position_mode,
  )
  // 7. Set next update time for command-based content
  let now = @ffi.monotonic_ms()
  let next_update_ms = match config.content_source {
    Command(_) => now + config.interval_ms.to_int64()
    Static(_) => 0L // unused for static content
  }
  // 8. Set up PTY size and scroll region based on position mode
  match config.position_mode {
    Simple => {
      // Simple mode: PTY uses full terminal size, no scroll region
      ()
    }
    Decstbm => {
      // Decstbm mode: resize PTY to child_rows and set DECSTBM scroll region
      let _ = @ffi.pty_resize(pty_handle, cols, overlay.child_rows())
      let _ = @ffi.io_write(1, overlay.scroll_region_set())
      // Move cursor into scroll region so child output starts in the right place
      let scroll_top_row = rows - overlay.child_rows() + 1
      match config.vertical {
        Top(_) => {
          let _ = @ffi.io_write(
            1,
            @ffi.to_utf8(
              "\u001b[" + scroll_top_row.to_string() + ";1H",
            ),
          )
        }
        Bottom(_) => {
          let _ = @ffi.io_write(1, @ffi.to_utf8("\u001b[1;1H"))
        }
      }
    }
    Cell => abort("unreachable: cell mode should be rejected at parse time")
  }
  Runner::{
    pty_handle,
    master_fd,
    child_pid,
    tty_handle,
    overlay,
    config,
    content_source: config.content_source,
    interval_ms: config.interval_ms,
    next_update_ms,
    stdin_open: true,
    prev_rows: rows,
    prev_cols: cols,
  }
}

/// Strip exactly one trailing newline from a string.
/// Command output typically ends with "\n", which causes split_lines to
/// produce an extra empty line. This trims only the last newline.
fn strip_trailing_newline(s : String) -> String {
  if s.has_suffix("\n") {
    try { s[:s.length() - 1].to_string() } catch { _ => s }
  } else {
    s
  }
}

/// Create a sub-Bytes from buf[0..len).
/// Design rationale: MoonBit's Bytes has no sub_bytes/slice method.
/// FixedArray::makei + Bytes::from_array is the idiomatic workaround.
fn bytes_take(buf : Bytes, len : Int) -> Bytes {
  Bytes::from_array(FixedArray::makei(len, fn(i) { buf[i] })[:])
}

/// Concatenate two Bytes values into a single Bytes.
fn bytes_concat(a : Bytes, b : Bytes) -> Bytes {
  let a_len = a.length()
  let b_len = b.length()
  let total = a_len + b_len
  let arr : Array[Byte] = Array::make(total, b'\x00')
  for i = 0; i < a_len; i = i + 1 {
    arr[i] = a[i]
  }
  for i = 0; i < b_len; i = i + 1 {
    arr[a_len + i] = b[i]
  }
  Bytes::from_array(arr[:])
}

/// Poll once for events. Returns false when the loop should exit (child ended).
pub fn Runner::poll_once(self : Runner) -> Bool {
  // 1. Check for terminal size change
  match @ffi.tty_size(0) {
    Some((rows, cols)) =>
      if rows != self.prev_rows || cols != self.prev_cols {
        self.prev_rows = rows
        self.prev_cols = cols
        self.overlay.update_size(rows, cols)
        match self.config.position_mode {
          Simple => {
            // Simple mode: resize PTY to full terminal size
            let _ = @ffi.pty_resize(self.pty_handle, cols, rows)
          }
          Decstbm => {
            // Decstbm mode: resize PTY to child_rows (terminal height minus overlay)
            let _ = @ffi.pty_resize(
              self.pty_handle, cols, self.overlay.child_rows(),
            )
            // Re-set DECSTBM scroll region for new terminal dimensions
            let _ = @ffi.io_write(1, self.overlay.scroll_region_set())
          }
          Cell =>
            abort("unreachable: cell mode should be rejected at parse time")
        }
      }
    None => ()
  }
  // 2. Content-command timer check
  match self.content_source {
    Command(cmd) => {
      let now = @ffi.monotonic_ms()
      if now >= self.next_update_ms {
        let text = strip_trailing_newline(@ffi.exec_capture(cmd))
        self.overlay.update_text(text)
        self.next_update_ms = @ffi.monotonic_ms() + self.interval_ms.to_int64()
        // Re-render overlay immediately so update is visible even without child output
        let overlay_bytes = self.overlay.render()
        let _ = @ffi.io_write(1, overlay_bytes)
      }
    }
    Static(_) => ()
  }
  // 3. Compute poll timeout
  let timeout = match self.content_source {
    Static(_) => -1 // infinite wait
    Command(_) => {
      let remaining = self.next_update_ms - @ffi.monotonic_ms()
      if remaining <= 0L { 0 } else { remaining.to_int() }
    }
  }
  // 4. Build pollfd array
  let entries : Array[(Int, Int)] = []
  if self.stdin_open {
    entries.push((0, @ffi.POLLIN))
  } else {
    entries.push((-1, 0)) // skip stdin
  }
  entries.push((self.master_fd, @ffi.POLLIN))
  let poll_fds = @ffi.pollfd_pack(entries)
  let nfds = entries.length()
  // 5. Poll
  let ready = @ffi.io_poll(poll_fds, nfds, timeout)
  if ready < 0 {
    return false // poll error
  }
  // 6. Handle stdin
  if self.stdin_open {
    let stdin_revents = @ffi.pollfd_revents(poll_fds, 0)
    if (stdin_revents & @ffi.POLLIN) != 0 {
      let buf = Bytes::make(4096, 0)
      let n = @ffi.io_read(0, buf, 4096)
      if n > 0 {
        let data = bytes_take(buf, n)
        let _ = @ffi.io_write(self.master_fd, data)

      } else {
        self.stdin_open = false
      }
    }
    if (stdin_revents & (@ffi.POLLHUP | @ffi.POLLERR)) != 0 {
      self.stdin_open = false
    }
  }
  // 7. Handle master_fd
  let master_idx = if self.stdin_open { 1 } else { 1 }
  let master_revents = @ffi.pollfd_revents(poll_fds, master_idx)
  if (master_revents & @ffi.POLLIN) != 0 {
    let buf = Bytes::make(4096, 0)
    let n = @ffi.io_read(self.master_fd, buf, 4096)
    if n > 0 {
      let child_output = bytes_take(buf, n)
      let overlay_bytes = self.overlay.render()
      // Combine child output + overlay into a single write
      let combined = bytes_concat(child_output, overlay_bytes)
      let _ = @ffi.io_write(1, combined)

    }
  }
  if (master_revents & (@ffi.POLLHUP | @ffi.POLLERR)) != 0 {
    return false // child process ended
  }
  true
}

/// Run the event loop until the child process exits.
/// Returns the child process exit status.
pub fn Runner::run(self : Runner) -> Int {
  while self.poll_once() {

  }
  self.cleanup()
  match @ffi.proc_waitpid(self.child_pid, false) {
    Some((_, status)) => status
    None => 1
  }
}

/// Release all resources: reset scroll region, restore terminal, close PTY.
fn Runner::cleanup(self : Runner) -> Unit {
  match self.config.position_mode {
    Simple => {
      // Simple mode: no scroll region to reset, no cursor move needed
      ()
    }
    Decstbm => {
      // Reset DECSTBM scroll region to full screen
      let _ = @ffi.io_write(1, self.overlay.scroll_region_reset())
      // Move cursor to last row and print newline so shell prompt appears correctly
      let _ = @ffi.io_write(
        1,
        @ffi.to_utf8(
          "\u001b[" + self.prev_rows.to_string() + ";1H\n",
        ),
      )
    }
    Cell => abort("unreachable: cell mode should be rejected at parse time")
  }
  match self.tty_handle {
    Some(h) => {
      let _ = @ffi.tty_restore(h)

    }
    None => ()
  }
  let _ = @ffi.pty_close(self.pty_handle)

}

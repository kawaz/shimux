///|
pub struct PtyHandle(Int)

///|
extern "C" fn shimux_pty_open(cols : Int, rows : Int) -> Int = "shimux_pty_open"

///|
#borrow(cmd)
extern "C" fn shimux_pty_spawn(handle : Int, cmd : Bytes) -> Int = "shimux_pty_spawn"

///|
extern "C" fn shimux_pty_master_fd(handle : Int) -> Int = "shimux_pty_master_fd"

///|
extern "C" fn shimux_pty_resize(handle : Int, cols : Int, rows : Int) -> Int = "shimux_pty_resize"

///|
extern "C" fn shimux_pty_close(handle : Int) -> Int = "shimux_pty_close"

///|
pub fn pty_open(cols : Int, rows : Int) -> PtyHandle? {
  let h = shimux_pty_open(cols, rows)
  if h < 0 { None } else { Some(PtyHandle(h)) }
}

///|
pub fn pty_spawn(handle : PtyHandle, cmd : Bytes) -> Int? {
  let PtyHandle(h) = handle
  let pid = shimux_pty_spawn(h, cmd)
  if pid < 0 { None } else { Some(pid) }
}

///|
pub fn pty_master_fd(handle : PtyHandle) -> Int {
  let PtyHandle(h) = handle
  shimux_pty_master_fd(h)
}

///|
pub fn pty_resize(handle : PtyHandle, cols : Int, rows : Int) -> Bool {
  let PtyHandle(h) = handle
  shimux_pty_resize(h, cols, rows) == 0
}

///|
pub fn pty_close(handle : PtyHandle) -> Bool {
  let PtyHandle(h) = handle
  shimux_pty_close(h) == 0
}

///|
#borrow(argv_bytes)
extern "C" fn shimux_pty_spawn_argv(
  handle : Int,
  argc : Int,
  argv_bytes : Bytes
) -> Int = "shimux_pty_spawn_argv"

/// Spawn a child process on the PTY with an explicit argv array.
/// Each string in `args` becomes an argv element. The first element is the
/// executable path. Returns the child PID on success, None on failure.
pub fn pty_spawn_argv(handle : PtyHandle, args : Array[String]) -> Int? {
  if args.length() == 0 {
    return None
  }
  let PtyHandle(h) = handle
  // Build a single Bytes buffer with null-terminated strings concatenated
  let parts : Array[Byte] = []
  for i = 0; i < args.length(); i = i + 1 {
    let utf8 = to_utf8(args[i])
    for j = 0; j < utf8.length(); j = j + 1 {
      parts.push(utf8[j])
    }
    parts.push(b'\x00')
  }
  let argv_bytes = Bytes::from_array(parts[:])
  let pid = shimux_pty_spawn_argv(h, args.length(), argv_bytes)
  if pid < 0 {
    None
  } else {
    Some(pid)
  }
}

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

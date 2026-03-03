///|
pub struct TtyHandle(Int)

///|
extern "C" fn shimux_tty_is_tty(fd : Int) -> Int = "shimux_tty_is_tty"

///|
extern "C" fn shimux_tty_size(fd : Int) -> Int64 = "shimux_tty_size"

///|
extern "C" fn shimux_tty_save(fd : Int) -> Int = "shimux_tty_save"

///|
extern "C" fn shimux_tty_set_raw(fd : Int) -> Int = "shimux_tty_set_raw"

///|
extern "C" fn shimux_tty_restore(handle : Int) -> Int = "shimux_tty_restore"

///|
pub fn tty_is_tty(fd : Int) -> Bool {
  shimux_tty_is_tty(fd) == 1
}

///|
pub fn tty_size(fd : Int) -> (Int, Int)? {
  let result = shimux_tty_size(fd)
  if result < 0L { return None }
  let rows = ((result >> 16) & 0xFFFF_L).to_int()
  let cols = (result & 0xFFFF_L).to_int()
  Some((rows, cols))
}

///|
pub fn tty_save(fd : Int) -> TtyHandle? {
  let h = shimux_tty_save(fd)
  if h < 0 { None } else { Some(TtyHandle(h)) }
}

///|
pub fn tty_set_raw(fd : Int) -> Bool {
  shimux_tty_set_raw(fd) == 0
}

///|
pub fn tty_restore(handle : TtyHandle) -> Bool {
  let TtyHandle(h) = handle
  shimux_tty_restore(h) == 0
}

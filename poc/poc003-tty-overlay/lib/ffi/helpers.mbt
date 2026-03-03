/// Convert a MoonBit String to UTF-8 encoded Bytes.
/// Design rationale: String.to_bytes() produces UTF-16 LE, but the Rust FFI
/// functions expect UTF-8. Using @utf8.encode() produces correct UTF-8 Bytes.
pub fn to_utf8(s : String) -> Bytes {
  @utf8.encode(s)
}

///|
#borrow(name)
extern "C" fn shimux_getenv(name : Bytes) -> Bytes = "shimux_getenv"

///|
extern "C" fn shimux_getpid() -> Int = "shimux_getpid"

///|
extern "C" fn shimux_getuid() -> UInt = "shimux_getuid"

///|
#borrow(path)
extern "C" fn shimux_ensure_dir(path : Bytes) -> Int = "shimux_ensure_dir"

///|
pub fn getenv(name : String) -> String {
  let result = shimux_getenv(to_utf8(name))
  if result.length() == 0 {
    ""
  } else {
    @utf8.decode_lossy(result[:])
  }
}

///|
pub fn getpid() -> Int {
  shimux_getpid()
}

///|
pub fn getuid() -> UInt {
  shimux_getuid()
}

///|
pub fn ensure_dir(path : String) -> Bool {
  shimux_ensure_dir(to_utf8(path)) == 0
}

///|
extern "C" fn shimux_argc() -> Int = "shimux_argc"

///|
extern "C" fn shimux_argv(index : Int) -> Bytes = "shimux_argv"

///|
extern "C" fn shimux_monotonic_ms() -> Int64 = "shimux_monotonic_ms"

///|
pub fn argc() -> Int {
  shimux_argc()
}

///|
pub fn argv(index : Int) -> String? {
  let result = shimux_argv(index)
  // shimux_argv returns null (empty Bytes with length 0) when index is out of bounds
  if result.length() == 0 {
    None
  } else {
    Some(@utf8.decode_lossy(result[:]))
  }
}

///|
pub fn monotonic_ms() -> Int64 {
  shimux_monotonic_ms()
}

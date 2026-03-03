///|
#borrow(cmd)
extern "C" fn shimux_exec_capture(cmd : Bytes) -> Bytes = "shimux_exec_capture"

/// Run a shell command and return its stdout output as a String.
pub fn exec_capture(cmd : String) -> String {
  let result = shimux_exec_capture(to_utf8(cmd))
  if result.length() == 0 {
    ""
  } else {
    @utf8.decode_lossy(result[:])
  }
}

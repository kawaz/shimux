///|
pub struct SockFd(Int)

///|
#borrow(path)
extern "C" fn shimux_sock_listen(path : Bytes) -> Int = "shimux_sock_listen"

///|
extern "C" fn shimux_sock_accept(fd : Int) -> Int = "shimux_sock_accept"

///|
#borrow(path)
extern "C" fn shimux_sock_connect(path : Bytes) -> Int = "shimux_sock_connect"

///|
#borrow(path)
extern "C" fn shimux_sock_unlink(path : Bytes) -> Int = "shimux_sock_unlink"

///|
pub fn sock_listen(path : Bytes) -> SockFd? {
  let fd = shimux_sock_listen(path)
  if fd < 0 { None } else { Some(SockFd(fd)) }
}

///|
pub fn sock_accept(sock : SockFd) -> SockFd? {
  let SockFd(fd) = sock
  let client = shimux_sock_accept(fd)
  if client < 0 { None } else { Some(SockFd(client)) }
}

///|
pub fn sock_connect(path : Bytes) -> SockFd? {
  let fd = shimux_sock_connect(path)
  if fd < 0 { None } else { Some(SockFd(fd)) }
}

///|
pub fn sock_unlink(path : Bytes) -> Bool {
  shimux_sock_unlink(path) == 0
}

///|
pub fn sock_close(sock : SockFd) -> Bool {
  let SockFd(fd) = sock
  io_close(fd)
}

///|
pub fn sock_set_nonblocking(sock : SockFd) -> Bool {
  let SockFd(fd) = sock
  io_set_nonblocking(fd)
}

///|
pub fn sock_fd_raw(sock : SockFd) -> Int {
  let SockFd(fd) = sock
  fd
}

# 各プロセスのfd・IOの対応

---

**kawaz:** shimux-server/shimux-client/childcmdの3者が持つfd(tty?pty?)のioがそれぞれどのfdのioと対応するのかの図または表を書いて。

**Claude:**

## fd対応表

### 各プロセスが持つfd

| プロセス | fd | 種別 | 接続先 |
|---|---|---|---|
| shimux-server | `master_fd` | PTY master | ↕ child cmd (PTY slave経由) |
| shimux-server | `server_fd` | Unix socket (listen) | ← shimux-client が connect |
| shimux-server | `client_fd` | Unix socket (accepted) | ↕ shimux-client |
| shimux-client | `socket_fd` | Unix socket | ↕ shimux-server |
| shimux-client | `stdin` (fd=0) | TTY | ← Ghostty のキー入力 |
| shimux-client | `stdout` (fd=1) | TTY | → Ghostty への表示 |
| child cmd ($SHELL) | `stdin` (fd=0) | PTY slave | ← shimux-server master_fd 経由 |
| child cmd ($SHELL) | `stdout` (fd=1) | PTY slave | → shimux-server master_fd 経由 |
| child cmd ($SHELL) | `stderr` (fd=2) | PTY slave | → shimux-server master_fd 経由 |

---

### データフロー図

```
[Ghostty]
  keyboard input
      ↓ (TTY)
[shimux-client]
  stdin(fd=0) でread
      ↓
  socket_fd にwrite
      ↓ (Unix domain socket)
[shimux-server]
  client_fd でread          ← on_input(bytes) → hooks
      ↓
  master_fd にwrite
      ↓ (PTY)
[child cmd / $SHELL]
  stdin(fd=0) でread
  （処理）
  stdout(fd=1) にwrite
      ↓ (PTY)
[shimux-server]
  master_fd でread          ← on_output(bytes) → hooks → broadcast
      ↓
  client_fd にwrite
      ↓ (Unix domain socket)
[shimux-client]
  socket_fd でread
      ↓
  stdout(fd=1) にwrite
      ↓ (TTY)
[Ghostty]
  表示
```

---

### PTYのmaster/slaveの関係

```
openpty() で生成されるペア：

master_fd  ←→  slave_fd
（server が持つ）  （fork後childのstdin/out/errになる）

master_fd への write = child の stdin に届く
master_fd の read   = child の stdout/stderr が読める

child側から見ると slave_fd は普通のTTYと区別がつかない
```

`master_fd` 1本が child の stdin/stdout/stderr **全部**に対応しているのがPTYの特徴です。

---

### イベントループとの対応（補足）

```moonbit
loop {
  let events = epoll_wait(epfd)
  for event in events {
    match event.fd {
      fd if fd == master_fd =>
        // master_fdから来た = shell→Ghostty = Output
        let bytes = read(master_fd)
        pipeline.on_output(bytes)
        // on_output内でclientsにbroadcast済み

      fd if fd == client_fd =>
        // client_fdから来た = Ghostty→shell = Input
        let bytes = read(client_fd)
        let bytes = pipeline.on_input(bytes)
        write(master_fd, bytes)  // shellに流す

      fd if fd == server_fd =>
        // 新しいclientのattach
        pipeline.add_client(accept(server_fd))

      _ => ()
    }
  }
}
```

**fdの素性がそのままDirectionを意味している**ため、イベントループが自然にon_input/on_outputを呼び分けるだけで入出力の区別が完了します。

---

### Pipeline内部の処理

```moonbit
// on_outputはbroadcastも内包する透過プロキシ
fn Pipeline::on_output(self, bytes : Bytes) -> Unit {
  let bytes = self.hooks.fold(bytes, fn(b, h) { h.on_bytes(Output, b) })
  self.observer.on_output(bytes)              // 画面状態更新（将来用）
  self.log.write(Output, bytes)               // ログ
  self.clients.each(fn(c) { send(c, bytes) }) // clientへbroadcast ← ここでGhosttyに届く
}

fn Pipeline::on_input(self, bytes : Bytes) -> Bytes {
  // hooksを通すだけ、observerには渡さない
  self.hooks.fold(bytes, fn(b, h) { h.on_bytes(Input, b) })
}
```

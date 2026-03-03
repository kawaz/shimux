---
created_at: "2026-03-04"
---

# TTY / PTY / PTS — 略称・歴史・デバイスファイル

Context: poc003-tty-overlay 実装中の議論から

## 略称

| 略称 | 正式名称 | 意味 |
|------|----------|------|
| **TTY** | **T**ele**TY**pewriter | 物理端末の総称。Unix では端末デバイス全般を指す |
| **PTY** | **P**seudo-**T**ele**TY**pewriter | ソフトウェアで実現された仮想端末ペア |
| **PTS** | **P**seudo **T**erminal **S**lave | PTY のスレーブ（replica）側デバイス |

POSIX Issue 8 (2024) から slave → **subsidiary** に改称の動きあり。macOS man では既に **primary/replica** を使用。

## 視点による区別

```
自分が今使っている端末 → TTY
子プロセスに渡す仮想端末 → PTY
子プロセスから見た PTY  → （そのプロセスにとっては）ただの TTY
```

shimux のような入れ子構造では PTY の中に PTY を作ることになる。各レイヤから見れば自分の端末は常に TTY。

## 2つの PTY 実装方式

### BSD 方式（macOS が今も使用）

```
/dev/ptyXY  ← master（プログラムが open）
/dev/ttyXY  ← slave（子プロセスの端末）
```

- 1対1の静的ペア。事前に MAKEDEV で作成
- 空きを探すためにループで open を試行 → レースコンディションの原因
- macOS: `[p-w]` × `[0-9a-f]` = 8 × 16 = **128 ペア**

### Unix98/SysV 方式（Linux 標準）

```
/dev/ptmx    ← multiplexer（open すると master fd を動的取得）
/dev/pts/N   ← slave（番号は動的割り当て）
```

- `posix_openpt()` → `grantpt()` → `unlockpt()` → `ptsname()` の API
- devpts 仮想ファイルシステムでオンデマンド生成/削除
- ペア数は実質無制限、レースコンディションなし
- macOS も `/dev/ptmx` をサポートしており、実際のターミナルエミュレータはこちらを使用

| 項目 | BSD 方式 | Unix98 方式 |
|------|----------|-------------|
| Master | `/dev/ptyXY`（個別） | `/dev/ptmx`（単一 multiplexer） |
| Slave | `/dev/ttyXY`（事前作成） | `/dev/pts/N`（動的生成） |
| ペア数上限 | 固定（128〜256） | 実質無制限 |
| レースコンディション | あり | なし |
| 初出 | 4.2BSD (1983) | AT&T System V → Unix98 標準化 |
| 現在 | deprecated（後方互換） | 標準 |

## `[p-w]` の由来 — MAKEDEV の名前空間

4.2BSD 時代、`/dev/tty` の名前空間はハードウェア種別ごとにアルファベットで区分されていた。

| ハードウェア | 割当文字 | 用途 |
|-------------|----------|------|
| DZ11/DZ32 | 数字のみ (`tty00`-`tty77`) | UNIBUS 端末マルチプレクサ |
| DH11/DMF32 | **h - o** | UNIBUS 端末マルチプレクサ |
| DHU11 | **S - Z**（大文字） | 端末マルチプレクサ |
| **PTY** | **p - u**（初期） | 疑似端末 |

物理端末デバイスが `h-o` を使っていたため、疑似端末はその次の **`p`** から始まった。
`p` = pseudo の頭文字という解釈もあるが、実際にはアルファベット順で空いている領域の割り当て結果（偶然の一致とも言える）。

必要に応じて `q, r, s, ... w` と拡張。他の BSD 実装では `[p-za-e]`（最大 256 ペア）まで行くものもある。

## macOS の tty と pty の数の違い

```
/dev/pty*  = 128 個（ptyp0〜ptyw15 = 8 グループ × 16）
/dev/tty*  = 148 個
```

差の 20 個は PTY ペアではない tty デバイス:

- `/dev/tty` — プロセスの制御端末を指す特殊ファイル
- `/dev/tty.Bluetooth-*` — Bluetooth シリアルポート
- `/dev/tty.usbmodem*` 等 — USB シリアルデバイス

PTY ペアとしては **128:128 で完全に 1 対 1**。

## shimux への示唆

- shimux は `posix_openpt()` (Unix98 方式) を使うのが正しい
  - poc002/poc003 の `ffi/src/pty.rs` は `posix_openpt` + `grantpt` + `unlockpt` + `ptsname` を使用中
- BSD 方式のデバイスファイルは後方互換のためだけに存在するので気にしなくてよい

## 参考

- [pty(7) - Linux manual page](https://man7.org/linux/man-pages/man7/pty.7.html)
- [The TTY demystified](https://www.linusakesson.net/programming/tty/)
- [4.2BSD MAKEDEV (TUHS Archive)](https://www.tuhs.org/cgi-bin/utree.pl?file=4.2BSD/dev/MAKEDEV)
- [A history of the tty](https://computer.rip/2024-02-25-a-history-of-the-tty.html)

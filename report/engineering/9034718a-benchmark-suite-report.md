# Benchmark Suite Report

## Task: 9034718a-benchmark-suite

## Summary

gmux 全コンポーネントのパフォーマンスベンチマークスイートを構築。66 benchmarks、全 PASS。

## Created Files (6)

| File | Benchmarks | Coverage |
|------|-----------|----------|
| `internal/tmux/parser_bench_test.go` | 15 | Parse() 全コマンドパターン |
| `internal/tmux/format_bench_test.go` | 8 | ExpandFormat() 手書きスキャナ性能 |
| `internal/tmux/keys_bench_test.go` | 11 | ExpandSpecialKey() + BuildSendKeysData() |
| `internal/tmux/commands_bench_test.go` | 10 | Execute() + ExtractPaneID() |
| `internal/agent/agent_bench_test.go` | 9 | ソケット通信 RT + SafeSocketPath |
| `internal/pane/manager_bench_test.go` | 25 | CRUD + Save/Load (1/10/50 scaling) |

## Key Performance Baselines (Apple M1 Max)

| Operation | Latency | Allocs |
|-----------|---------|--------|
| Parse(simple) | 26ns | 1 |
| ExpandFormat(single var) | 29ns | 1 |
| ExpandSpecialKey(Enter) | 7.5ns | 0 |
| Execute(has-session) | 21ns | 1 |
| Execute(list-panes, 10) | 1.2us | 22 |
| SendKeys RT (hello) | 39us | 42 |
| SendKeys RT (64KB) | 473us | 49 |
| PaneManager.Get() | 14ns | 0 (O(1)) |
| PaneManager.List(50) | 520ns | 1 |
| PaneManager.Save(50) | 252us | 14 |

## Run Command

```bash
go test -bench=. -benchmem ./...
```

## Commit

```
8baef88 perf: ベンチマークスイート構築 (66 benchmarks)
```

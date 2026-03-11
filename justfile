# shimux - MoonBit native + Rust FFI

# Default: check + test
default: check test

# Format MoonBit code
fmt:
    moon fmt

# Type check
check:
    moon check --deny-warn

# Run tests
test:
    moon test --target native

# Update snapshot tests
test-update:
    moon test --target native --update

# Build Rust FFI
ffi:
    (cd ffi && cargo build --release)

# Build nosuspend (Ctrl+Z auto-resume wrapper, MoonBit native)
nosuspend:
    moon build cmd/nosuspend --target native

# Build shimux binary (FFI + native)
build: ffi nosuspend
    moon build cmd/shimux --target native

# Build release binary
build-release: ffi
    moon build cmd/shimux --target native --release

# Run shimux with arguments (e.g. just run -- claude --tmux --worktree)
run *args: build
    _build/native/debug/build/cmd/shimux/shimux.exe {{args}}

# Clean all build artifacts
clean:
    moon clean
    (cd ffi && cargo clean)

# Generate type definition files (.mbti)
info:
    moon info

# Pre-release check
release-check: fmt info check test build

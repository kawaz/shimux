package tmux

import "testing"

// --- ExpandSpecialKey benchmarks ---

func BenchmarkExpandSpecialKey_Enter(b *testing.B) {
	for b.Loop() {
		ExpandSpecialKey("Enter")
	}
}

func BenchmarkExpandSpecialKey_ControlKey(b *testing.B) {
	for b.Loop() {
		ExpandSpecialKey("C-c")
	}
}

func BenchmarkExpandSpecialKey_FunctionKey(b *testing.B) {
	for b.Loop() {
		ExpandSpecialKey("F12")
	}
}

func BenchmarkExpandSpecialKey_PlainText(b *testing.B) {
	for b.Loop() {
		ExpandSpecialKey("hello")
	}
}

func BenchmarkExpandSpecialKey_ArrowKeys(b *testing.B) {
	keys := []string{"Up", "Down", "Left", "Right"}
	for b.Loop() {
		for _, k := range keys {
			ExpandSpecialKey(k)
		}
	}
}

// --- BuildSendKeysData benchmarks ---

func BenchmarkBuildSendKeysData_SimpleText(b *testing.B) {
	args := []string{"hello", " ", "world"}
	for b.Loop() {
		BuildSendKeysData(args, false)
	}
}

func BenchmarkBuildSendKeysData_WithSpecialKeys(b *testing.B) {
	args := []string{"ls", " ", "-la", "Enter"}
	for b.Loop() {
		BuildSendKeysData(args, false)
	}
}

func BenchmarkBuildSendKeysData_MixedKeys(b *testing.B) {
	args := []string{"C-c", "hello", "Enter", "Tab", "world", "Escape", "Up", "Down"}
	for b.Loop() {
		BuildSendKeysData(args, false)
	}
}

func BenchmarkBuildSendKeysData_Literal(b *testing.B) {
	args := []string{"Enter", "Tab", "C-c", "hello"}
	for b.Loop() {
		BuildSendKeysData(args, true)
	}
}

func BenchmarkBuildSendKeysData_ManyArgs(b *testing.B) {
	args := make([]string, 100)
	for i := range args {
		if i%3 == 0 {
			args[i] = "Enter"
		} else if i%3 == 1 {
			args[i] = "C-a"
		} else {
			args[i] = "text"
		}
	}
	b.ResetTimer()
	for b.Loop() {
		BuildSendKeysData(args, false)
	}
}

func BenchmarkBuildSendKeysData_AllSpecialKeys(b *testing.B) {
	args := []string{
		"Enter", "Escape", "Tab", "Space", "BSpace", "BTab",
		"Up", "Down", "Right", "Left",
		"DC", "End", "Home", "IC", "NPage", "PPage",
		"F1", "F2", "F3", "F4", "F5", "F6",
		"F7", "F8", "F9", "F10", "F11", "F12",
	}
	for b.Loop() {
		BuildSendKeysData(args, false)
	}
}

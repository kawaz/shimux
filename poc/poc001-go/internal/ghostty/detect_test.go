package ghostty

import (
	"runtime"
	"testing"
)

func TestIsInsideGhostty(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		// TC-GC004a
		{"TERM_PROGRAM=ghostty", map[string]string{"TERM_PROGRAM": "ghostty"}, true},
		// TC-GC004b
		{"GHOSTTY_RESOURCES_DIR set", map[string]string{"GHOSTTY_RESOURCES_DIR": "/path"}, true},
		// TC-GC004c
		{"both set", map[string]string{"TERM_PROGRAM": "ghostty", "GHOSTTY_RESOURCES_DIR": "/path"}, true},
		// TC-GC004d
		{"neither set", map[string]string{}, false},
		// TC-GC004e
		{"TERM_PROGRAM=iterm2", map[string]string{"TERM_PROGRAM": "iTerm.app"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 既存の環境変数をクリア
			t.Setenv("TERM_PROGRAM", "")
			t.Setenv("GHOSTTY_RESOURCES_DIR", "")
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			got := IsInsideGhostty()
			if got != tt.expected {
				t.Errorf("IsInsideGhostty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGhosttyVersion(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			"ghostty with version",
			map[string]string{"TERM_PROGRAM": "ghostty", "TERM_PROGRAM_VERSION": "1.1.0"},
			"1.1.0",
		},
		{
			"ghostty with pre-release version",
			map[string]string{"TERM_PROGRAM": "ghostty", "TERM_PROGRAM_VERSION": "1.2.0-dev+abc123"},
			"1.2.0-dev+abc123",
		},
		{
			"ghostty detected by GHOSTTY_RESOURCES_DIR, version set",
			map[string]string{"GHOSTTY_RESOURCES_DIR": "/usr/share/ghostty", "TERM_PROGRAM_VERSION": "1.0.0"},
			"1.0.0",
		},
		{
			"ghostty with empty version",
			map[string]string{"TERM_PROGRAM": "ghostty", "TERM_PROGRAM_VERSION": ""},
			"",
		},
		{
			"ghostty without version env",
			map[string]string{"TERM_PROGRAM": "ghostty"},
			"",
		},
		{
			"not ghostty, version set but ignored",
			map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "3.4.0"},
			"",
		},
		{
			"no terminal detected",
			map[string]string{},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TERM_PROGRAM", "")
			t.Setenv("TERM_PROGRAM_VERSION", "")
			t.Setenv("GHOSTTY_RESOURCES_DIR", "")
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			got := GhosttyVersion()
			if got != tt.expected {
				t.Errorf("GhosttyVersion() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsNestedTerminal(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			"TMUX set",
			map[string]string{"TMUX": "/tmp/tmux-501/default,12345,0"},
			true,
		},
		{
			"TMUX empty",
			map[string]string{"TMUX": ""},
			false,
		},
		{
			"TMUX not set",
			map[string]string{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TMUX", "")
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			got := IsNestedTerminal()
			if got != tt.expected {
				t.Errorf("IsNestedTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDetectEnvironment(t *testing.T) {
	// 全テストで使用する環境変数をクリアするヘルパー
	clearEnv := func(t *testing.T) {
		t.Helper()
		t.Setenv("TERM_PROGRAM", "")
		t.Setenv("TERM_PROGRAM_VERSION", "")
		t.Setenv("GHOSTTY_RESOURCES_DIR", "")
		t.Setenv("TMUX", "")
		t.Setenv("SHIMUX", "")
		t.Setenv("SHIMUX_SESSION", "")
	}

	tests := []struct {
		name     string
		envVars  map[string]string
		expected EnvironmentInfo
	}{
		{
			"full ghostty environment with shimux active",
			map[string]string{
				"TERM_PROGRAM":          "ghostty",
				"TERM_PROGRAM_VERSION":  "1.1.0",
				"GHOSTTY_RESOURCES_DIR": "/usr/share/ghostty",
				"SHIMUX":                  "1",
				"SHIMUX_SESSION":          "my-session",
			},
			EnvironmentInfo{
				InsideGhostty:  true,
				GhosttyVersion: "1.1.0",
				NestedTerminal: false,
				OS:             runtime.GOOS,
				Arch:           runtime.GOARCH,
				ShimuxSession:    "my-session",
				ShimuxActive:     true,
				ResourcesDir:   "/usr/share/ghostty",
			},
		},
		{
			"ghostty with tmux nested",
			map[string]string{
				"TERM_PROGRAM":         "ghostty",
				"TERM_PROGRAM_VERSION": "1.0.0",
				"TMUX":                 "/tmp/tmux-501/default,12345,0",
			},
			EnvironmentInfo{
				InsideGhostty:  true,
				GhosttyVersion: "1.0.0",
				NestedTerminal: true,
				OS:             runtime.GOOS,
				Arch:           runtime.GOARCH,
				ShimuxSession:    "",
				ShimuxActive:     false,
				ResourcesDir:   "",
			},
		},
		{
			"not ghostty, no shimux",
			map[string]string{},
			EnvironmentInfo{
				InsideGhostty:  false,
				GhosttyVersion: "",
				NestedTerminal: false,
				OS:             runtime.GOOS,
				Arch:           runtime.GOARCH,
				ShimuxSession:    "",
				ShimuxActive:     false,
				ResourcesDir:   "",
			},
		},
		{
			"not ghostty but shimux active",
			map[string]string{
				"SHIMUX":         "1",
				"SHIMUX_SESSION": "fallback",
			},
			EnvironmentInfo{
				InsideGhostty:  false,
				GhosttyVersion: "",
				NestedTerminal: false,
				OS:             runtime.GOOS,
				Arch:           runtime.GOARCH,
				ShimuxSession:    "fallback",
				ShimuxActive:     true,
				ResourcesDir:   "",
			},
		},
		{
			"ghostty detected by resources dir only",
			map[string]string{
				"GHOSTTY_RESOURCES_DIR": "/opt/ghostty/resources",
			},
			EnvironmentInfo{
				InsideGhostty:  true,
				GhosttyVersion: "",
				NestedTerminal: false,
				OS:             runtime.GOOS,
				Arch:           runtime.GOARCH,
				ShimuxSession:    "",
				ShimuxActive:     false,
				ResourcesDir:   "/opt/ghostty/resources",
			},
		},
		{
			"ghostty with shimux and tmux nested",
			map[string]string{
				"TERM_PROGRAM":          "ghostty",
				"TERM_PROGRAM_VERSION":  "1.1.0",
				"GHOSTTY_RESOURCES_DIR": "/usr/share/ghostty",
				"TMUX":                  "/tmp/tmux-501/default,99999,0",
				"SHIMUX":                  "1",
				"SHIMUX_SESSION":          "dev",
			},
			EnvironmentInfo{
				InsideGhostty:  true,
				GhosttyVersion: "1.1.0",
				NestedTerminal: true,
				OS:             runtime.GOOS,
				Arch:           runtime.GOARCH,
				ShimuxSession:    "dev",
				ShimuxActive:     true,
				ResourcesDir:   "/usr/share/ghostty",
			},
		},
		{
			"SHIMUX set to non-1 value",
			map[string]string{
				"TERM_PROGRAM": "ghostty",
				"SHIMUX":         "0",
			},
			EnvironmentInfo{
				InsideGhostty:  true,
				GhosttyVersion: "",
				NestedTerminal: false,
				OS:             runtime.GOOS,
				Arch:           runtime.GOARCH,
				ShimuxSession:    "",
				ShimuxActive:     false,
				ResourcesDir:   "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			got := DetectEnvironment()
			if got != tt.expected {
				t.Errorf("DetectEnvironment() =\n  %+v\nwant\n  %+v", got, tt.expected)
			}
		})
	}
}

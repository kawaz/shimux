package ghostty

import (
	"os"
	"runtime"
)

// IsInsideGhostty はGhosttyターミナル内で実行されているかを判定する。
func IsInsideGhostty() bool {
	return os.Getenv("TERM_PROGRAM") == "ghostty" || os.Getenv("GHOSTTY_RESOURCES_DIR") != ""
}

// GhosttyVersion は TERM_PROGRAM_VERSION からバージョンを返す。
// Ghostty 環境外では空文字を返す。
func GhosttyVersion() string {
	if !IsInsideGhostty() {
		return ""
	}
	return os.Getenv("TERM_PROGRAM_VERSION")
}

// IsNestedTerminal は Ghostty がネストされた環境（tmux内等）かを判定。
// TMUX 環境変数の存在でネスト判定。
func IsNestedTerminal() bool {
	return os.Getenv("TMUX") != ""
}

// EnvironmentInfo は実行環境の検出結果をまとめた構造体。
type EnvironmentInfo struct {
	InsideGhostty  bool   `json:"inside_ghostty"`
	GhosttyVersion string `json:"ghostty_version,omitempty"`
	NestedTerminal bool   `json:"nested_terminal"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	ShimuxSession    string `json:"shimux_session,omitempty"`
	ShimuxActive     bool   `json:"shimux_active"`
	ResourcesDir   string `json:"resources_dir,omitempty"`
}

// DetectEnvironment は現在の環境情報を検出して返す。
func DetectEnvironment() EnvironmentInfo {
	return EnvironmentInfo{
		InsideGhostty:  IsInsideGhostty(),
		GhosttyVersion: GhosttyVersion(),
		NestedTerminal: IsNestedTerminal(),
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		ShimuxSession:    os.Getenv("SHIMUX_SESSION"),
		ShimuxActive:     os.Getenv("SHIMUX") == "1",
		ResourcesDir:   os.Getenv("GHOSTTY_RESOURCES_DIR"),
	}
}

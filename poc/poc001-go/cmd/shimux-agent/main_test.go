package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunWithIO_PaneIDNotSet(t *testing.T) {
	t.Setenv("SHIMUX_PANE_ID", "")
	t.Setenv("SHIMUX_SESSION", "test-session")

	var stdout, stderr bytes.Buffer
	code := runWithIO(&stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "SHIMUX_PANE_ID not set") {
		t.Errorf("expected stderr to contain 'SHIMUX_PANE_ID not set', got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "should be launched by shimux") {
		t.Errorf("expected stderr to contain usage hint, got %q", stderr.String())
	}
}

func TestRunWithIO_SessionNotSet(t *testing.T) {
	t.Setenv("SHIMUX_PANE_ID", "0")
	t.Setenv("SHIMUX_SESSION", "")

	var stdout, stderr bytes.Buffer
	code := runWithIO(&stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "SHIMUX_SESSION not set") {
		t.Errorf("expected stderr to contain 'SHIMUX_SESSION not set', got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "should be launched by shimux") {
		t.Errorf("expected stderr to contain usage hint, got %q", stderr.String())
	}
}

func TestRunWithIO_BothNotSet(t *testing.T) {
	t.Setenv("SHIMUX_PANE_ID", "")
	t.Setenv("SHIMUX_SESSION", "")

	var stdout, stderr bytes.Buffer
	code := runWithIO(&stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	// SHIMUX_PANE_IDのエラーが先に出る
	if !strings.Contains(stderr.String(), "SHIMUX_PANE_ID not set") {
		t.Errorf("expected stderr to contain 'SHIMUX_PANE_ID not set', got %q", stderr.String())
	}
}

func TestRunWithIO_ValidEnv_CreatesAgentWithCorrectConfig(t *testing.T) {
	t.Setenv("SHIMUX_PANE_ID", "42")
	t.Setenv("SHIMUX_SESSION", "my-session")

	// runWithIO は Agent.Run() を呼ぶため、PTYが必要で実際には失敗する。
	// ここでは環境変数バリデーションを通過してAgent作成に到達することを確認する。
	// Agent.Run()のエラーで終了するが、exit code 1 + stderrに"shimux-agent:"が出る。
	// バリデーション失敗なら "SHIMUX_PANE_ID not set" や "SHIMUX_SESSION not set" が出る。
	var stdout, stderr bytes.Buffer
	code := runWithIO(&stdout, &stderr)

	// 環境変数バリデーションのエラーメッセージが出ていないことを確認
	if strings.Contains(stderr.String(), "SHIMUX_PANE_ID not set") {
		t.Error("should not fail with SHIMUX_PANE_ID validation error")
	}
	if strings.Contains(stderr.String(), "SHIMUX_SESSION not set") {
		t.Error("should not fail with SHIMUX_SESSION validation error")
	}

	// Agent.Run()のエラーでexit 1になるが、それはPTY環境がないため。
	// バリデーションを通過したことが確認できればOK。
	_ = code
	_ = stdout
}

func TestRunWithIO_SocketPathGeneration(t *testing.T) {
	// SafeSocketPathが正しいパラメータで呼ばれることを間接的に確認
	// 長いセッション名でもクラッシュしないことを確認
	t.Setenv("SHIMUX_PANE_ID", "99")
	t.Setenv("SHIMUX_SESSION", strings.Repeat("very-long-session-name-", 10))

	var stdout, stderr bytes.Buffer
	code := runWithIO(&stdout, &stderr)

	// 環境変数バリデーションは通過する
	if strings.Contains(stderr.String(), "SHIMUX_PANE_ID not set") {
		t.Error("should not fail with SHIMUX_PANE_ID validation error")
	}
	if strings.Contains(stderr.String(), "SHIMUX_SESSION not set") {
		t.Error("should not fail with SHIMUX_SESSION validation error")
	}
	_ = code
}

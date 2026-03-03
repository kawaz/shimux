package agent

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestStress_ConcurrentSendKeys は20クライアントが同一サーバに同時送信し、
// 全リクエストが正常処理されることを検証する。
func TestStress_ConcurrentSendKeys(t *testing.T) {
	sockPath := testSocketPath(t)

	var mu sync.Mutex
	received := make(map[string]bool)

	srv, err := ListenAndServe(sockPath, func(data string) error {
		mu.Lock()
		received[data] = true
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	const numClients = 20
	var wg sync.WaitGroup
	barrier := make(chan struct{})
	errs := make([]error, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier
			data := fmt.Sprintf("client-%d", idx)
			errs[idx] = SendKeysWithTimeout(sockPath, data, 5*time.Second)
		}(i)
	}

	close(barrier)
	wg.Wait()

	// 全リクエストが成功していること
	for i, err := range errs {
		if err != nil {
			t.Errorf("client %d: %v", i, err)
		}
	}

	// 全データが受信されていること
	mu.Lock()
	defer mu.Unlock()
	for i := 0; i < numClients; i++ {
		key := fmt.Sprintf("client-%d", i)
		if !received[key] {
			t.Errorf("data %q not received", key)
		}
	}
}

// TestStress_SendKeysServerClose は送信中にサーバをCloseし、
// パニックが発生しないことを検証する。
func TestStress_SendKeysServerClose(t *testing.T) {
	sockPath := testSocketPath(t)

	// ハンドラに少し遅延を入れてリクエスト処理中にCloseがかかるようにする
	srv, err := ListenAndServe(sockPath, func(data string) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	const numClients = 10
	var wg sync.WaitGroup
	barrier := make(chan struct{})

	// クライアントgoroutineを起動
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier
			// エラーは期待通り。パニックしなければOK
			_ = SendKeysWithTimeout(sockPath, fmt.Sprintf("data-%d", idx), 2*time.Second)
		}(i)
	}

	// サーバを少し遅れてClose
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-barrier
		time.Sleep(20 * time.Millisecond)
		srv.Close()
	}()

	close(barrier)
	wg.Wait()
	// パニックしなければ成功
}

// TestStress_MassConnections は100接続を連続送信し、
// セマフォ(16)が正しく機能して全リクエストが処理されることを検証する。
func TestStress_MassConnections(t *testing.T) {
	sockPath := testSocketPath(t)

	var processedCount atomic.Int64
	var peakConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	srv, err := ListenAndServe(sockPath, func(data string) error {
		cur := currentConcurrent.Add(1)
		// peakを更新
		for {
			peak := peakConcurrent.Load()
			if cur <= peak {
				break
			}
			if peakConcurrent.CompareAndSwap(peak, cur) {
				break
			}
		}
		// 少し処理時間をシミュレート
		time.Sleep(5 * time.Millisecond)
		currentConcurrent.Add(-1)
		processedCount.Add(1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	const numClients = 100
	var wg sync.WaitGroup
	barrier := make(chan struct{})
	errs := make([]error, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier
			errs[idx] = SendKeysWithTimeout(sockPath, fmt.Sprintf("mass-%d", idx), 10*time.Second)
		}(i)
	}

	close(barrier)
	wg.Wait()

	// エラーチェック
	errCount := 0
	for i, err := range errs {
		if err != nil {
			errCount++
			if errCount <= 3 {
				t.Logf("client %d error (showing first 3): %v", i, err)
			}
		}
	}
	if errCount > 0 {
		t.Errorf("%d/%d clients failed", errCount, numClients)
	}

	// セマフォ制限が機能していること: peak concurrent <= maxConcurrent(16)
	peak := peakConcurrent.Load()
	if peak > int32(maxConcurrent) {
		t.Errorf("peak concurrent handlers = %d, want <= %d", peak, maxConcurrent)
	}
	t.Logf("peak concurrent handlers: %d (limit: %d)", peak, maxConcurrent)

	// 全リクエストが処理されたこと
	processed := processedCount.Load()
	if processed != numClients {
		t.Errorf("processed = %d, want %d", processed, numClients)
	}
}

package speedtest

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/21state/celestia-snapshot-finder/internal/provider"
)

const (
	testDuration = 10 * time.Second
	bufferSize   = 32 * 1024
)

type SpeedTester struct {
	client   *http.Client
	debugLog provider.DebugLogger
}

func NewSpeedTester(debugLog provider.DebugLogger) *SpeedTester {
	return &SpeedTester{
		client: &http.Client{
			Timeout: testDuration + 5*time.Second,
		},
		debugLog: debugLog,
	}
}

func (st *SpeedTester) TestProviders(providers []provider.ProviderInfo) []provider.ProviderInfo {
	result := make([]provider.ProviderInfo, len(providers))
	copy(result, providers)

	var wg sync.WaitGroup
	wg.Add(len(providers))

	for i := range result {
		go func(idx int) {
			defer wg.Done()
			st.debugLog("Running speed test for provider %s", result[idx].Name)
			speed := st.testSpeed(result[idx].URL)
			result[idx].Speed = speed
			st.debugLog("Speed test result for %s: %.2f MB/s", result[idx].Name, speed)
		}(i)
	}

	wg.Wait()
	return result
}

func (st *SpeedTester) testSpeed(url string) float64 {
	resp, err := st.client.Get(url)
	if err != nil {
		st.debugLog("Speed test failed for URL %s: %v", url, err)
		return 0
	}
	defer resp.Body.Close()

	start := time.Now()
	deadline := start.Add(testDuration)

	var totalBytes int64
	buffer := make([]byte, bufferSize)

	for time.Now().Before(deadline) {
		n, err := resp.Body.Read(buffer)
		if err != nil && err != io.EOF {
			st.debugLog("Error during speed test: %v", err)
			break
		}
		totalBytes += int64(n)
		if err == io.EOF {
			break
		}
	}

	duration := time.Since(start).Seconds()
	if duration == 0 {
		return 0
	}

	return float64(totalBytes) / duration / 1000 / 1000
}

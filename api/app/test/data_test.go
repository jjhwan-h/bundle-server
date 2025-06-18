package test

import (
	"net/http"
	"sync"
	"testing"
)

func TestBuildDataJson(t *testing.T) {
	var wg sync.WaitGroup

	cocurrency := 10

	for i := 0; i < cocurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:4001/data", nil)
			if err != nil {
				t.Errorf("[%d]failed to create request: %v", i, err)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("[%d]failed to request /data: %v", i, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusAccepted {
				t.Errorf("[%d]unexpected status: got %d, expected %d", i, resp.StatusCode, http.StatusAccepted)
			}
		}(i)
	}

	wg.Wait()
}

package test

import (
	"net/http"
	"testing"
)

func TestBuildDataJson(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:4001/data", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to request /data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusAccepted)
	}
}

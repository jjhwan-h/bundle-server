package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

func TestBuildDataNBundles(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:4001/services/casb/data/trigger", nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to request /services/casb/data/trigger: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusAccepted)
	}
}

func TestCreateBundle(t *testing.T) {
	// policy.rego, data.json 필요

	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:4001/services/casb/policy/trigger", nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to request /services/casb/policy/trigger: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusAccepted)
	}
}

func TestRegisterPolicy(t *testing.T) {
	regoSrc := `
		package example

		default allow = false

		allow {
		input.user == "admin"
		}
	`

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("file", "policy.rego")
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(part, strings.NewReader(regoSrc))
	if err != nil {
		panic(err)
	}

	err = writer.Close()
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:4001/services/casb/policy", &requestBody)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to request /services/casb/policy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestServeBundle(t *testing.T) {
	// delta, regular bundle 필요

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:4001/services/casb/bundle?type=delta", nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to request /services/casb/bundle?type=delta: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusOK)
	}

	req, err = http.NewRequest(http.MethodGet, "http://127.0.0.1:4001/services/casb/bundle", nil)
	if err != nil {
		t.Errorf("failed to create request : %v", err)
	}

	client = &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Errorf("failed to request /services/casb/bundle: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusOK)
	}
}

func TestRegisterClients(t *testing.T) {
	payload := []string{
		"127.0.0.1:5556",
		"127.0.0.1:5557",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:4001/services/casb/clients", bytes.NewBuffer(body))
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to request /services/casb/clients: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusOK)
	}
}

func TestServeClients(t *testing.T) {
	// clients등록필요

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:4001/services/clients", nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to request /services/clients: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println("clients", string(body))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusOK)
	}
}

func TestServeServiceClients(t *testing.T) {
	// clients등록필요

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:4001/services/casb/clients", nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to request /services/casb/clients: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println("clients", string(body))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusOK)
	}
}

func TestDeleteClients(t *testing.T) {
	// clients등록 필요

	req, err := http.NewRequest(http.MethodDelete, "http://127.0.0.1:4001/services/casb/clients?client=127.0.0.1:5556", nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("failed to request /services/casb/clients?client=127.0.0.1:5556 : %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusOK)
	}

	req, err = http.NewRequest(http.MethodGet, "http://127.0.0.1:4001/services/casb/clients", nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	client = &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Errorf("failed to request services/casb/clients : %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: got %d, expected %d", resp.StatusCode, http.StatusOK)
	}
}

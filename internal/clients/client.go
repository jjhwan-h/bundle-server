package clients

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	appErr "github.com/jjhwan-h/bundle-server/internal/errors"
)

type Client struct { // 이벤트발생 시 알림보낼 client
	data map[string][]string
	mu   sync.Mutex
}

func NewClient(clients map[string][]string) *Client {
	Client := &Client{
		data: make(map[string][]string),
	}

	Client.mu.Lock()
	defer Client.mu.Unlock()

	if len(clients) > 0 {
		Client.data = clients
	}
	return Client
}

func (b *Client) New(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.data[key] = []string{}
}

func (b *Client) Add(key, val string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data[key] = append(b.data[key], val)
}

func (b *Client) GetAll() map[string][]string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data
}

func (b *Client) Get(service string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.data[service]
}

func (b *Client) Delete(service, value string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	clients := b.data[service]
	var (
		tmp   []string
		found bool
	)

	for _, client := range clients {
		if client == value {
			found = true
			continue
		}
		tmp = append(tmp, client)
	}

	if !found {
		return fmt.Errorf("client '%s' not found for service '%s'", value, service)
	}

	b.data[service] = tmp
	return nil
}

func (b *Client) DeleteAll(service string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.data[service] = []string{}
}

func (b *Client) IsDuplicate(key, val string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, v := range b.data[key] {
		if v == val {
			return true
		}
	}
	return false
}

func (b *Client) Hook(path string, service string) error {
	var (
		err           error
		failedTargets []string
	)
	clients := b.Get(service)

	for _, addr := range clients {
		_, errHTTPPost := http.Post(fmt.Sprintf("%s/%s", addr, path), "application/json", nil)
		if errHTTPPost != nil {
			failedTargets = append(failedTargets, addr)
		}
	}

	if len(failedTargets) != 0 {
		err = fmt.Errorf("%w: %s", appErr.ErrSendEventNotification, strings.Join(failedTargets, ", "))
	}
	return err
}

func (b *Client) AddHookClient(clients []string, service string) error {
	for _, client := range clients {
		if b.IsDuplicate(service, client) {
			return fmt.Errorf("%w : %s", appErr.ErrAlreadyRegistered, client)
		}
	}

	for _, client := range clients {
		b.Add(service, client)
	}
	return nil
}

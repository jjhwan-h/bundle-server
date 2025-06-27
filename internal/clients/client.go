package clients

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/jjhwan-h/bundle-server/config"
	"github.com/jjhwan-h/bundle-server/internal/bundle"
	appErr "github.com/jjhwan-h/bundle-server/internal/errors"
	"go.uber.org/zap"
)

type Client struct { // 이벤트발생 시 알림보낼 client
	data   map[string][]string
	Bundle map[string]*bundle.Bundle
	mu     sync.Mutex
}

func NewClient(logger *zap.Logger, clients map[string][]string) *Client {
	Client := &Client{
		data:   make(map[string][]string),
		Bundle: make(map[string]*bundle.Bundle),
	}

	Client.mu.Lock()
	defer Client.mu.Unlock()

	if len(clients) > 0 {
		Client.data = clients
	}

	for k := range clients {
		Client.Bundle[k] = bundle.NewBundle(
			fmt.Sprintf("%s/%s", config.Cfg.OpaDataPath, k),
		)

		minor := Client.Bundle[k].Latest.Minor
		major := Client.Bundle[k].Latest.Major
		if minor == 0 && major == 0 {
			logger.Info("bundle version is starting from v0.1", zap.String("service", k))
		} else {
			etag, err := Client.Bundle[k].ETagFromFile()
			if err != nil {
				logger.Error("failed to hash latest bundle", zap.String("version", fmt.Sprintf("v%d.%d", major, minor)), zap.Error(err))
				return nil
			}

			logger.Info("latest bundle", zap.String("version", fmt.Sprintf("v%d.%d", major, minor)))
			logger.Info("successfully hashed latest bundle",
				zap.String("version", fmt.Sprintf("v%d.%d", major, minor)),
				zap.String("etag", etag[:8]+"..."), // 해시 결과도 함께 기록
			)

		}
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
		p := fmt.Sprintf("%s/%s", addr, path)

		req, err := http.NewRequest("POST", p, nil)
		if err != nil {
			failedTargets = append(failedTargets, p)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode >= 400 {
			failedTargets = append(failedTargets, p)
			continue
		}
		defer resp.Body.Close()
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

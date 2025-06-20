package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

func ToInterfaceSlice(strs []string) []any {
	args := make([]any, len(strs))
	for i, s := range strs {
		args[i] = s
	}

	return args
}

func SaveToFile(ctx context.Context, reader io.Reader, path string) error {
	if reader == nil {
		return fmt.Errorf("reader parameter is nil")
	}

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	lockPath := path + ".lock"
	fileLock := flock.New(lockPath) // 멀티 프로세스/컨테이너 환경에서의 쓰기 충돌 방지

	locked, err := fileLock.TryLockContext(ctx, time.Millisecond*500)
	if err != nil {
		return fmt.Errorf("failed to acuire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("timeout: could not acquire file lock")
	}
	defer fileLock.Unlock()

	tmpPath := path + ".tmp"
	dst, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if _, err = io.Copy(dst, reader); err != nil {
		dst.Close()
		os.Remove(tmpPath)
		return err
	}
	dst.Close()

	return os.Rename(tmpPath, path) // atomic write
}

func GetPath(p string) string {
	baseDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	return path.Join(baseDir, p)
}

func EncodeJson(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err := enc.Encode(v)

	return err
}

func StructToMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

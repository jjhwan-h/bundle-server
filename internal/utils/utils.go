package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

func ToInterfaceSlice(strs []string) []any {
	args := make([]any, len(strs))
	for i, s := range strs {
		args[i] = s
	}

	return args
}

func SaveToFile(reader io.Reader, path string) error {
	if reader == nil {
		return fmt.Errorf("reader parameter is nil")
	}

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, reader)
	return err
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

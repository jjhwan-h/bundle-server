package utils

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

func ToInterfaceSlice(strs []string) []interface{} {
	args := make([]interface{}, len(strs))
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

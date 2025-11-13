package json

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteTemplate(path string, contents string, overwrite bool) (string, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create dirs for %s: %w", path, err)
	}

	finalPath := path
	if !overwrite {
		if _, err := os.Stat(finalPath); err == nil {
			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			ext := filepath.Ext(path)
			for i := 2; ; i++ {
				try := filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
				if _, err := os.Stat(try); os.IsNotExist(err) {
					finalPath = try
					break
				}
			}
		}
	}

	if err := os.WriteFile(finalPath, []byte(contents), 0o644); err != nil {
		return "", fmt.Errorf("write template %s: %w", finalPath, err)
	}
	return finalPath, nil
}

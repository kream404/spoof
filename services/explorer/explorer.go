package explorer

import (
	"os"
	"path/filepath"
	"sort"
)

type Entry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type ExplorerService struct{}

func NewExplorerService() *ExplorerService {
	return &ExplorerService{}
}

func (e *ExplorerService) ListDir(dir string) ([]Entry, error) {
	entries := []Entry{}

	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		entry := Entry{
			Name: item.Name(),
			Path: filepath.Join(dir, item.Name()),
			Type: "file",
		}
		if item.IsDir() {
			entry.Type = "directory"
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type == entries[j].Type {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Type == "directory"
	})

	return entries, nil
}

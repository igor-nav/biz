package biz

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func LoadCandidates(root string) ([]Candidate, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", root, err)
	}

	var candidates []Candidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		path := filepath.Join(root, slug, "data.json")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		var biz Business
		if err := json.Unmarshal(data, &biz); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		candidates = append(candidates, Candidate{
			Slug: slug,
			Path: filepath.ToSlash(path),
			Biz:  biz,
		})
	}
	return candidates, nil
}

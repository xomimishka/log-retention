package fsmodel

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// версия формата снимка
const SnapshotVersion = 1

// FileInfo описывает один файл в снимке файловой системы.
type FileInfo struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	IsDir   bool      `json:"is_dir"`
	Symlink bool      `json:"symlink"`
}

// Snapshot описывает состояние объявленных корней на момент Now.
type Snapshot struct {
	Version int        `json:"version"`
	Now     time.Time  `json:"now"`
	Roots   []string   `json:"roots"`
	Files   []FileInfo `json:"files"`
}

func (s *Snapshot) Normalize() {
	if s.Version == 0 {
		s.Version = SnapshotVersion
	}

	s.Now = s.Now.UTC()

	if s.Roots == nil {
		s.Roots = []string{}
	}

	if s.Files == nil {
		s.Files = []FileInfo{}
	}

	sort.Strings(s.Roots)

	for i := range s.Files {
		s.Files[i].ModTime = s.Files[i].ModTime.UTC()
	}

	sort.Slice(s.Files, func(i, j int) bool {
		return s.Files[i].Path < s.Files[j].Path
	})
}

func MarshalSnapshotJSON(s Snapshot) ([]byte, error) {
	s.Normalize()

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(b, '\n'), nil
}

func ParseSnapshotJSON(data []byte) (Snapshot, error) {
	var s Snapshot

	if err := json.Unmarshal(data, &s); err != nil {
		return Snapshot{}, err
	}

	if s.Version != SnapshotVersion {
		return Snapshot{}, fmt.Errorf("unsupported snapshot version: %d", s.Version)
	}

	s.Normalize()

	return s, nil
}

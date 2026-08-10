package fsmodel

import (
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
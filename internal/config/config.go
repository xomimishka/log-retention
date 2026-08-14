package config

import (
	"time"
)

type Config struct {
	Version  int      `yaml:"version" json:"version"`
	Vars     Vars     `yaml:"vars,omitempty" json:"vars,omitempty"`
	Policies []Policy `yaml:"policies" json:"policies"`
}

type Vars map[string]Var

type Var struct {
	Default string `yaml:"default,omitempty" json:"default,omitempty"`
}

type Policy struct {
	Name         string     `yaml:"name" json:"name"`
	Priority     int        `yaml:"priority,omitempty" json:"priority,omitempty"`
	Roots        []string   `yaml:"roots" json:"roots"`
	Recursive    bool       `yaml:"recursive,omitempty" json:"recursive,omitempty"`
	Include      []string   `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude      []string   `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	Group        Group      `yaml:"group,omitempty" json:"group,omitempty"`
	Select       Select     `yaml:"select,omitempty" json:"select,omitempty"`
	Schedule     Schedule   `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Archive      Archive    `yaml:"archive,omitempty" json:"archive,omitempty"`
	AfterArchive string     `yaml:"after_archive,omitempty" json:"after_archive,omitempty"`
	Retention    *Retention `yaml:"retention,omitempty" json:"retention,omitempty"`
}

type Group struct {
	By     string `yaml:"by,omitempty" json:"by,omitempty"`
	Regexp string `yaml:"regexp,omitempty" json:"regexp,omitempty"`
}

type Select struct {
	MinAge          string `yaml:"min_age,omitempty" json:"min_age,omitempty"`
	MinSize         string `yaml:"min_size,omitempty" json:"min_size,omitempty"`
	KeepGenerations int    `yaml:"keep_generations,omitempty" json:"keep_generations,omitempty"`

	MinAgeDur  time.Duration `yaml:"-" json:"-"`
	MinSizeVal int64         `yaml:"-" json:"-"`
}

type Schedule struct {
	Window   string `yaml:"window,omitempty" json:"window,omitempty"`
	Timezone string `yaml:"timezone,omitempty" json:"timezone,omitempty"`
}

type Archive struct {
	Compress        string `yaml:"compress,omitempty" json:"compress,omitempty"`
	Level           *int   `yaml:"level,omitempty" json:"level,omitempty"`
	Dest            string `yaml:"dest,omitempty" json:"dest,omitempty"`
	Name            string `yaml:"name,omitempty" json:"name,omitempty"`
	FolderInArchive string `yaml:"folder_in_archive,omitempty" json:"folder_in_archive,omitempty"`
	MergeSameDay    bool   `yaml:"merge_same_day,omitempty" json:"merge_same_day,omitempty"`
}

type Retention struct {
	MaxAge       string `yaml:"max_age,omitempty" json:"max_age,omitempty"`
	MaxCount     int    `yaml:"max_count,omitempty" json:"max_count,omitempty"`
	MaxTotalSize string `yaml:"max_total_size,omitempty" json:"max_total_size,omitempty"`
	KeepMin      int    `yaml:"keep_min,omitempty" json:"keep_min,omitempty"`

	MaxAgeDur       time.Duration `yaml:"-" json:"-"`
	MaxTotalSizeVal int64         `yaml:"-" json:"-"`
}

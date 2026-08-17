package policy

import (
	"testing"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
)

func TestGroupByDir(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFile("/var/log/a/app.log", now),
		makeFile("/var/log/a/app.log.1", now),
		makeFile("/var/log/b/agent.log", now),
	}

	p := config.Policy{
		Name:  "p1",
		Group: config.Group{By: "dir"},
	}

	res, err := GroupCandidates(files, p)
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(res.Groups))
	}
	if len(res.Groups["/var/log/a"]) != 2 {
		t.Errorf("group /var/log/a has %d files, want 2", len(res.Groups["/var/log/a"]))
	}
	if len(res.Groups["/var/log/b"]) != 1 {
		t.Errorf("group /var/log/b has %d files, want 1", len(res.Groups["/var/log/b"]))
	}
}

func TestGroupByName(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFile("/var/log/service.log", now),
		makeFile("/var/log/service.log.1", now),
		makeFile("/var/log/service.log.2", now),
		makeFile("/var/log/agent.log", now),
	}

	p := config.Policy{
		Name:  "p1",
		Group: config.Group{By: "name"},
	}

	res, err := GroupCandidates(files, p)
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(res.Groups))
	}
	if len(res.Groups["service"]) != 3 {
		t.Errorf("group service has %d files, want 3", len(res.Groups["service"]))
	}
	if len(res.Groups["agent"]) != 1 {
		t.Errorf("group agent has %d files, want 1", len(res.Groups["agent"]))
	}
}

func TestGroupByRegexp(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFile("/var/log/service.log", now),
		makeFile("/var/log/service.log.1", now),
		makeFile("/var/log/service.log.2", now),
		makeFile("/var/log/agent.log", now),
		makeFile("/var/log/notes.txt", now),
	}

	p := config.Policy{
		Name: "p1",
		Group: config.Group{
			By:     "regexp",
			Regexp: `^(?P<group>.+?)\.log(\.\d+)?$`,
		},
	}

	res, err := GroupCandidates(files, p)
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(res.Groups))
	}
	if len(res.Groups["service"]) != 3 {
		t.Errorf("group service has %d files, want 3", len(res.Groups["service"]))
	}
	if len(res.Groups["agent"]) != 1 {
		t.Errorf("group agent has %d files, want 1", len(res.Groups["agent"]))
	}
	if len(res.NoGroup) != 1 {
		t.Fatalf("got %d no_group files, want 1", len(res.NoGroup))
	}
	if res.NoGroup[0].Path != "/var/log/notes.txt" {
		t.Errorf("no_group file = %q, want /var/log/notes.txt", res.NoGroup[0].Path)
	}
}

func TestGroupOrdering(t *testing.T) {
	t1 := time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 8, 17, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

	files := []fsmodel.FileInfo{
		makeFile("/var/log/app.log.1", t1),
		makeFile("/var/log/app.log.2", t2),
		makeFile("/var/log/app.log", t3),
	}

	p := config.Policy{
		Name:  "p1",
		Group: config.Group{By: "name"},
	}

	res, err := GroupCandidates(files, p)
	if err != nil {
		t.Fatal(err)
	}

	group := res.Groups["app"]
	if len(group) != 3 {
		t.Fatalf("got %d files, want 3", len(group))
	}

	if group[0].Path != "/var/log/app.log" {
		t.Errorf("first = %q, want /var/log/app.log", group[0].Path)
	}
	if group[1].Path != "/var/log/app.log.2" {
		t.Errorf("second = %q, want /var/log/app.log.2", group[1].Path)
	}
	if group[2].Path != "/var/log/app.log.1" {
		t.Errorf("third = %q, want /var/log/app.log.1", group[2].Path)
	}
}

func TestGroupOrderingEqualModTime(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFile("/var/log/b.log", now),
		makeFile("/var/log/a.log", now),
		makeFile("/var/log/c.log", now),
	}

	p := config.Policy{
		Name:  "p1",
		Group: config.Group{By: "name"},
	}

	res, err := GroupCandidates(files, p)
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Groups) != 3 {
		t.Fatalf("got %d groups, want 3", len(res.Groups))
	}
}

package policy

import (
	"fmt"
	"strings"
	"time"

	"example/log-retention/internal/config"
)

type Window struct {
	Start     int
	End       int
	Overnight bool
	AllDay    bool
	Timezone  *time.Location
}

func ParseWindow(window, timezone string) (*Window, error) {
	if window == "" {
		return nil, nil
	}

	var loc *time.Location
	if timezone == "" {
		loc = time.Local
	} else {
		var err error
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return nil, fmt.Errorf("unknown timezone %q: %w", timezone, err)
		}
	}

	parts := strings.Split(window, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid window format %q, expected HH:MM-HH:MM", window)
	}

	startMin, err := parseHHMM(parts[0])
	if err != nil {
		return nil, err
	}
	endMin, err := parseHHMM(parts[1])
	if err != nil {
		return nil, err
	}

	w := &Window{
		Start:    startMin,
		End:      endMin,
		Timezone: loc,
	}

	if startMin == 0 && endMin == 0 {
		w.AllDay = true
		return w, nil
	}

	if startMin > endMin {
		w.Overnight = true
	}

	return w, nil
}

func parseHHMM(s string) (int, error) {
	if len(s) != 5 || s[2] != ':' {
		return 0, fmt.Errorf("invalid time %q, expected HH:MM", s)
	}
	hh := s[:2]
	mm := s[3:]
	h := int(hh[0]-'0')*10 + int(hh[1]-'0')
	m := int(mm[0]-'0')*10 + int(mm[1]-'0')
	if h > 23 {
		return 0, fmt.Errorf("invalid hour %d in %q", h, s)
	}
	if m > 59 {
		return 0, fmt.Errorf("invalid minute %d in %q", m, s)
	}
	return h*60 + m, nil
}

func (w *Window) Contains(t time.Time) bool {
	if w == nil || w.AllDay {
		return true
	}

	local := t.In(w.Timezone)
	minutes := local.Hour()*60 + local.Minute()

	if w.Overnight {
		return minutes >= w.Start || minutes <= w.End
	}
	return minutes >= w.Start && minutes <= w.End
}

func (w *Window) String() string {
	if w == nil {
		return "always"
	}
	if w.AllDay {
		return "00:00-00:00 (all day)"
	}
	sh, sm := w.Start/60, w.Start%60
	eh, em := w.End/60, w.End%60
	return fmt.Sprintf("%02d:%02d-%02d:%02d (%s)", sh, sm, eh, em, w.Timezone)
}

func IsOutsideWindow(now time.Time, p config.Policy) (bool, error) {
	if p.Schedule.Window == "" {
		return false, nil
	}
	w, err := ParseWindow(p.Schedule.Window, p.Schedule.Timezone)
	if err != nil {
		return false, err
	}
	return !w.Contains(now), nil
}

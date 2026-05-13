package model

import "time"

type State string

const (
	StateOK    State = "ok"
	StateError State = "error"
	StateStale State = "stale"
)

type ThresholdZone string

const (
	ZoneOK        ThresholdZone = "ok"
	ZoneWarning   ThresholdZone = "warning"
	ZoneCritical  ThresholdZone = "critical"
	ZoneOverLimit ThresholdZone = "over-limit"
)

type Usage struct {
	Provider    string     `json:"provider"`
	Label       string     `json:"label"`
	State       State      `json:"state"`
	Percent     *float64   `json:"percent,omitempty"`
	Cost        *float64   `json:"cost,omitempty"`
	RefreshedAt time.Time  `json:"refreshed_at"`
	ErrorMsg    string     `json:"error_msg,omitempty"`
}

type ThresholdConfig struct {
	Warning  int
	Critical int
}

func EvaluateThreshold(percent *float64, cfg ThresholdConfig) ThresholdZone {
	if percent == nil {
		return ZoneOK
	}
	pct := *percent
	if pct > 100 {
		return ZoneOverLimit
	}
	if cfg.Critical > 0 && pct >= float64(cfg.Critical) {
		return ZoneCritical
	}
	if cfg.Warning > 0 && pct >= float64(cfg.Warning) {
		return ZoneWarning
	}
	return ZoneOK
}

type DebugInfo struct {
	Provider string `json:"provider"`
	Path     string `json:"path"`
	RawData  string `json:"raw_data"`
	Usage    *Usage `json:"usage,omitempty"`
	State    string `json:"state"`
	Error    string `json:"error,omitempty"`
}

package iec104

import "time"

type PointValue struct {
	Timestamp     time.Time `json:"timestamp"`
	CommonAddress uint16    `json:"common_address"`
	IOA           uint32    `json:"ioa"`
	Name          string    `json:"name,omitempty"`
	Type          string    `json:"type"`
	Cause         string    `json:"cause,omitempty"`
	Value         any       `json:"value"`
	Unit          string    `json:"unit,omitempty"`
	Quality       Quality   `json:"quality"`
	RawTypeID     uint8     `json:"raw_type_id,omitempty"`
}

type Quality struct {
	Invalid     bool `json:"invalid"`
	NotTopical  bool `json:"not_topical"`
	Substituted bool `json:"substituted"`
	Blocked     bool `json:"blocked"`
}

func (q Quality) Display() string {
	parts := make([]string, 0, 4)
	if q.Invalid {
		parts = append(parts, "invalid")
	}
	if q.NotTopical {
		parts = append(parts, "not_topical")
	}
	if q.Substituted {
		parts = append(parts, "substituted")
	}
	if q.Blocked {
		parts = append(parts, "blocked")
	}
	if len(parts) == 0 {
		return "good"
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += "," + part
	}
	return out
}

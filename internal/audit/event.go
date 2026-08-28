package audit

import "time"

type Event struct {
	OccurredAt     time.Time
	RequestID      string
	ActorAccountID string
	ActorDeviceID  string
	EffectiveRole  string
	CapabilityID   string
	TargetType     string
	TargetID       string
	Outcome        string
	SourceIP       string
	UserAgent      string
	Metadata       map[string]any
}

type Record struct {
	ID int64
	Event
}

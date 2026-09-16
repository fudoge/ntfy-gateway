package domain

import "time"

type Event struct {
	SourceID  string
	Type      string
	Title     string
	Message   string
	Severity  string
	Timestamp time.Time
	Metadata  map[string]string
}

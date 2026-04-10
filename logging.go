package logging

import (
	"context"
	"strings"
	"time"
)

type Event struct {
	ID      string         `json:"id"`
	TS      time.Time      `json:"ts"`
	Source  string         `json:"source"`
	Kind    string         `json:"kind"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Plugin  string         `json:"plugin,omitempty"`
	Device  string         `json:"device,omitempty"`
	Entity  string         `json:"entity,omitempty"`
	Action  string         `json:"action,omitempty"`
	TraceID string         `json:"trace_id,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

func (e *Event) Normalize() {
	e.ID = strings.TrimSpace(e.ID)
	e.Source = strings.TrimSpace(e.Source)
	e.Kind = strings.TrimSpace(e.Kind)
	e.Level = strings.TrimSpace(e.Level)
	e.Message = strings.TrimSpace(e.Message)
	e.Plugin = strings.TrimSpace(e.Plugin)
	e.Device = strings.TrimSpace(e.Device)
	e.Entity = strings.TrimSpace(e.Entity)
	e.Action = strings.TrimSpace(e.Action)
	e.TraceID = strings.TrimSpace(e.TraceID)
	if e.TS.IsZero() {
		e.TS = time.Now().UTC()
	} else {
		e.TS = e.TS.UTC()
	}
}

type ListRequest struct {
	Since   time.Time `json:"since,omitempty"`
	Until   time.Time `json:"until,omitempty"`
	Source  string    `json:"source,omitempty"`
	Kind    string    `json:"kind,omitempty"`
	Level   string    `json:"level,omitempty"`
	Plugin  string    `json:"plugin,omitempty"`
	Device  string    `json:"device,omitempty"`
	Entity  string    `json:"entity,omitempty"`
	TraceID string    `json:"trace_id,omitempty"`
	Limit   int       `json:"limit,omitempty"`
}

func (r *ListRequest) Normalize() {
	r.Source = strings.TrimSpace(r.Source)
	r.Kind = strings.TrimSpace(r.Kind)
	r.Level = strings.TrimSpace(r.Level)
	r.Plugin = strings.TrimSpace(r.Plugin)
	r.Device = strings.TrimSpace(r.Device)
	r.Entity = strings.TrimSpace(r.Entity)
	r.TraceID = strings.TrimSpace(r.TraceID)
	if !r.Since.IsZero() {
		r.Since = r.Since.UTC()
	}
	if !r.Until.IsZero() {
		r.Until = r.Until.UTC()
	}
}

type Store interface {
	Append(ctx context.Context, event Event) error
	Get(ctx context.Context, id string) (Event, error)
	List(ctx context.Context, req ListRequest) ([]Event, error)
}

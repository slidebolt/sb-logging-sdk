package logging

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	messenger "github.com/slidebolt/sb-messenger-sdk"
)

func TestAppendReturnsBeforeServerResponse(t *testing.T) {
	msg, err := messenger.Mock()
	if err != nil {
		t.Fatalf("Mock: %v", err)
	}
	defer msg.Close()

	sub, err := msg.Subscribe("logging.append", func(m *messenger.Message) {
		time.Sleep(300 * time.Millisecond)
		resp, _ := json.Marshal(Response{OK: true})
		_ = m.Respond(resp)
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	if err := msg.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	client := newClient(msg, defaultAppendQueueSize)
	defer client.Close()

	start := time.Now()
	if err := client.Append(context.Background(), Event{
		ID:      "evt-1",
		Source:  "test",
		Kind:    "append",
		Level:   "info",
		Message: "async append",
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Append blocked too long: %v", elapsed)
	}
}

func TestAppendEventuallyPublishesQueuedEvent(t *testing.T) {
	msg, err := messenger.Mock()
	if err != nil {
		t.Fatalf("Mock: %v", err)
	}
	defer msg.Close()

	var (
		mu       sync.Mutex
		events   []Event
		received = make(chan struct{}, 1)
	)
	sub, err := msg.Subscribe("logging.append", func(m *messenger.Message) {
		var req AppendRequest
		if err := json.Unmarshal(m.Data, &req); err != nil {
			t.Errorf("Unmarshal: %v", err)
			return
		}
		mu.Lock()
		events = append(events, req.Event)
		mu.Unlock()
		select {
		case received <- struct{}{}:
		default:
		}
		resp, _ := json.Marshal(Response{OK: true})
		_ = m.Respond(resp)
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	if err := msg.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	client := newClient(msg, defaultAppendQueueSize)
	defer client.Close()

	want := Event{
		ID:      "evt-2",
		Source:  "plugin-esphome",
		Kind:    "state.updated",
		Level:   "info",
		Message: "state updated",
		TraceID: "trace-1",
	}
	if err := client.Append(context.Background(), want); err != nil {
		t.Fatalf("Append: %v", err)
	}

	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async append delivery")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 1 {
		t.Fatalf("received %d events, want 1", len(events))
	}
	if events[0].ID != want.ID || events[0].TraceID != want.TraceID || events[0].Kind != want.Kind {
		t.Fatalf("received %+v, want %+v", events[0], want)
	}
}

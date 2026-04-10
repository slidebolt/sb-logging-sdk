package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	messenger "github.com/slidebolt/sb-messenger-sdk"
)

const defaultTimeout = 5 * time.Second
const defaultAppendQueueSize = 1024
const appendWarningInterval = 5 * time.Second

type AppendRequest struct {
	Event Event `json:"event"`
}

type GetRequest struct {
	ID string `json:"id"`
}

type ListLogsRequest struct {
	Request ListRequest `json:"request"`
}

type Response struct {
	OK     bool    `json:"ok"`
	Event  *Event  `json:"event,omitempty"`
	Events []Event `json:"events,omitempty"`
	Error  string  `json:"error,omitempty"`
}

type client struct {
	msg        messenger.Messenger
	appendCh   chan []byte
	closeCh    chan struct{}
	closed     chan struct{}
	closeOnce  sync.Once
	dropped    atomic.Uint64
	lastWarnAt atomic.Int64
}

func Connect(deps map[string]json.RawMessage) (Store, error) {
	msg, err := messenger.Connect(deps)
	if err != nil {
		return nil, fmt.Errorf("logging: %w", err)
	}
	return newClient(msg, defaultAppendQueueSize), nil
}

func ConnectURL(url string) (Store, error) {
	msg, err := messenger.ConnectURL(url)
	if err != nil {
		return nil, fmt.Errorf("logging: %w", err)
	}
	return newClient(msg, defaultAppendQueueSize), nil
}

func ClientFrom(msg messenger.Messenger) Store {
	return newClient(msg, defaultAppendQueueSize)
}

func (c *client) Append(_ context.Context, event Event) error {
	req, err := json.Marshal(AppendRequest{Event: event})
	if err != nil {
		return fmt.Errorf("logging: marshal append: %w", err)
	}
	select {
	case c.appendCh <- req:
		return nil
	default:
		c.noteDropped()
		return nil
	}
}

func (c *client) Get(_ context.Context, id string) (Event, error) {
	req, err := json.Marshal(GetRequest{ID: id})
	if err != nil {
		return Event{}, fmt.Errorf("logging: marshal get: %w", err)
	}
	resp, err := c.request("logging.get", req)
	if err != nil {
		return Event{}, err
	}
	if resp.Event == nil {
		return Event{}, ErrNotFound
	}
	return *resp.Event, nil
}

func (c *client) List(_ context.Context, request ListRequest) ([]Event, error) {
	request.Normalize()
	req, err := json.Marshal(ListLogsRequest{Request: request})
	if err != nil {
		return nil, fmt.Errorf("logging: marshal list: %w", err)
	}
	resp, err := c.request("logging.list", req)
	if err != nil {
		return nil, err
	}
	return resp.Events, nil
}

func (c *client) request(subject string, data []byte) (*Response, error) {
	msg, err := c.msg.Request(subject, data, defaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("logging: %s: %w", subject, err)
	}
	var resp Response
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("logging: parse response: %w", err)
	}
	if resp.Error != "" {
		if resp.Error == ErrNotFound.Error() {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("logging: %s", resp.Error)
	}
	return &resp, nil
}

func (c *client) Close() {
	c.closeOnce.Do(func() {
		close(c.closeCh)
		<-c.closed
		c.msg.Close()
	})
}

func newClient(msg messenger.Messenger, appendQueueSize int) *client {
	if appendQueueSize <= 0 {
		appendQueueSize = defaultAppendQueueSize
	}
	c := &client{
		msg:      msg,
		appendCh: make(chan []byte, appendQueueSize),
		closeCh:  make(chan struct{}),
		closed:   make(chan struct{}),
	}
	go c.run()
	return c
}

func (c *client) run() {
	defer close(c.closed)
	for {
		select {
		case req := <-c.appendCh:
			if len(req) == 0 {
				continue
			}
			if _, err := c.request("logging.append", req); err != nil {
				log.Printf("sb-logging-sdk: append failed: %v", err)
			}
		case <-c.closeCh:
			for {
				select {
				case req := <-c.appendCh:
					if len(req) == 0 {
						continue
					}
					if _, err := c.request("logging.append", req); err != nil {
						log.Printf("sb-logging-sdk: append failed during close: %v", err)
					}
				default:
					return
				}
			}
		}
	}
}

func (c *client) noteDropped() {
	dropped := c.dropped.Add(1)
	now := time.Now().UnixNano()
	last := c.lastWarnAt.Load()
	if last != 0 && time.Duration(now-last) < appendWarningInterval {
		return
	}
	if !c.lastWarnAt.CompareAndSwap(last, now) {
		return
	}
	log.Printf("sb-logging-sdk: append queue full, dropped=%d", dropped)
}

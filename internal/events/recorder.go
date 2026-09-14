// Package events enqueues user-activity events into the events-service
// asynq queue (Redis DB = cfg.Redis.EventsQueueDB). The consumer contract
// (task name, queue name, JSON field names) mirrors
// services/events-service/internal/queue in the events-service repo:
//   - task:       event:activity
//   - queue:      events
//   - payload:    ActivityEventPayload (event_id, user_id, event_type,
//     stream_id, payload, occurred_at)
package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	TaskActivityEvent = "event:activity"
	TaskActivityQueue = "events"
)

// ActivityEventPayload is part of the producer contract with events-service.
type ActivityEventPayload struct {
	EventID   uuid.UUID       `json:"event_id"`
	UserID    uuid.UUID       `json:"user_id"`
	EventType string          `json:"event_type"`
	StreamID  *uuid.UUID      `json:"stream_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
}

func NewActivityEventTask(p ActivityEventPayload) (*asynq.Task, error) {
	payload, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskActivityEvent, payload), nil
}

func NewActivityClient(addr, password string, db int) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// Recorder enqueues activity events into the events queue. Enqueue is
// best-effort: callers should treat a returned error as non-fatal.
type Recorder struct {
	client *asynq.Client
}

func NewRecorder(client *asynq.Client) *Recorder {
	return &Recorder{client: client}
}

func (r *Recorder) Close() error {
	return r.client.Close()
}

func (r *Recorder) RecordActivity(ctx context.Context, userID uuid.UUID, eventType string, streamID *uuid.UUID, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task, err := NewActivityEventTask(ActivityEventPayload{
		EventID:    uuid.New(),
		UserID:     userID,
		EventType:  eventType,
		StreamID:   streamID,
		Payload:    raw,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	_, err = r.client.EnqueueContext(ctx, task,
		asynq.Queue(TaskActivityQueue),
		asynq.MaxRetry(3),
		asynq.Timeout(30*time.Second),
	)
	return err
}
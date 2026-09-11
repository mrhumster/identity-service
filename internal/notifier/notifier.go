package notifier

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/mrhumster/identity-service/internal/queue"
)

// Notifier enqueues email tasks into the asynq queue (consumed by
// mailer-service). Creation is best-effort: a failure is logged by the caller
// and never fails the underlying request.
type Notifier struct {
	client *asynq.Client
}

func NewNotifier(client *asynq.Client) *Notifier {
	return &Notifier{client: client}
}

func (n *Notifier) Close() error {
	return n.client.Close()
}

func (n *Notifier) EnqueueVerification(ctx context.Context, userID uuid.UUID, email, token string) error {
	task, err := queue.NewEmailVerificationTask(userID, email, token)
	if err != nil {
		return err
	}
	_, err = n.client.EnqueueContext(ctx, task,
		asynq.Queue(queue.TaskEmailQueue),
		asynq.MaxRetry(3),
		asynq.Timeout(30*time.Second),
	)
	return err
}
package queue

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// Task/queue names mirror the consumer contract in mailer-service. Tasks are
// enqueued into the asynq DB (cfg.Redis.QueueDB, default 2).
const (
	TaskEmailVerification = "email:verification"
	TaskEmailQueue        = "emails"
)

type EmailVerificationPayload struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Token  string    `json:"token"`
}

func NewEmailVerificationTask(userID uuid.UUID, email, token string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailVerificationPayload{
		UserID: userID,
		Email:  email,
		Token:  token,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskEmailVerification, payload), nil
}

func NewEmailClient(addr, password string, db int) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}
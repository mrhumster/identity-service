package response

import "time"

type Success struct {
	Message      string    `json:"message"`
	Generated_at time.Time `json:"generated_at"`
}

func SuccessResponse(message string) Success {
	return Success{
		Message:      message,
		Generated_at: time.Now(),
	}
}

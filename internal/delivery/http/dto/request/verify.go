package request

type VerifyRequest struct {
	Token string `json:"token" binding:"required"`
}
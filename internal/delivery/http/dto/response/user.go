package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/domain/models"
)

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UsersListReponse struct {
	Users []UserResponse `json:"users"`
	Total int64          `json:"total"`
	Page  int64          `json:"page"`
	Limit int64          `json:"limit"`
}

func (u *UserResponse) FillInTheModel(m *models.User) {
	u.ID = m.ID
	u.Email = m.Email
	u.CreatedAt = m.CreatedAt
	u.UpdatedAt = m.UpdatedAt
}

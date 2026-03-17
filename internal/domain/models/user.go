package models

import (
	"log"

	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	BaseModel
	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"not null" json:"-"`
	Role         string `gorm:"" json:"role"`
	TokenVersion string `gorm:"default:'v1'"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) SetPassword(password string) error {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hash := string(hashedBytes)
	u.PasswordHash = hash
	return nil
}

func (u *User) CheckPassword(password string) bool {
	hashedBytes := []byte(u.PasswordHash)
	err := bcrypt.CompareHashAndPassword(hashedBytes, []byte(password))
	return err == nil
}

func (u *User) FillInTheRequest(r request.UserRequest) {
	u.Email = r.Email
	u.SetPassword(r.Password)
}

func (u *User) FillInTheUpdateRequest(r request.UpdateUserRequest) {
	u.Email = r.Email
}

func (u *User) Debug() {
	log.Printf("\tEmail: %s", u.Email)
	log.Printf("\tPasswordHash: %s", u.PasswordHash)
}

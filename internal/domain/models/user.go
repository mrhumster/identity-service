package models

import (
	"log"
	"strings"

	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"golang.org/x/crypto/bcrypt"
)

// NormalizeEmail приводит email к каноническому виду: обрезает пробелы и
// переводит в нижний регистр, чтобы один аккаунт не зависел от регистра ввода.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

type User struct {
	BaseModel
	Email        string `gorm:"not null;uniqueIndex:idx_users_email_lower,expression:lower(email)" json:"email"`
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
	u.Email = NormalizeEmail(r.Email)
	u.SetPassword(r.Password)
}

func (u *User) FillInTheUpdateRequest(r request.UpdateUserRequest) {
	u.Email = NormalizeEmail(r.Email)
}

func (u *User) Debug() {
	log.Printf("\tEmail: %s", u.Email)
	log.Printf("\tPasswordHash: %s", u.PasswordHash)
}

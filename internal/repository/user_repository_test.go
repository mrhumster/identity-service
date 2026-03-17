package repository

import (
	"testing"

	mocks "github.com/mrhumster/identity-service/internal/repository/mock"
)

func TestUserRepositoryInterface(t *testing.T) {
	var _ UserRepository = (*GormUserRepository)(nil)
	var _ UserRepository = (*mocks.MockUserRepository)(nil)
}

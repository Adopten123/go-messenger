package service

import (
	"context"
	"fmt"

	"github.com/Adopten123/go-messenger/internal/repo/pgdb"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *pgdb.Queries
}

func NewUserService(repo *pgdb.Queries) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, email, username, password string) (pgdb.User, error) {
	// Main registration method
	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return pgdb.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	params := pgdb.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: string(passHash),
	}

	user, err := s.repo.CreateUser(ctx, params)
	if err != nil {
		return pgdb.User{}, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

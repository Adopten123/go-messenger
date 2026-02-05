package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Adopten123/go-messenger/internal/repo/pgdb"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/jackc/pgx/v5/pgconn"
)

type UserService struct {
	repo        *pgdb.Queries
	tokenSecret string
}

func NewUserService(repo *pgdb.Queries, tokenSecret string) *UserService {
	return &UserService{
		repo:        repo,
		tokenSecret: tokenSecret,
	}
}

func (s *UserService) CreateUser(ctx context.Context, email, username, password string) (*pgdb.User, error) {
	// Main registration method
	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return &pgdb.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	params := pgdb.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: string(passHash),
	}

	user, err := s.repo.CreateUser(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return nil, fmt.Errorf("user with this email already exists")
			}
		}

		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("invalid credentials: %w", err)
	}

	// Comparing passwords
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID.String(),
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	// Add secret key to token
	tokenString, err := token.SignedString([]byte(s.tokenSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

func (s *UserService) UpdateAvatar(ctx context.Context, userID pgtype.UUID, avatarURL string) error {
	return s.repo.UpdateUserAvatar(ctx, pgdb.UpdateUserAvatarParams{
		ID:        userID,
		AvatarUrl: pgtype.Text{String: avatarURL, Valid: true},
	})
}

func (s *UserService) GetUser(ctx context.Context, id string) (*pgdb.User, error) {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &user, nil
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (pgdb.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return pgdb.User{}, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}
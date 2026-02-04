package service

import (
	"context"
	"fmt"

	"github.com/Adopten123/go-messenger/internal/repo/pgdb"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatService struct {
	repo *pgdb.Queries
	pool *pgxpool.Pool
}

func NewChatService(repo *pgdb.Queries, pool *pgxpool.Pool) *ChatService {
	return &ChatService{
		repo: repo,
		pool: pool,
	}
}

func (s *ChatService) CreateChat(ctx context.Context, name string, creatorID string, userIDs []string) (*pgdb.Chat, error) {
	// 1. Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 2. Making transaction version of repo
	qtx := s.repo.WithTx(tx)

	// 3. Making chat
	isGroup := name != ""

	chatParams := pgdb.CreateChatParams{
		Name:    pgtype.Text{String: name, Valid: name != ""},
		IsGroup: isGroup,
	}

	chat, err := qtx.CreateChat(ctx, chatParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	// 4. Add members in chat (first invite for admin)
	var creatorUUID pgtype.UUID
	if err := creatorUUID.Scan(creatorID); err != nil {
		return nil, fmt.Errorf("invalid creator UUID: %w", err)
	}

	err = qtx.AddChatMember(ctx, pgdb.AddChatMemberParams{
		ChatID: chat.ID,
		UserID: creatorUUID,
		Role:   "admin",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to add creator: %w", err)
	}

	for _, uid := range userIDs {
		if uid == creatorID {
			continue
		}

		var memberUUID pgtype.UUID
		if err := memberUUID.Scan(uid); err != nil {
			return nil, fmt.Errorf("invalid member UUID %s: %w", uid, err)
		}

		err = qtx.AddChatMember(ctx, pgdb.AddChatMemberParams{
			ChatID: chat.ID,
			UserID: memberUUID,
			Role:   "member",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add member %s: %w", uid, err)
		}
	}

	// 5. Making transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return &chat, nil
}

// GetMessages - return chat history with pagination
func (s *ChatService) GetMessages(ctx context.Context, chatID string, limit, offset int) ([]pgdb.ListMessagesRow, error) {
	// 1. ID to UUID
	var ChatUUID pgtype.UUID
	if err := ChatUUID.Scan(chatID); err != nil {
		return nil, fmt.Errorf("invalid chat ID: %w", err)
	}

	// 2. Making request to DB
	params := pgdb.ListMessagesParams{
		ChatID: ChatUUID,
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	msgs, err := s.repo.ListMessages(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	return msgs, nil
}

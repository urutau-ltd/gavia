package session

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        string
	AccountID string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, tokenHash string, expiresAt time.Time) (*Session, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        id.String(),
		AccountID: "account",
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO user_sessions (id, account_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`,
		session.ID,
		session.AccountID,
		session.TokenHash,
		session.ExpiresAt,
		session.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (r *SessionRepository) GetValidByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*Session, error) {
	session := &Session{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, account_id, token_hash, expires_at, created_at
		FROM user_sessions
		WHERE token_hash = ? AND expires_at > ?
	`, tokenHash, now).Scan(
		&session.ID,
		&session.AccountID,
		&session.TokenHash,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return session, nil
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (r *SessionRepository) DeleteExpired(ctx context.Context, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE expires_at <= ?`, now)
	return err
}

func (r *SessionRepository) DeleteAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions`)
	return err
}

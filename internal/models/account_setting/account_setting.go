package accountsetting

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const singletonID = "account"

type AccountSettings struct {
	ID                string    `json:"id"`
	Username          string    `json:"username"`
	PasswordHash      string    `json:"password_hash"`
	APITokenHash      string    `json:"api_token_hash"`
	APITokenHint      string    `json:"api_token_hint"`
	AvatarPath        string    `json:"avatar_path"`
	RecoveryPublicKey string    `json:"recovery_public_key"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type AccountSettingsRepository struct {
	db *sql.DB
}

func NewAccountSettingsRepository(db *sql.DB) *AccountSettingsRepository {
	return &AccountSettingsRepository{db: db}
}

func (r *AccountSettingsRepository) Exists(ctx context.Context) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM account_settings WHERE id = ?)`,
		singletonID,
	).Scan(&exists)
	return exists, err
}

func (r *AccountSettingsRepository) Get(ctx context.Context) (*AccountSettings, error) {
	account := &AccountSettings{}
	err := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			username,
			password_hash,
			api_token_hash,
			api_token_hint,
			avatar_path,
			recovery_public_key,
			created_at,
			updated_at
		FROM account_settings
		WHERE id = ?
	`, singletonID).Scan(
		&account.ID,
		&account.Username,
		&account.PasswordHash,
		&account.APITokenHash,
		&account.APITokenHint,
		&account.AvatarPath,
		&account.RecoveryPublicKey,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return account, nil
}

func (r *AccountSettingsRepository) Create(ctx context.Context, account *AccountSettings) error {
	account.ID = singletonID
	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO account_settings (
			id,
			username,
			password_hash,
			api_token_hash,
			api_token_hint,
			avatar_path,
			recovery_public_key,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		account.ID,
		account.Username,
		account.PasswordHash,
		account.APITokenHash,
		account.APITokenHint,
		account.AvatarPath,
		account.RecoveryPublicKey,
		account.CreatedAt,
		account.UpdatedAt,
	)
	return err
}

func (r *AccountSettingsRepository) Update(ctx context.Context, account *AccountSettings) error {
	account.ID = singletonID
	account.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE account_settings
		SET
			username = ?,
			password_hash = ?,
			api_token_hash = ?,
			api_token_hint = ?,
			avatar_path = ?,
			recovery_public_key = ?,
			updated_at = ?
		WHERE id = ?
	`,
		account.Username,
		account.PasswordHash,
		account.APITokenHash,
		account.APITokenHint,
		account.AvatarPath,
		account.RecoveryPublicKey,
		account.UpdatedAt,
		account.ID,
	)
	return err
}

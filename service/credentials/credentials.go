package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"golang.org/x/crypto/scrypt"
)

type IntegrationType string

const (
	IntegrationSignal   IntegrationType = "signal"
	IntegrationProton   IntegrationType = "proton"
	IntegrationTelegram IntegrationType = "telegram"
)

type ProtonCredentials struct {
	Email        string            `json:"email"`
	Password     string            `json:"password,omitempty"`
	UID          string            `json:"uid,omitempty"`
	AccessToken  string            `json:"access_token,omitempty"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	Scope        string            `json:"scope,omitempty"`
	KeySalts     map[string][]byte `json:"key_salts,omitempty"`
	State        *ProtonState      `json:"state,omitempty"`
}

type ProtonState struct {
	LastEventID string `json:"last_event_id"`
}

type TelegramCredentials struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type SignalCredentials struct {
	Linked      bool   `json:"linked"`
	PhoneNumber string `json:"phone_number"`
}

type Store struct {
	db            *sql.DB
	encryptionKey []byte
	logger        *slog.Logger
}

func NewStore(db *sql.DB, masterPassword string) (*Store, error) {
	return NewStoreWithLogger(db, masterPassword, slog.Default())
}

func NewStoreWithLogger(db *sql.DB, masterPassword string, logger *slog.Logger) (*Store, error) {
	key, err := deriveKey(masterPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	store := &Store{
		db:            db,
		encryptionKey: key,
		logger:        logger,
	}

	if err := store.createTable(); err != nil {
		return nil, err
	}

	if err := store.CheckIntegrity(); err != nil {
		if strings.Contains(err.Error(), "corrupted") {
			logger.Warn("Credentials corrupted (API_KEY likely changed), clearing all integration credentials", "error", err)
			if clearErr := store.ClearAll(); clearErr != nil {
				logger.Error("Failed to clear corrupted credentials", "error", clearErr)
			} else {
				logger.Info("Cleared all integration credentials - please reconfigure integrations")
			}
		}
	}

	return store, nil
}

func (s *Store) createTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS integration_credentials (
integration_type TEXT PRIMARY KEY,
credentials_encrypted BLOB NOT NULL,
enabled BOOLEAN NOT NULL DEFAULT 1,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
	`
	_, err := s.db.Exec(query)
	return err
}

func deriveKey(password string) ([]byte, error) {
	salt := []byte("prism-integration-salt-v1")
	return scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
}

func (s *Store) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (s *Store) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (s *Store) SaveProton(creds *ProtonCredentials) error {
	return s.saveCredentials(IntegrationProton, creds)
}

func (s *Store) GetProton() (*ProtonCredentials, error) {
	var creds ProtonCredentials
	err := s.getCredentials(IntegrationProton, &creds)
	if err != nil {
		return nil, err
	}
	return &creds, nil
}

func (s *Store) SaveTelegram(creds *TelegramCredentials) error {
	return s.saveCredentials(IntegrationTelegram, creds)
}

func (s *Store) GetTelegram() (*TelegramCredentials, error) {
	var creds TelegramCredentials
	err := s.getCredentials(IntegrationTelegram, &creds)
	if err != nil {
		return nil, err
	}
	return &creds, nil
}

func (s *Store) SaveSignal(creds *SignalCredentials) error {
	return s.saveCredentials(IntegrationSignal, creds)
}

func (s *Store) GetSignal() (*SignalCredentials, error) {
	var creds SignalCredentials
	err := s.getCredentials(IntegrationSignal, &creds)
	if err != nil {
		return nil, err
	}
	return &creds, nil
}

func (s *Store) saveCredentials(integrationType IntegrationType, credentials interface{}) error {
	jsonData, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encrypted, err := s.encrypt(jsonData)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	query := `
		INSERT INTO integration_credentials (integration_type, credentials_encrypted, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(integration_type) DO UPDATE SET
			credentials_encrypted = excluded.credentials_encrypted,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err = s.db.Exec(query, string(integrationType), encrypted)
	return err
}

func (s *Store) getCredentials(integrationType IntegrationType, dest interface{}) error {
	query := `
		SELECT credentials_encrypted
		FROM integration_credentials
		WHERE integration_type = ? AND enabled = 1
	`
	var encrypted []byte
	err := s.db.QueryRow(query, string(integrationType)).Scan(&encrypted)
	if err == sql.ErrNoRows {
		return fmt.Errorf("integration %s not configured", integrationType)
	}
	if err != nil {
		return err
	}

	decrypted, err := s.decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	if err := json.Unmarshal(decrypted, dest); err != nil {
		return fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return nil
}

func (s *Store) DeleteIntegration(integrationType IntegrationType) error {
	query := `DELETE FROM integration_credentials WHERE integration_type = ?`
	_, err := s.db.Exec(query, string(integrationType))
	return err
}

func (s *Store) SetEnabled(integrationType IntegrationType, enabled bool) error {
	query := `UPDATE integration_credentials SET enabled = ? WHERE integration_type = ?`
	_, err := s.db.Exec(query, enabled, string(integrationType))
	return err
}

func (s *Store) IsEnabled(integrationType IntegrationType) (bool, error) {
	query := `SELECT enabled FROM integration_credentials WHERE integration_type = ?`
	var enabled bool
	err := s.db.QueryRow(query, string(integrationType)).Scan(&enabled)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

func (s *Store) ClearAll() error {
	query := `DELETE FROM integration_credentials`
	_, err := s.db.Exec(query)
	return err
}

func (s *Store) CheckIntegrity() error {
	query := `SELECT integration_type, credentials_encrypted FROM integration_credentials`
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var integrationType string
		var encrypted []byte
		if err := rows.Scan(&integrationType, &encrypted); err != nil {
			return err
		}

		if _, err := s.decrypt(encrypted); err != nil {
			return fmt.Errorf("credentials corrupted for %s (likely API_KEY changed): %w", integrationType, err)
		}
	}

	return rows.Err()
}

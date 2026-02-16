package notification

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// foreign_keys(1): enable FK constraints (disabled by default in SQLite)
	// _busy_timeout=5000: wait up to 5s when DB is locked (default=0, fails immediately)
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{db: db}
	if err := store.createTables(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS apps (
			appName TEXT PRIMARY KEY
		)`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id TEXT PRIMARY KEY,
			appName TEXT NOT NULL,
			channel TEXT NOT NULL,
			signalGroupId TEXT,
			signalAccount TEXT,
			telegramChatId TEXT,
			pushEndpoint TEXT,
			p256dh TEXT,
			auth TEXT,
			vapidPrivateKey TEXT,
			FOREIGN KEY(appName) REFERENCES apps(appName) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS signal_groups (
			appName TEXT PRIMARY KEY,
			groupId TEXT NOT NULL,
			account TEXT NOT NULL,
			FOREIGN KEY(appName) REFERENCES apps(appName) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_appName ON subscriptions(appName)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create tables: %w", err)
		}
	}

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) GetDB() *sql.DB {
	return s.db
}

func (s *Store) RegisterApp(appName string) error {
	query := `INSERT INTO apps (appName) VALUES (?) ON CONFLICT(appName) DO NOTHING`
	_, err := s.db.Exec(query, appName)
	return err
}

func (s *Store) AddSubscription(sub Subscription) error {
	if err := s.RegisterApp(sub.AppName); err != nil {
		return err
	}

	query := `
		INSERT INTO subscriptions (id, appName, channel, signalGroupId, signalAccount, telegramChatId, pushEndpoint, p256dh, auth, vapidPrivateKey)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var signalGroupID, signalAccount, telegramChatID, pushEndpoint, p256dh, auth, vapidPrivateKey *string

	if sub.Signal != nil {
		signalGroupID = &sub.Signal.GroupID
		signalAccount = &sub.Signal.Account
	}
	if sub.Telegram != nil {
		telegramChatID = &sub.Telegram.ChatID
	}
	if sub.WebPush != nil {
		pushEndpoint = &sub.WebPush.Endpoint
		p256dh = &sub.WebPush.P256dh
		auth = &sub.WebPush.Auth
		vapidPrivateKey = &sub.WebPush.VapidPrivateKey
	}

	_, err := s.db.Exec(query, sub.ID, sub.AppName, sub.Channel, signalGroupID, signalAccount, telegramChatID, pushEndpoint, p256dh, auth, vapidPrivateKey)
	return err
}

func (s *Store) GetApp(appName string) (*App, error) {
	query := `SELECT appName FROM apps WHERE appName = ?`
	row := s.db.QueryRow(query, appName)

	var app App
	if err := row.Scan(&app.AppName); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	subs, err := s.GetSubscriptions(appName)
	if err != nil {
		return nil, err
	}
	app.Subscriptions = subs

	return &app, nil
}

func (s *Store) GetSubscriptions(appName string) ([]Subscription, error) {
	query := `
		SELECT id, appName, channel, signalGroupId, signalAccount, telegramChatId, pushEndpoint, p256dh, auth, vapidPrivateKey
		FROM subscriptions
		WHERE appName = ?
	`
	rows, err := s.db.Query(query, appName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		var sub Subscription
		var signalGroupID, signalAccount, telegramChatID, pushEndpoint, p256dh, auth, vapidPrivateKey sql.NullString

		if err := rows.Scan(&sub.ID, &sub.AppName, &sub.Channel, &signalGroupID, &signalAccount, &telegramChatID, &pushEndpoint, &p256dh, &auth, &vapidPrivateKey); err != nil {
			return nil, err
		}

		if signalGroupID.Valid && signalAccount.Valid {
			sub.Signal = &SignalSubscription{
				GroupID: signalGroupID.String,
				Account: signalAccount.String,
			}
		}
		if telegramChatID.Valid {
			sub.Telegram = &TelegramSubscription{
				ChatID: telegramChatID.String,
			}
		}
		if pushEndpoint.Valid {
			sub.WebPush = &WebPushSubscription{
				Endpoint: pushEndpoint.String,
			}
			if p256dh.Valid {
				sub.WebPush.P256dh = p256dh.String
			}
			if auth.Valid {
				sub.WebPush.Auth = auth.String
			}
			if vapidPrivateKey.Valid {
				sub.WebPush.VapidPrivateKey = vapidPrivateKey.String
			}
		}

		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, rows.Err()
}

func (s *Store) GetAllApps() ([]App, error) {
	query := `SELECT appName FROM apps ORDER BY appName`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var app App
		if err := rows.Scan(&app.AppName); err != nil {
			return nil, err
		}

		subs, err := s.GetSubscriptions(app.AppName)
		if err != nil {
			return nil, err
		}
		app.Subscriptions = subs

		apps = append(apps, app)
	}

	return apps, rows.Err()
}

func (s *Store) GetSignalGroup(appName string) (*SignalSubscription, error) {
	query := `SELECT groupId, account FROM signal_groups WHERE appName = ?`
	row := s.db.QueryRow(query, appName)

	var groupID, account string
	if err := row.Scan(&groupID, &account); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &SignalSubscription{
		GroupID: groupID,
		Account: account,
	}, nil
}

func (s *Store) SaveSignalGroup(appName string, sub *SignalSubscription) error {
	if err := s.RegisterApp(appName); err != nil {
		return err
	}

	query := `INSERT INTO signal_groups (appName, groupId, account) VALUES (?, ?, ?)
			  ON CONFLICT(appName) DO UPDATE SET groupId=excluded.groupId, account=excluded.account`
	_, err := s.db.Exec(query, appName, sub.GroupID, sub.Account)
	return err
}

func (s *Store) DeleteSubscription(subscriptionID string) error {
	query := `DELETE FROM subscriptions WHERE id = ?`
	_, err := s.db.Exec(query, subscriptionID)
	return err
}

func (s *Store) GetSubscription(subscriptionID string) (*Subscription, error) {
	query := `
		SELECT id, appName, channel, signalGroupId, signalAccount, telegramChatId, pushEndpoint, p256dh, auth, vapidPrivateKey
		FROM subscriptions
		WHERE id = ?
	`
	row := s.db.QueryRow(query, subscriptionID)

	var sub Subscription
	var signalGroupID, signalAccount, telegramChatID, pushEndpoint, p256dh, auth, vapidPrivateKey sql.NullString

	err := row.Scan(&sub.ID, &sub.AppName, &sub.Channel, &signalGroupID, &signalAccount, &telegramChatID, &pushEndpoint, &p256dh, &auth, &vapidPrivateKey)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if signalGroupID.Valid && signalAccount.Valid {
		sub.Signal = &SignalSubscription{
			GroupID: signalGroupID.String,
			Account: signalAccount.String,
		}
	}
	if telegramChatID.Valid {
		sub.Telegram = &TelegramSubscription{
			ChatID: telegramChatID.String,
		}
	}
	if pushEndpoint.Valid {
		sub.WebPush = &WebPushSubscription{
			Endpoint: pushEndpoint.String,
		}
		if p256dh.Valid {
			sub.WebPush.P256dh = p256dh.String
		}
		if auth.Valid {
			sub.WebPush.Auth = auth.String
		}
		if vapidPrivateKey.Valid {
			sub.WebPush.VapidPrivateKey = vapidPrivateKey.String
		}
	}

	return &sub, nil
}

func (s *Store) RemoveApp(appName string) error {
	_, err := s.db.Exec(`DELETE FROM apps WHERE appName = ?`, appName)
	return err
}

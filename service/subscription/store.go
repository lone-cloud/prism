package subscription

import (
	"database/sql"
	"fmt"
)

type Store struct {
	DB *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
	store := &Store{DB: db}
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
			id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
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
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_appName ON subscriptions(appName)`,
	}

	for _, query := range queries {
		if _, err := s.DB.Exec(query); err != nil {
			return fmt.Errorf("failed to create tables: %w", err)
		}
	}

	return nil
}

func (s *Store) RegisterApp(appName string) error {
	_, err := s.DB.Exec(`INSERT INTO apps (appName) VALUES (?) ON CONFLICT(appName) DO NOTHING`, appName)
	return err
}

func (s *Store) AddSubscription(sub Subscription) (string, error) {
	if err := s.RegisterApp(sub.AppName); err != nil {
		return "", err
	}

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

	var id string
	err := s.DB.QueryRow(`
		INSERT INTO subscriptions (appName, channel, signalGroupId, signalAccount, telegramChatId, pushEndpoint, p256dh, auth, vapidPrivateKey)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id
	`, sub.AppName, sub.Channel, signalGroupID, signalAccount, telegramChatID, pushEndpoint, p256dh, auth, vapidPrivateKey).Scan(&id)

	return id, err
}

func (s *Store) GetApp(appName string) (*App, error) {
	row := s.DB.QueryRow(`SELECT appName FROM apps WHERE appName = ?`, appName)

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
	rows, err := s.DB.Query(`
		SELECT id, appName, channel, signalGroupId, signalAccount, telegramChatId, pushEndpoint, p256dh, auth, vapidPrivateKey
		FROM subscriptions
		WHERE appName = ?
	`, appName)
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
	rows, err := s.DB.Query(`SELECT appName FROM apps ORDER BY appName`)
	if err != nil {
		return nil, err
	}

	appNames := make([]string, 0)
	for rows.Next() {
		var appName string
		if err := rows.Scan(&appName); err != nil {
			_ = rows.Close()
			return nil, err
		}
		appNames = append(appNames, appName)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	apps := make([]App, 0, len(appNames))
	for _, appName := range appNames {
		app := App{AppName: appName}

		subs, err := s.GetSubscriptions(appName)
		if err != nil {
			return nil, err
		}
		app.Subscriptions = subs

		apps = append(apps, app)
	}

	return apps, nil
}

func (s *Store) DeleteSubscription(subscriptionID string) error {
	_, err := s.DB.Exec(`DELETE FROM subscriptions WHERE id = ?`, subscriptionID)
	return err
}

func (s *Store) DeleteSubscriptionsByChannel(channel Channel) error {
	_, err := s.DB.Exec(`DELETE FROM subscriptions WHERE channel = ?`, channel)
	return err
}

func (s *Store) GetSubscription(subscriptionID string) (*Subscription, error) {
	row := s.DB.QueryRow(`
		SELECT id, appName, channel, signalGroupId, signalAccount, telegramChatId, pushEndpoint, p256dh, auth, vapidPrivateKey
		FROM subscriptions
		WHERE id = ?
	`, subscriptionID)

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
	_, err := s.DB.Exec(`DELETE FROM apps WHERE appName = ?`, appName)
	return err
}

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

	db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000&cache=shared")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)

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
	query := `
		CREATE TABLE IF NOT EXISTS mappings (
			appName TEXT PRIMARY KEY,
			signalGroupId TEXT,
			signalAccount TEXT,
			channel TEXT NOT NULL DEFAULT 'webpush',
			pushEndpoint TEXT,
			p256dh TEXT,
			auth TEXT,
			vapidPrivateKey TEXT
		)
	`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Register(appName string, channel *Channel, signal *SignalSubscription, webPush *WebPushSubscription) error {
	var signalGroupID, signalAccount *string
	if signal != nil {
		signalGroupID = &signal.GroupID
		signalAccount = &signal.Account
	}

	var pushEndpoint, p256dh, auth, vapidPrivateKey *string
	if webPush != nil {
		pushEndpoint = &webPush.Endpoint
		p256dh = &webPush.P256dh
		auth = &webPush.Auth
		vapidPrivateKey = &webPush.VapidPrivateKey
	}

	ch := ChannelWebPush
	if channel != nil {
		ch = *channel
	}

	query := `
		INSERT INTO mappings (appName, signalGroupId, signalAccount, channel, pushEndpoint, p256dh, auth, vapidPrivateKey)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(appName) DO UPDATE SET
			channel = excluded.channel,
			signalGroupId = COALESCE(excluded.signalGroupId, mappings.signalGroupId),
			signalAccount = COALESCE(excluded.signalAccount, mappings.signalAccount),
			pushEndpoint = COALESCE(excluded.pushEndpoint, mappings.pushEndpoint),
			p256dh = COALESCE(excluded.p256dh, mappings.p256dh),
			auth = COALESCE(excluded.auth, mappings.auth),
			vapidPrivateKey = COALESCE(excluded.vapidPrivateKey, mappings.vapidPrivateKey)
	`
	_, err := s.db.Exec(query, appName, signalGroupID, signalAccount, ch, pushEndpoint, p256dh, auth, vapidPrivateKey)
	return err
}

func (s *Store) RegisterDefault(appName string, availableChannels []Channel) error {
	existing, _ := s.GetApp(appName)
	if existing != nil && (existing.Signal != nil || existing.WebPush != nil) {
		return fmt.Errorf("app %s already exists with subscriptions, refusing to overwrite", appName)
	}

	var channel Channel
	if len(availableChannels) > 0 {
		channel = availableChannels[0]
	} else {
		channel = ChannelWebPush
	}

	return s.Register(appName, &channel, nil, nil)
}

func (s *Store) GetApp(appName string) (*Mapping, error) {
	query := `
		SELECT appName, signalGroupId, signalAccount, channel, pushEndpoint, p256dh, auth, vapidPrivateKey
		FROM mappings
		WHERE appName = ?
	`
	row := s.db.QueryRow(query, appName)

	var m Mapping
	var signalGroupID, signalAccount, pushEndpoint, p256dh, auth, vapidPrivateKey sql.NullString
	err := row.Scan(&m.AppName, &signalGroupID, &signalAccount, &m.Channel, &pushEndpoint, &p256dh, &auth, &vapidPrivateKey)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if signalGroupID.Valid && signalAccount.Valid {
		m.Signal = &SignalSubscription{
			GroupID: signalGroupID.String,
			Account: signalAccount.String,
		}
	}
	if pushEndpoint.Valid {
		m.WebPush = &WebPushSubscription{
			Endpoint: pushEndpoint.String,
		}
		if p256dh.Valid {
			m.WebPush.P256dh = p256dh.String
		}
		if auth.Valid {
			m.WebPush.Auth = auth.String
		}
		if vapidPrivateKey.Valid {
			m.WebPush.VapidPrivateKey = vapidPrivateKey.String
		}
	}

	return &m, nil
}

func (s *Store) GetAllMappings() ([]Mapping, error) {
	query := `
		SELECT appName, signalGroupId, signalAccount, channel, pushEndpoint, p256dh, auth, vapidPrivateKey
		FROM mappings
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []Mapping
	for rows.Next() {
		var m Mapping
		var signalGroupID, signalAccount, pushEndpoint, p256dh, auth, vapidPrivateKey sql.NullString
		if err := rows.Scan(&m.AppName, &signalGroupID, &signalAccount, &m.Channel, &pushEndpoint, &p256dh, &auth, &vapidPrivateKey); err != nil {
			return nil, err
		}

		if signalGroupID.Valid && signalAccount.Valid {
			m.Signal = &SignalSubscription{
				GroupID: signalGroupID.String,
				Account: signalAccount.String,
			}
		}
		if pushEndpoint.Valid {
			m.WebPush = &WebPushSubscription{
				Endpoint: pushEndpoint.String,
			}
			if p256dh.Valid {
				m.WebPush.P256dh = p256dh.String
			}
			if auth.Valid {
				m.WebPush.Auth = auth.String
			}
			if vapidPrivateKey.Valid {
				m.WebPush.VapidPrivateKey = vapidPrivateKey.String
			}
		}

		mappings = append(mappings, m)
	}

	return mappings, rows.Err()
}

func (s *Store) UpdateChannel(appName string, channel Channel) error {
	query := `UPDATE mappings SET channel = ? WHERE appName = ?`
	_, err := s.db.Exec(query, channel, appName)
	return err
}

func (s *Store) UpdateSignal(appName string, signal *SignalSubscription) error {
	if signal == nil {
		return fmt.Errorf("signal cannot be nil")
	}

	query := `UPDATE mappings SET signalGroupId = ?, signalAccount = ? WHERE appName = ?`
	_, err := s.db.Exec(query, signal.GroupID, signal.Account, appName)
	return err
}

func (s *Store) UpdateWebPush(appName string, webPush *WebPushSubscription) error {
	if webPush == nil {
		return fmt.Errorf("webPush cannot be nil")
	}

	query := `
		UPDATE mappings 
		SET pushEndpoint = ?, p256dh = ?, auth = ?, vapidPrivateKey = ?
		WHERE appName = ?
	`
	_, err := s.db.Exec(query, webPush.Endpoint, webPush.P256dh, webPush.Auth, webPush.VapidPrivateKey, appName)
	return err
}

func (s *Store) RemoveApp(appName string) error {
	query := `DELETE FROM mappings WHERE appName = ?`
	_, err := s.db.Exec(query, appName)
	return err
}

func (s *Store) ClearWebPush(appName string) error {
	query := `
		UPDATE mappings 
		SET pushEndpoint = NULL, p256dh = NULL, auth = NULL, vapidPrivateKey = NULL, channel = 'signal'
		WHERE appName = ?
	`
	_, err := s.db.Exec(query, appName)
	return err
}

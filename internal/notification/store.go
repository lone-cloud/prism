package notification

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
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
	query := `
		CREATE TABLE IF NOT EXISTS mappings (
			endpoint TEXT PRIMARY KEY,
			groupId TEXT,
			appName TEXT NOT NULL,
			channel TEXT NOT NULL DEFAULT 'signal',
			upEndpoint TEXT
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

func (s *Store) Register(endpoint, appName string, channel Channel, groupID, upEndpoint *string) error {
	query := `
		INSERT OR IGNORE INTO mappings (endpoint, groupId, appName, channel, upEndpoint)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query, endpoint, groupID, appName, channel, upEndpoint)
	return err
}

func (s *Store) GetMapping(endpoint string) (*Mapping, error) {
	query := `
		SELECT endpoint, groupId, appName, channel, upEndpoint
		FROM mappings
		WHERE endpoint = ?
	`
	row := s.db.QueryRow(query, endpoint)

	var m Mapping
	var groupID, upEndpoint sql.NullString
	err := row.Scan(&m.Endpoint, &groupID, &m.AppName, &m.Channel, &upEndpoint)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if groupID.Valid {
		m.GroupID = &groupID.String
	}
	if upEndpoint.Valid {
		m.UpEndpoint = &upEndpoint.String
	}

	return &m, nil
}

func (s *Store) GetAllMappings() ([]Mapping, error) {
	query := `
		SELECT endpoint, groupId, appName, channel, upEndpoint
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
		var groupID, upEndpoint sql.NullString
		if err := rows.Scan(&m.Endpoint, &groupID, &m.AppName, &m.Channel, &upEndpoint); err != nil {
			return nil, err
		}

		if groupID.Valid {
			m.GroupID = &groupID.String
		}
		if upEndpoint.Valid {
			m.UpEndpoint = &upEndpoint.String
		}

		mappings = append(mappings, m)
	}

	return mappings, rows.Err()
}

func (s *Store) UpdateChannel(endpoint string, channel Channel) error {
	query := `UPDATE mappings SET channel = ? WHERE endpoint = ?`
	_, err := s.db.Exec(query, channel, endpoint)
	return err
}

func (s *Store) UpdateGroupID(endpoint string, groupID string) error {
	query := `UPDATE mappings SET groupId = ? WHERE endpoint = ?`
	_, err := s.db.Exec(query, groupID, endpoint)
	return err
}

func (s *Store) RemoveEndpoint(endpoint string) error {
	query := `DELETE FROM mappings WHERE endpoint = ?`
	_, err := s.db.Exec(query, endpoint)
	return err
}

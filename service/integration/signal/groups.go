package signal

import (
	"database/sql"
	"fmt"

	"prism/service/subscription"
)

type GroupCache struct {
	db *sql.DB
}

func NewGroupCache(db *sql.DB) (*GroupCache, error) {
	c := &GroupCache{db: db}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS signal_groups (
		appName TEXT PRIMARY KEY,
		groupId TEXT NOT NULL,
		account TEXT NOT NULL,
		FOREIGN KEY(appName) REFERENCES apps(appName) ON DELETE CASCADE
	)`)
	if err != nil {
		return nil, fmt.Errorf("failed to create signal_groups table: %w", err)
	}
	return c, nil
}

func (c *GroupCache) Get(appName string) (*subscription.SignalSubscription, error) {
	var groupID, account string
	err := c.db.QueryRow(`SELECT groupId, account FROM signal_groups WHERE appName = ?`, appName).Scan(&groupID, &account)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &subscription.SignalSubscription{GroupID: groupID, Account: account}, nil
}

func (c *GroupCache) Save(appName string, sub *subscription.SignalSubscription) error {
	_, err := c.db.Exec(
		`INSERT INTO signal_groups (appName, groupId, account) VALUES (?, ?, ?)
		 ON CONFLICT(appName) DO UPDATE SET groupId=excluded.groupId, account=excluded.account`,
		appName, sub.GroupID, sub.Account,
	)
	return err
}

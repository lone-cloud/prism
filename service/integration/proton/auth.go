package proton

import (
	"fmt"

	"prism/service/credentials"

	"github.com/emersion/hydroxide/protonmail"
)

func (m *Monitor) authenticateAndSetup(credStore *credentials.Store) error {
	creds, err := credStore.GetProton()
	if err != nil {
		m.logger.Debug("Proton credentials not configured", "error", err)
		return nil
	}

	m.credStore = credStore
	m.logger.Info("Starting Proton Mail monitor", "email", creds.Email)

	c := &protonmail.Client{
		RootURL:    "https://mail.proton.me/api",
		AppVersion: "Other",
	}

	var auth *protonmail.Auth

	if creds.UID != "" && creds.AccessToken != "" && creds.RefreshToken != "" {
		auth = &protonmail.Auth{
			UID:          creds.UID,
			AccessToken:  creds.AccessToken,
			RefreshToken: creds.RefreshToken,
			Scope:        creds.Scope,
		}

		_, err = c.Unlock(auth, creds.KeySalts, creds.Password)
		if err != nil {
			m.logger.Error("Failed to unlock keys - password may have changed", "error", err)
			if deleteErr := credStore.DeleteIntegration(credentials.IntegrationProton); deleteErr != nil {
				m.logger.Error("Failed to clear invalid credentials", "error", deleteErr)
			}
			return fmt.Errorf("failed to unlock keys (password changed?): %v", err)
		}

		m.logger.Info("Restored Proton session from stored tokens")
	} else if creds.Password != "" {
		authInfo, err := c.AuthInfo(creds.Email)
		if err != nil {
			return err
		}

		authResult, err := c.Auth(creds.Email, creds.Password, authInfo)
		if err != nil {
			return err
		}
		auth = authResult

		keySalts, err := c.ListKeySalts()
		if err != nil {
			return fmt.Errorf("failed to get key salts: %v", err)
		}

		_, err = c.Unlock(auth, keySalts, creds.Password)
		if err != nil {
			m.logger.Error("Failed to unlock keys", "error", err)
			if deleteErr := credStore.DeleteIntegration(credentials.IntegrationProton); deleteErr != nil {
				m.logger.Error("Failed to clear invalid credentials", "error", deleteErr)
			}
			return fmt.Errorf("failed to unlock keys: %v", err)
		}

		creds.KeySalts = keySalts
		if err := credStore.SaveProton(creds); err != nil {
			m.logger.Warn("Failed to cache key salts", "error", err)
		}

		m.logger.Info("Authenticated and unlocked Proton session")
	} else {
		return fmt.Errorf("no valid credentials found - need password or tokens")
	}

	m.setupTokenRefresh(c, auth, creds)
	m.client = c

	if creds.State != nil {
		m.eventID = creds.State.LastEventID
		m.logger.Info("Restored Proton state")
	} else {
		m.eventID = auth.EventID
		if err := m.saveState(creds); err != nil {
			m.logger.Warn("Failed to save initial state", "error", err)
		}
		m.logger.Info("Initialized Proton state")
	}

	return nil
}

func (m *Monitor) setupTokenRefresh(c *protonmail.Client, auth *protonmail.Auth, creds *credentials.ProtonCredentials) {
	c.ReAuth = func() error {
		m.logger.Info("Refreshing Proton session tokens")
		newAuth, err := c.AuthRefresh(auth)
		if err != nil {
			m.logger.Error("Token refresh failed", "error", err)
			return err
		}

		_, err = c.Unlock(newAuth, creds.KeySalts, creds.Password)
		if err != nil {
			m.logger.Error("Token refresh failed - cannot unlock keys", "error", err)
			return err
		}

		auth = newAuth

		updatedCreds, err := m.credStore.GetProton()
		if err != nil {
			m.logger.Warn("Failed to get credentials for token update", "error", err)
			return nil
		}

		updatedCreds.UID = newAuth.UID
		updatedCreds.AccessToken = newAuth.AccessToken
		updatedCreds.RefreshToken = newAuth.RefreshToken
		updatedCreds.Scope = newAuth.Scope

		if err := m.credStore.SaveProton(updatedCreds); err != nil {
			m.logger.Warn("Failed to save refreshed tokens", "error", err)
		} else {
			m.logger.Info("Proton tokens refreshed and saved")
		}

		return nil
	}
}

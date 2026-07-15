package internal

import (
	"fmt"
	"strings"
	"time"
)

const (
	accountStatusChecking = "checking"
	accountStatusInvalid  = "invalid"
	accountStatusStale    = "stale"
	accountStatusTesting  = "testing"
	accountStatusUnknown  = "unknown"
	accountStatusValid    = "valid"
)

const defaultAccountValidityTTL = 24 * time.Hour

func configuredAccountValidityTTL() time.Duration {
	if Cfg != nil && Cfg.AccountValidityTTL > 0 {
		return Cfg.AccountValidityTTL
	}
	return defaultAccountValidityTTL
}

func (s *Store) BeginAccountTest(id string) error {
	return s.beginAccountCheck(id, accountStatusTesting, true)
}

func (s *Store) BeginAccountAttempt(id string) error {
	return s.beginAccountCheck(id, accountStatusChecking, false)
}

func (s *Store) beginAccountCheck(id, status string, updateTestTime bool) error {
	now := nowISO()
	query := `UPDATE qianwen_accounts SET status=?, last_error='', updated_at=? WHERE id=?`
	args := []interface{}{status, now, id}
	if updateTestTime {
		query = `UPDATE qianwen_accounts SET status=?, last_error='', last_test_at=?, updated_at=? WHERE id=?`
		args = []interface{}{status, now, now, id}
	}
	result, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("account %s not found", id)
	}
	return nil
}

func (s *Store) RecoverInterruptedAccountChecks() error {
	_, err := s.db.Exec(`UPDATE qianwen_accounts
		SET status='unknown', last_error='Account validation was interrupted by a service restart.', updated_at=?
		WHERE status IN ('testing','checking')`, nowISO())
	return err
}

func (s *Store) ExpireStaleAccounts(now time.Time, ttl time.Duration) (int64, error) {
	if ttl <= 0 {
		ttl = defaultAccountValidityTTL
	}
	cutoff := now.UTC().Add(-ttl).Format(time.RFC3339)
	message := fmt.Sprintf("Last successful validation is older than %s; run account test again.", ttl)
	result, err := s.db.Exec(`UPDATE qianwen_accounts
		SET status='stale',
			last_error=CASE WHEN COALESCE(last_error,'')='' THEN ? ELSE last_error END,
			updated_at=?
		WHERE status='valid' AND (COALESCE(last_success_at,'')='' OR last_success_at < ?)`,
		message, now.UTC().Format(time.RFC3339), cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) refreshAccountReadiness() error {
	if err := s.QuarantineAccountsWithoutLoginMaterial(); err != nil {
		return err
	}
	_, err := s.ExpireStaleAccounts(time.Now(), configuredAccountValidityTTL())
	return err
}

func (s *Store) QuarantineAccountsWithoutLoginMaterial() error {
	rows, err := s.db.Query(`SELECT id, type, COALESCE(cookie_json,''), COALESCE(cookie_string,'')
		FROM qianwen_accounts
		WHERE enabled=1 AND type!='guest' AND status NOT IN ('invalid','maintenance_pending_validation')`)
	if err != nil {
		return err
	}
	var accountIDs []string
	for rows.Next() {
		var account AccountRecord
		if err := rows.Scan(&account.ID, &account.Type, &account.CookieJSON, &account.CookieString); err != nil {
			_ = rows.Close()
			return err
		}
		if !accountHasQianwenLoginMaterial(account) {
			accountIDs = append(accountIDs, account.ID)
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, id := range accountIDs {
		if err := s.UpdateAccountRuntimeFailure(id, accountStatusInvalid,
			"Account has no strong qianwen.com login ticket; scan and capture login again."); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateAccountRuntimeFailure(id, status, lastError string) error {
	status = strings.TrimSpace(status)
	if status != accountStatusInvalid {
		status = accountStatusUnknown
	}
	_, err := s.db.Exec(`UPDATE qianwen_accounts SET status=?, last_error=?, updated_at=? WHERE id=?`,
		status, lastError, nowISO(), id)
	return err
}

func (s *Store) IsAccountRunnable(account AccountRecord, capability string) (bool, error) {
	if !account.Enabled || account.Status != accountStatusValid {
		return false, nil
	}
	if strings.TrimSpace(account.CookieJSON) == "" && strings.TrimSpace(account.CookieString) == "" {
		return false, nil
	}
	if account.Type != "guest" && !accountHasQianwenLoginMaterial(account) {
		return false, nil
	}
	inMaintenance, err := s.IsAccountInMaintenance(account.ID)
	if err != nil {
		return false, err
	}
	return !inMaintenance && accountSupportsCapability(account, capability), nil
}

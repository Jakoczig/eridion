package history

import (
	"database/sql"
	"encoding/json"
	"errors"
)

type ProviderDisk struct {
	Accounts      map[string]map[string]json.RawMessage `json:"accounts,omitempty"`
	ActiveAccount string                                `json:"activeAccount,omitempty"`
	Providers     map[string]json.RawMessage            `json:"providers"`
	Ledger        map[string]int64                      `json:"ledger"`
	Uploader      string                                `json:"uploader"`
}

// validateProviderDisk is the shared bounds contract for persisted provider
// state, used by ordinary saves and by restore sanitization.
func validateProviderDisk(disk ProviderDisk) error {
	if len(disk.Providers) > 32 || len(disk.Ledger) > 4096 || len(disk.Accounts) > 64 {
		return errors.New("Invalid provider state")
	}
	for account, providers := range disk.Accounts {
		if len(account) != 64 || len(providers) > 32 {
			return errors.New("Invalid account consent")
		}
		for id, cfg := range providers {
			if len(id) > 32 || len(cfg) > 16384 || !json.Valid(cfg) {
				return errors.New("Invalid account provider configuration")
			}
		}
	}
	return nil
}

// pausedConsent returns cfg with uploads disabled and the provider paused,
// preserving every other field (read preference, categories, identity).
func pausedConsent(cfg json.RawMessage) (json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(cfg, &fields); err != nil || fields == nil {
		return nil, errors.New("Invalid provider configuration in backup")
	}
	fields["upload"] = json.RawMessage("false")
	fields["paused"] = json.RawMessage("true")
	return json.Marshal(fields)
}

// pauseRestoredConsent runs inside the staged restore transaction. It
// requires renewed consent for every account namespace, not only the active
// rows: publication policies become Never, exceptions are removed, pending
// candidates are held, and the active provider rows plus every account in
// the provider ledger are paused with uploads disabled. Credentials stay in
// their OS vault namespaces; dedupe/history counters are retained.
func pauseRestoredConsent(tx *sql.Tx) error {
	if _, err := tx.Exec(`UPDATE publication_policy SET mode='Never'; DELETE FROM publication_exceptions; UPDATE publication_candidates SET status='held',reason='Restored backup requires renewed consent' WHERE status IN ('eligible','queued');`); err != nil {
		return err
	}
	rows, err := tx.Query("SELECT provider,config_json FROM provider_consent")
	if err != nil {
		return err
	}
	active := map[string]json.RawMessage{}
	for rows.Next() {
		var id string
		var raw []byte
		if err = rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return err
		}
		active[id] = json.RawMessage(raw)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for id, cfg := range active {
		paused, err := pausedConsent(cfg)
		if err != nil {
			return err
		}
		if _, err = tx.Exec("UPDATE provider_consent SET config_json=?,updated_at=? WHERE provider=?", string(paused), utc(), id); err != nil {
			return err
		}
	}
	var b []byte
	var hash string
	err = tx.QueryRow("SELECT value,content_hash FROM app_state WHERE key='provider-ledger'").Scan(&b, &hash)
	if err == sql.ErrNoRows || len(b) == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	if digest(b) != hash {
		return errors.New("Provider ledger checksum mismatch")
	}
	// Decode the ledger generically so fields this version does not model
	// survive; only account provider configurations are rewritten.
	var ledger map[string]json.RawMessage
	if err = json.Unmarshal(b, &ledger); err != nil {
		return errors.New("Invalid provider ledger in backup")
	}
	var accounts map[string]map[string]json.RawMessage
	if raw, ok := ledger["accounts"]; ok {
		if err = json.Unmarshal(raw, &accounts); err != nil {
			return errors.New("Invalid provider ledger in backup")
		}
	}
	for _, providers := range accounts {
		for id, cfg := range providers {
			if providers[id], err = pausedConsent(cfg); err != nil {
				return err
			}
		}
	}
	if accounts != nil {
		if ledger["accounts"], err = json.Marshal(accounts); err != nil {
			return err
		}
	}
	out, err := json.Marshal(ledger)
	if err != nil {
		return err
	}
	var check ProviderDisk
	if err = json.Unmarshal(out, &check); err != nil {
		return errors.New("Invalid provider ledger in backup")
	}
	if err = validateProviderDisk(check); err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE app_state SET value=?,content_hash=?,updated_at=? WHERE key='provider-ledger'", out, digest(out), utc())
	return err
}

func (s *Store) LoadProviderConsent(initial []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var marker int
	err := s.writer.QueryRow("SELECT count(*) FROM app_state WHERE key='provider-consent-migrated-v1'").Scan(&marker)
	if err != nil {
		return nil, err
	}
	if marker == 0 {
		if err = s.saveConsentLocked(initial); err != nil {
			return nil, err
		}
	}
	disk := ProviderDisk{Providers: map[string]json.RawMessage{}, Ledger: map[string]int64{}}
	rows, err := s.writer.Query("SELECT provider,config_json FROM provider_consent ORDER BY provider LIMIT 33")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		var raw []byte
		if err = rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		disk.Providers[id] = json.RawMessage(raw)
	}
	err = rows.Err()
	rows.Close()
	if err != nil || len(disk.Providers) > 32 {
		return nil, errors.New("Invalid provider consent database")
	}
	var b []byte
	var hash string
	if err = s.writer.QueryRow("SELECT value,content_hash FROM app_state WHERE key='provider-ledger'").Scan(&b, &hash); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if len(b) > 0 {
		if digest(b) != hash {
			return nil, errors.New("Provider ledger checksum mismatch")
		}
		var ledger ProviderDisk
		if err = json.Unmarshal(b, &ledger); err != nil {
			return nil, err
		}
		disk.Accounts = ledger.Accounts
		disk.ActiveAccount = ledger.ActiveAccount
		disk.Ledger = ledger.Ledger
		disk.Uploader = ledger.Uploader
	}
	return json.Marshal(disk)
}
func (s *Store) SaveProviderConsent(raw []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveConsentLocked(raw)
}
func (s *Store) saveConsentLocked(raw []byte) error {
	if len(raw) > 1<<20 {
		return errors.New("Consent budget exceeded")
	}
	var disk ProviderDisk
	if json.Unmarshal(raw, &disk) != nil {
		return errors.New("Invalid provider state")
	}
	if err := validateProviderDisk(disk); err != nil {
		return err
	}
	tx, err := s.writer.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for id, cfg := range disk.Providers {
		if len(id) > 32 || len(cfg) > 16384 || !json.Valid(cfg) {
			return errors.New("Invalid provider configuration")
		}
		if _, err = tx.Exec("INSERT INTO provider_consent VALUES(?,?,?) ON CONFLICT(provider) DO UPDATE SET config_json=excluded.config_json,updated_at=excluded.updated_at", id, string(cfg), utc()); err != nil {
			return err
		}
	}
	disk.Providers = nil
	b, _ := json.Marshal(disk)
	for key, value := range map[string][]byte{"provider-ledger": b, "provider-consent-migrated-v1": []byte("1")} {
		if _, err = tx.Exec("INSERT INTO app_state VALUES(?,?,?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value,content_hash=excluded.content_hash,updated_at=excluded.updated_at", key, value, digest(value), utc()); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) SingleConsent(id string, initial []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b []byte
	err := s.writer.QueryRow("SELECT config_json FROM provider_consent WHERE provider=?", id).Scan(&b)
	if err == sql.ErrNoRows {
		if !json.Valid(initial) {
			return nil, errors.New("Invalid consent")
		}
		_, err = s.writer.Exec("INSERT INTO provider_consent VALUES(?,?,?)", id, string(initial), utc())
		return initial, err
	}
	return b, err
}
func (s *Store) SetSingleConsent(id string, raw []byte) error {
	if len(id) > 32 || len(raw) > 16384 || !json.Valid(raw) {
		return errors.New("Invalid consent")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.writer.Exec("INSERT INTO provider_consent VALUES(?,?,?) ON CONFLICT(provider) DO UPDATE SET config_json=excluded.config_json,updated_at=excluded.updated_at", id, string(raw), utc())
	return err
}

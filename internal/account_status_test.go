package internal

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	oldCfg := Cfg
	oldStore := AppStore
	dir := t.TempDir()

	Cfg = &Config{
		DataDir:           dir,
		DatabasePath:      filepath.Join(dir, "qianwen-test.sqlite"),
		DefaultChatModel:  "tongyi-qwen3-max-model",
		DefaultImageModel: "Qwen-Image-2.0",
		DefaultVideoModel: QianwenVideoModelID,
	}
	if err := InitStore(); err != nil {
		t.Fatalf("InitStore() error = %v", err)
	}
	store := AppStore
	t.Cleanup(func() {
		if store != nil && store.db != nil {
			_ = store.db.Close()
		}
		Cfg = oldCfg
		AppStore = oldStore
	})
	return store
}

func createRunnableTestAccount(t *testing.T, store *Store) AccountRecord {
	t.Helper()

	account := AccountRecord{
		Name:             "test-login",
		Type:             "login_cookie",
		Status:           "valid",
		Enabled:          true,
		CookieString:     "XSRF-TOKEN=test-xsrf; qwen-test-cookie=value",
		CapabilitiesJSON: `{"chat":true,"image":true,"video":true}`,
	}
	if err := store.CreateAccount(&account); err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}
	return account
}

func TestRecordAccountTaskFailurePreservesRunnableStatus(t *testing.T) {
	store := newTestStore(t)
	account := createRunnableTestAccount(t, store)

	message := "qianwen video generation completed without a media URL"
	if err := store.RecordAccountTaskFailure(account.ID, message); err != nil {
		t.Fatalf("RecordAccountTaskFailure() error = %v", err)
	}

	got, err := store.GetAccount(account.ID)
	if err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}
	if got.Status != "valid" {
		t.Fatalf("status = %q, want valid", got.Status)
	}
	if !strings.Contains(got.LastError, message) {
		t.Fatalf("last_error = %q, want to contain %q", got.LastError, message)
	}

	accounts, err := store.ListRunnableAccountsForCapability("video")
	if err != nil {
		t.Fatalf("ListRunnableAccountsForCapability() error = %v", err)
	}
	if len(accounts) != 1 || accounts[0].ID != account.ID {
		t.Fatalf("runnable accounts = %+v, want account %s", accounts, account.ID)
	}
}

func TestRecordQianwenProviderFailureOnlyInvalidatesAuthErrors(t *testing.T) {
	store := newTestStore(t)
	account := createRunnableTestAccount(t, store)

	recordQianwenProviderFailure(account.ID, errors.New("qianwen image generation did not return media url before timeout"))
	got, err := store.GetAccount(account.ID)
	if err != nil {
		t.Fatalf("GetAccount() after task failure error = %v", err)
	}
	if got.Status != "valid" {
		t.Fatalf("status after task failure = %q, want valid", got.Status)
	}

	recordQianwenProviderFailure(account.ID, errors.New(`qianwen upstream status 403: {"code":"EX015","msg":"signature error"}`))
	got, err = store.GetAccount(account.ID)
	if err != nil {
		t.Fatalf("GetAccount() after signature failure error = %v", err)
	}
	if got.Status != "valid" {
		t.Fatalf("status after signature failure = %q, want valid", got.Status)
	}

	recordQianwenProviderFailure(account.ID, errors.New("qianwen upstream status 401: unauthorized"))
	got, err = store.GetAccount(account.ID)
	if err != nil {
		t.Fatalf("GetAccount() after auth failure error = %v", err)
	}
	if got.Status != "invalid" {
		t.Fatalf("status after auth failure = %q, want invalid", got.Status)
	}
}

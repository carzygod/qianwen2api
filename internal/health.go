package internal

import (
	"net/http"
	"time"
)

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	payload, _ := providerHealthPayload()
	payload["ok"] = true
	writeJSON(w, http.StatusOK, payload)
}

func HandleReadiness(w http.ResponseWriter, r *http.Request) {
	payload, ready := providerHealthPayload()
	payload["ok"] = ready
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, payload)
}

func providerHealthPayload() (map[string]interface{}, bool) {
	storeReady := AppStore != nil
	guestCount := GuestPoolAccountCount()
	guestReady := Cfg.PoolSize > 0 && guestCount > 0 && GuestPoolInitError == ""
	accountStatus := map[string]int{}
	freshValid := 0
	if storeReady {
		if accounts, err := AppStore.ListAccounts(); err == nil {
			for _, account := range accounts {
				accountStatus[account.Status]++
				runnable, runnableErr := AppStore.IsAccountRunnable(account, "")
				if runnableErr != nil {
					storeReady = false
					continue
				}
				if runnable {
					freshValid++
				}
			}
		} else {
			storeReady = false
		}
	}
	ready := storeReady && (freshValid > 0 || guestReady)
	return map[string]interface{}{
		"service":                   "QIANWEN-WEB-01",
		"ready":                     ready,
		"store_ready":               storeReady,
		"fresh_valid_account_count": freshValid,
		"account_status":            accountStatus,
		"account_validity_hours":    configuredAccountValidityTTL() / time.Hour,
		"guest_ready":               guestReady,
		"guest_pool_size":           Cfg.PoolSize,
		"guest_pool_count":          guestCount,
		"guest_pool_error":          GuestPoolInitError,
		"data_dir":                  Cfg.DataDir,
		"database":                  Cfg.DatabasePath,
	}, ready
}

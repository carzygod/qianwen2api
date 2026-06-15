package internal

import "net/http"

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	storeReady := AppStore != nil
	guestReady := GlobalPool != nil && Cfg.PoolSize > 0
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":              true,
		"service":         "QIANWEN-WEB-01",
		"store_ready":     storeReady,
		"guest_ready":     guestReady,
		"guest_pool_size": Cfg.PoolSize,
		"data_dir":        Cfg.DataDir,
		"database":        Cfg.DatabasePath,
	})
}

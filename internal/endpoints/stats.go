package endpoints

import (
	"net/http"
)

// StatsRetrieve holds a given stat
type StatsRetrieve struct {
	Message string `json:"message"`
}

func Stats(w http.ResponseWriter, r *http.Request) {

	// stat := r.URL.Query().Get("stat")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Statuses"))
}

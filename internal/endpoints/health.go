package endpoints

import (
	"net/http"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"gorm.io/gorm"
)

// HealthCheckHandler checks for active DB connection and operational API
func HealthCheckHandler(db *gorm.DB, cfg config.TrackerConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d, err := db.DB()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err = d.Ping(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

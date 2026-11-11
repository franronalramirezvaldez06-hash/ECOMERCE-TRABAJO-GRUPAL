package httpapi

// Handlers de reportes (solo administrador).

import (
	"net/http"
	"time"

	"ecommerce/internal/apperr"
)

// salesReport: ?from=YYYY-MM-DD&to=YYYY-MM-DD (to incluido). Por defecto, ultimos 30 dias.
func (a *API) salesReport(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	from := to.AddDate(0, 0, -30)
	var err error
	if v := r.URL.Query().Get("from"); v != "" {
		if from, err = time.Parse("2006-01-02", v); err != nil {
			a.writeError(w, apperr.Validation("from debe tener formato YYYY-MM-DD"))
			return
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		last, perr := time.Parse("2006-01-02", v)
		if perr != nil {
			a.writeError(w, apperr.Validation("to debe tener formato YYYY-MM-DD"))
			return
		}
		to = last.AddDate(0, 0, 1)
	}
	if !to.After(from) {
		a.writeError(w, apperr.Validation("el rango de fechas es invalido"))
		return
	}
	s, err := a.Orders.Summary(r.Context(), from, to)
	respond(a, w, http.StatusOK, s, err)
}

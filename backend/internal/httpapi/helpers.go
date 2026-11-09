package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"ecommerce/internal/apperr"
)

func respond(a *API, w http.ResponseWriter, okStatus int, body any, err error) {
	if err != nil {
		a.writeError(w, err)
		return
	}
	writeJSON(w, okStatus, body)
}

func (a *API) writeError(w http.ResponseWriter, err error) {
	var ve *apperr.ValidationError
	switch {
	case errors.As(err, &ve):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": ve.Msg})
	case errors.Is(err, apperr.ErrUnauthorized):
		w.Header().Set("WWW-Authenticate", `Bearer realm="ecommerce"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
	case errors.Is(err, apperr.ErrForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
	case errors.Is(err, apperr.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, apperr.ErrConflict), errors.Is(err, apperr.ErrInsufficientStock):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, apperr.ErrGateway):
		a.Log.Error("error de pasarela", "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "la pasarela de pago no pudo completar la operacion"})
	default:
		a.Log.Error("error interno", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "error interno del servidor"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON invalido: " + err.Error()})
		return false
	}
	return true
}

func paging(r *http.Request) (limit, offset int) {
	limit, offset = 50, 0
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = min(v, 200)
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v > 0 {
		offset = v
	}
	return
}

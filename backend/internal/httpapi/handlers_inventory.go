package httpapi

// Handlers de inventario (solo administrador).

import (
	"net/http"
	"strconv"
)

func (a *API) getStock(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	q, err := a.Inventory.Get(r.Context(), id)
	respond(a, w, http.StatusOK, map[string]any{"product_id": id, "quantity": q}, err)
}

func (a *API) setStock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Quantity int `json:"quantity"`
	}
	if !decode(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	q, err := a.Inventory.Set(r.Context(), id, in.Quantity)
	respond(a, w, http.StatusOK, map[string]any{"product_id": id, "quantity": q}, err)
}

func (a *API) adjustStock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Delta  int    `json:"delta"`
		Reason string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	q, err := a.Inventory.Adjust(r.Context(), id, in.Delta, in.Reason)
	respond(a, w, http.StatusOK, map[string]any{"product_id": id, "quantity": q}, err)
}

func (a *API) stockMovements(w http.ResponseWriter, r *http.Request) {
	limit, _ := paging(r)
	ms, err := a.Inventory.Movements(r.Context(), r.PathValue("id"), limit)
	respond(a, w, http.StatusOK, ms, err)
}

func (a *API) lowStock(w http.ResponseWriter, r *http.Request) {
	threshold := 5
	if v, err := strconv.Atoi(r.URL.Query().Get("threshold")); err == nil && v >= 0 {
		threshold = v
	}
	limit, _ := paging(r)
	items, err := a.Inventory.LowStock(r.Context(), threshold, limit)
	respond(a, w, http.StatusOK, items, err)
}

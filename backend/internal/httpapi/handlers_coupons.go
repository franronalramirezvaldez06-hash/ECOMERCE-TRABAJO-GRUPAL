package httpapi

// Handlers de cupones de descuento (solo administrador).

import (
	"net/http"

	"ecommerce/internal/apperr"
	"ecommerce/internal/coupon"
)

func (a *API) listCoupons(w http.ResponseWriter, r *http.Request) {
	cs, err := a.Coupons.List(r.Context())
	a.ok(w, cs, err)
}

func (a *API) createCoupon(w http.ResponseWriter, r *http.Request) {
	var in coupon.CreateInput
	if !decode(w, r, &in) {
		return
	}
	c, err := a.Coupons.Create(r.Context(), in)
	a.created(w, c, err)
}

func (a *API) setCouponActive(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active *bool `json:"active"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Active == nil {
		a.writeError(w, apperr.Validation("active es obligatorio"))
		return
	}
	err := a.Coupons.SetActive(r.Context(), r.PathValue("code"), *in.Active)
	a.ok(w, map[string]any{"code": r.PathValue("code"), "active": *in.Active}, err)
}

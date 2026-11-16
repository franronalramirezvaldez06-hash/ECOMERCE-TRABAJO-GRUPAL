package httpapi

// Handlers de autenticacion: registro, login y perfil.

import (
	"net/http"

	"ecommerce/internal/auth"
)

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var in auth.RegisterInput
	if !decode(w, r, &in) {
		return
	}
	u, err := a.Auth.Register(r.Context(), in)
	a.created(w, u, err)
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	res, err := a.Auth.Login(r.Context(), in.Email, in.Password)
	a.ok(w, res, err)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	u, err := a.Auth.Get(r.Context(), p.UserID)
	a.ok(w, u, err)
}

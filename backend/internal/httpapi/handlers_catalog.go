package httpapi

// Handlers de catalogo: productos, combos (Composite) y categorias.

import (
	"net/http"

	"ecommerce/internal/apperr"
	"ecommerce/internal/auth"
	"ecommerce/internal/catalog"
)

func isAdmin(r *http.Request) bool {
	p, ok := auth.FromContext(r.Context())
	return ok && p.IsAdmin()
}

func (a *API) listProducts(w http.ResponseWriter, r *http.Request) {
	limit, offset := paging(r)
	q := r.URL.Query()
	if k := q.Get("kind"); k != "" && k != "simple" && k != "bundle" {
		a.writeError(w, apperr.Validation("tipo de producto invalido"))
		return
	}
	if sort := q.Get("sort"); sort != "" && sort != "newest" && sort != "name" {
		a.writeError(w, apperr.Validation("orden invalido"))
		return
	}
	recs, err := a.Catalog.List(r.Context(), catalog.ListFilter{
		Query:           q.Get("q"),
		CategoryID:      q.Get("category_id"),
		Kind:            q.Get("kind"),
		Sort:            q.Get("sort"),
		InStock:         q.Get("in_stock") == "true",
		IncludeInactive: q.Get("include_inactive") == "true" && isAdmin(r),
		Limit:           limit,
		Offset:          offset,
	})
	a.ok(w, recs, err)
}

func (a *API) getProduct(w http.ResponseWriter, r *http.Request) {
	d, err := a.Catalog.Detail(r.Context(), r.PathValue("id"), isAdmin(r))
	a.ok(w, d, err)
}

func (a *API) createProduct(w http.ResponseWriter, r *http.Request) {
	var in catalog.CreateSimpleInput
	if !decode(w, r, &in) {
		return
	}
	rec, err := a.Catalog.CreateSimple(r.Context(), in)
	a.created(w, rec, err)
}

func (a *API) updateProduct(w http.ResponseWriter, r *http.Request) {
	var in catalog.UpdateInput
	if !decode(w, r, &in) {
		return
	}
	rec, err := a.Catalog.Update(r.Context(), r.PathValue("id"), in)
	a.ok(w, rec, err)
}

func (a *API) createBundle(w http.ResponseWriter, r *http.Request) {
	var in catalog.CreateBundleInput
	if !decode(w, r, &in) {
		return
	}
	rec, err := a.Catalog.CreateBundle(r.Context(), in)
	a.created(w, rec, err)
}

func (a *API) listCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := a.Catalog.ListCategories(r.Context())
	a.ok(w, cats, err)
}

func (a *API) createCategory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	c, err := a.Catalog.CreateCategory(r.Context(), in.Name)
	a.created(w, c, err)
}

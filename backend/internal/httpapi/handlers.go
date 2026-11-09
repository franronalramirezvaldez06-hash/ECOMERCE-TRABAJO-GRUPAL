package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"ecommerce/internal/apperr"
	"ecommerce/internal/auth"
	"ecommerce/internal/catalog"
	"ecommerce/internal/coupon"
	"ecommerce/internal/sales"
)

// ----------------------------------------------------------------- auth

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var in auth.RegisterInput
	if !decode(w, r, &in) {
		return
	}
	u, err := a.Auth.Register(r.Context(), in)
	respond(a, w, http.StatusCreated, u, err)
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
	respond(a, w, http.StatusOK, res, err)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	u, err := a.Auth.Get(r.Context(), p.UserID)
	respond(a, w, http.StatusOK, u, err)
}

// ------------------------------------------------------------- catalogo

func isAdmin(r *http.Request) bool {
	p, ok := auth.FromContext(r.Context())
	return ok && p.IsAdmin()
}

func (a *API) listProducts(w http.ResponseWriter, r *http.Request) {
	limit, offset := paging(r)
	q := r.URL.Query()
	recs, err := a.Catalog.List(r.Context(), catalog.ListFilter{
		Query:           q.Get("q"),
		CategoryID:      q.Get("category_id"),
		IncludeInactive: q.Get("include_inactive") == "true" && isAdmin(r),
		Limit:           limit,
		Offset:          offset,
	})
	respond(a, w, http.StatusOK, recs, err)
}

func (a *API) getProduct(w http.ResponseWriter, r *http.Request) {
	d, err := a.Catalog.Detail(r.Context(), r.PathValue("id"), isAdmin(r))
	respond(a, w, http.StatusOK, d, err)
}

func (a *API) createProduct(w http.ResponseWriter, r *http.Request) {
	var in catalog.CreateSimpleInput
	if !decode(w, r, &in) {
		return
	}
	rec, err := a.Catalog.CreateSimple(r.Context(), in)
	respond(a, w, http.StatusCreated, rec, err)
}

func (a *API) updateProduct(w http.ResponseWriter, r *http.Request) {
	var in catalog.UpdateInput
	if !decode(w, r, &in) {
		return
	}
	rec, err := a.Catalog.Update(r.Context(), r.PathValue("id"), in)
	respond(a, w, http.StatusOK, rec, err)
}

func (a *API) createBundle(w http.ResponseWriter, r *http.Request) {
	var in catalog.CreateBundleInput
	if !decode(w, r, &in) {
		return
	}
	rec, err := a.Catalog.CreateBundle(r.Context(), in)
	respond(a, w, http.StatusCreated, rec, err)
}

func (a *API) listCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := a.Catalog.ListCategories(r.Context())
	respond(a, w, http.StatusOK, cats, err)
}

func (a *API) createCategory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	c, err := a.Catalog.CreateCategory(r.Context(), in.Name)
	respond(a, w, http.StatusCreated, c, err)
}

// ----------------------------------------------------------- inventario

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

// -------------------------------------------------------------- cupones

func (a *API) listCoupons(w http.ResponseWriter, r *http.Request) {
	cs, err := a.Coupons.List(r.Context())
	respond(a, w, http.StatusOK, cs, err)
}

func (a *API) createCoupon(w http.ResponseWriter, r *http.Request) {
	var in coupon.CreateInput
	if !decode(w, r, &in) {
		return
	}
	c, err := a.Coupons.Create(r.Context(), in)
	respond(a, w, http.StatusCreated, c, err)
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
	respond(a, w, http.StatusOK, map[string]any{"code": r.PathValue("code"), "active": *in.Active}, err)
}

// --------------------------------------------------------------- ventas

func (a *API) quote(w http.ResponseWriter, r *http.Request) {
	var in sales.QuoteInput
	if !decode(w, r, &in) {
		return
	}
	b, err := a.Sales.Quote(r.Context(), in)
	respond(a, w, http.StatusOK, b, err)
}

func (a *API) checkout(w http.ResponseWriter, r *http.Request) {
	var in sales.CheckoutInput
	if !decode(w, r, &in) {
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) > 255 {
		a.writeError(w, apperr.Validation("Idempotency-Key demasiado largo (maximo 255)"))
		return
	}
	p, _ := auth.FromContext(r.Context())
	order, err := a.Sales.Checkout(r.Context(), sales.CheckoutParams{
		UserID: p.UserID, Email: p.Email, IdempotencyKey: key, Input: in,
	})
	if err != nil {
		a.writeError(w, err)
		return
	}
	status := http.StatusCreated
	if order.Status == sales.StatusFailed {
		status = http.StatusPaymentRequired
	}
	writeJSON(w, status, order)
}

func (a *API) listOrders(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	userID := p.UserID // los clientes solo ven sus ordenes
	if p.IsAdmin() {
		userID = r.URL.Query().Get("user_id")
	}
	limit, offset := paging(r)
	orders, err := a.Orders.List(r.Context(), userID, r.URL.Query().Get("status"), limit, offset)
	respond(a, w, http.StatusOK, orders, err)
}

// ownedOrder carga la orden y oculta (404) las ajenas a un cliente.
func (a *API) ownedOrder(w http.ResponseWriter, r *http.Request) (*sales.Order, bool) {
	order, err := a.Orders.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.writeError(w, err)
		return nil, false
	}
	p, _ := auth.FromContext(r.Context())
	if !p.IsAdmin() && order.UserID != p.UserID {
		a.writeError(w, apperr.ErrNotFound)
		return nil, false
	}
	return order, true
}

func (a *API) getOrder(w http.ResponseWriter, r *http.Request) {
	if order, ok := a.ownedOrder(w, r); ok {
		writeJSON(w, http.StatusOK, order)
	}
}

// cancelOrder: el cliente cancela su orden pagada (reembolso total, stock repuesto).
func (a *API) cancelOrder(w http.ResponseWriter, r *http.Request) {
	order, ok := a.ownedOrder(w, r)
	if !ok {
		return
	}
	o, err := a.Sales.Refund(r.Context(), order.ID, "cancelada por el cliente", sales.StatusCancelled)
	respond(a, w, http.StatusOK, o, err)
}

func (a *API) refundOrder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Reason string `json:"reason"`
	}
	if r.ContentLength != 0 && !decode(w, r, &in) {
		return
	}
	if in.Reason == "" {
		in.Reason = "reembolso por administrador"
	}
	o, err := a.Sales.Refund(r.Context(), r.PathValue("id"), in.Reason, sales.StatusRefunded)
	respond(a, w, http.StatusOK, o, err)
}

func (a *API) shipOrder(w http.ResponseWriter, r *http.Request) {
	o, err := a.Sales.MarkShipped(r.Context(), r.PathValue("id"))
	respond(a, w, http.StatusOK, o, err)
}

func (a *API) deliverOrder(w http.ResponseWriter, r *http.Request) {
	o, err := a.Sales.MarkDelivered(r.Context(), r.PathValue("id"))
	respond(a, w, http.StatusOK, o, err)
}

func (a *API) listGateways(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"gateways": a.Gateways.Names()})
}

// -------------------------------------------------------------- reportes

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

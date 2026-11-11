package httpapi

// Handlers de ventas: cotizacion, checkout, ordenes y pasarelas.

import (
	"net/http"

	"ecommerce/internal/apperr"
	"ecommerce/internal/auth"
	"ecommerce/internal/sales"
)

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
	reason := "cancelada por el cliente"
	if isAdmin(r) {
		reason = "cancelada por el administrador"
	}
	o, err := a.Sales.Refund(r.Context(), order.ID, reason, sales.StatusCancelled)
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

func (a *API) approveOrder(w http.ResponseWriter, r *http.Request) {
	o, err := a.Sales.Approve(r.Context(), r.PathValue("id"))
	respond(a, w, http.StatusOK, o, err)
}

func (a *API) deliverOrder(w http.ResponseWriter, r *http.Request) {
	o, err := a.Sales.MarkDelivered(r.Context(), r.PathValue("id"))
	respond(a, w, http.StatusOK, o, err)
}

func (a *API) listGateways(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"gateways": a.Gateways.Names()})
}

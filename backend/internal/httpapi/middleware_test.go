package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecommerce/internal/auth"
)

func testAPI() *API {
	return &API{Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func TestLimiterBlocksBurstAndRecovers(t *testing.T) {
	l := newLimiter(1, 2)
	now := time.Now()
	if !l.allow("ip", now) || !l.allow("ip", now) {
		t.Fatal("la rafaga inicial debio pasar")
	}
	if l.allow("ip", now) {
		t.Fatal("la tercera peticion inmediata debio bloquearse")
	}
	if !l.allow("ip", now.Add(1100*time.Millisecond)) {
		t.Fatal("tras 1s debio recuperar un token")
	}
	if !l.allow("otra-ip", now) {
		t.Fatal("cada IP tiene su propio cupo")
	}
}

func TestRoleGuards(t *testing.T) {
	a := testAPI()
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

	cases := []struct {
		name  string
		guard http.HandlerFunc
		p     *auth.Principal
		want  int
	}{
		{"admin sin sesion", a.requireAdmin(ok), nil, http.StatusUnauthorized},
		{"admin con cliente", a.requireAdmin(ok), &auth.Principal{UserID: "u", Role: auth.RoleCustomer}, http.StatusForbidden},
		{"admin con admin", a.requireAdmin(ok), &auth.Principal{UserID: "u", Role: auth.RoleAdmin}, http.StatusOK},
		{"auth sin sesion", a.requireAuth(ok), nil, http.StatusUnauthorized},
		{"auth con cliente", a.requireAuth(ok), &auth.Principal{UserID: "u", Role: auth.RoleCustomer}, http.StatusOK},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if c.p != nil {
			req = req.WithContext(auth.WithPrincipal(req.Context(), *c.p))
		}
		rec := httptest.NewRecorder()
		c.guard(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: esperaba %d, obtuvo %d", c.name, c.want, rec.Code)
		}
	}
}

func TestCORSPreflightAndOrigins(t *testing.T) {
	a := testAPI()
	a.Options.AllowedOrigins = []string{"https://tienda.com"}
	h := a.cors(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/orders", nil)
	req.Header.Set("Origin", "https://tienda.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "https://tienda.com" {
		t.Fatalf("preflight permitido incorrecto: %d %v", rec.Code, rec.Header())
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://malo.com")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("un origen no permitido no debe recibir cabeceras CORS")
	}
}

func TestPagingBounds(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?limit=9999&offset=-5", nil)
	limit, offset := paging(req)
	if limit != 200 || offset != 0 {
		t.Fatalf("limit=%d offset=%d", limit, offset)
	}
}

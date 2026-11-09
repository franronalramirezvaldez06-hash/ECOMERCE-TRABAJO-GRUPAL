package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"ecommerce/internal/apperr"
	"ecommerce/internal/auth"
)

// ------------------------------------------------------- request id / log

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" || len(id) > 64 {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (a *API) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		a.Log.Info("request",
			"id", w.Header().Get("X-Request-Id"), "method", r.Method, "path", r.URL.Path,
			"status", rec.status, "duration", time.Since(start).String())
	})
}

func (a *API) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				a.Log.Error("panic recuperado", "panic", rec, "path", r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "error interno del servidor"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// ------------------------------------------------------------------ CORS

func (a *API) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	any := false
	for _, o := range a.Options.AllowedOrigins {
		if o == "*" {
			any = true
		}
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (any || allowed[origin]) {
			h := w.Header()
			if any {
				h.Set("Access-Control-Allow-Origin", "*")
			} else {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Add("Vary", "Origin")
			}
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-Id")
			h.Set("Access-Control-Expose-Headers", "X-Request-Id")
			h.Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ----------------------------------------------------------- rate limit

type bucket struct {
	tokens float64
	last   time.Time
}

// limiter es un token bucket por IP, en memoria.
type limiter struct {
	mu        sync.Mutex
	rate      float64
	burst     float64
	clients   map[string]*bucket
	lastClean time.Time
}

func newLimiter(rps float64, burst int) *limiter {
	if rps <= 0 {
		rps = 20
	}
	if burst <= 0 {
		burst = 60
	}
	return &limiter{rate: rps, burst: float64(burst), clients: map[string]*bucket{}, lastClean: time.Now()}
}

func (l *limiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastClean) > time.Minute {
		for k, b := range l.clients {
			if now.Sub(b.last) > 10*time.Minute {
				delete(l.clients, k)
			}
		}
		l.lastClean = now
	}
	b, ok := l.clients[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.clients[key] = b
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *limiter) wrap(trustProxy bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions && !l.allow(clientIP(r, trustProxy), time.Now()) {
			w.Header().Set("Retry-After", "1")
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "demasiadas peticiones, intenta de nuevo en un momento"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ------------------------------------------------------- autenticacion

// authenticate lee el token Bearer (si viene) y deja la identidad en el contexto.
func (a *API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			next.ServeHTTP(w, r)
			return
		}
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok {
			a.writeError(w, apperr.ErrUnauthorized)
			return
		}
		p, err := a.Auth.Authenticate(r.Context(), strings.TrimSpace(token))
		if err != nil {
			a.writeError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
	})
}

func (a *API) requireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.FromContext(r.Context()); !ok {
			a.writeError(w, apperr.ErrUnauthorized)
			return
		}
		h(w, r)
	}
}

func (a *API) requireAdmin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.FromContext(r.Context())
		if !ok {
			a.writeError(w, apperr.ErrUnauthorized)
			return
		}
		if !p.IsAdmin() {
			a.writeError(w, apperr.ErrForbidden)
			return
		}
		h(w, r)
	}
}

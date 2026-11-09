# Ecommerce — Backend (Grupo 3)

Sistema ecommerce: **ventas, inventarios y pasarelas de pago**.
Monolito modular en **Go**, **PostgreSQL en Docker**, patrones **Composite** y **Decorator**.
Documentación de la API: [`docs/openapi.yaml`](docs/openapi.yaml) (ábrelo en <https://editor.swagger.io>).

## Arranque rápido

```bash
cp .env.example .env
docker compose up -d --build        # postgres + api en http://localhost:8080
curl localhost:8080/health
```

En desarrollo se crea un administrador `admin@ecommerce.local` / `Admin12345!` (cámbialo con `ADMIN_EMAIL` / `ADMIN_PASSWORD`).

Desarrollo local (API con `go run`, solo la base en Docker):

```bash
make db
go mod tidy          # primera vez: genera go.sum
make run
```

## Pruebas

```bash
make test               # unitarias (Composite, Decorator, JWT, Stripe con servidor falso, middleware)
make db && make test-integration   # + integración contra PostgreSQL real
```

Las de integración cubren: venta de combos con descuento de inventario de las hojas, reembolso que repone stock,
pago rechazado que devuelve stock y cupón, stock insuficiente, idempotencia, **10 compras concurrentes por 3 unidades (nunca se sobrevende)**,
flujo envío/entrega y registro/login.

## Estructura

```
cmd/api/main.go            composición de dependencias y arranque (graceful shutdown)
migrations/                esquema SQL versionado, embebido en el binario
internal/
  auth/                    usuarios, bcrypt, JWT HS256, roles admin/customer
  catalog/                 COMPOSITE: productos simples y combos + categorías
  pricing/                 DECORATOR: subtotal -> cupón -> envío -> impuesto
  payment/                 DECORATOR: pasarelas (Stripe real + simuladas) con logging/retry/timeout/límite
  coupon/                  cupones en BD (vigencia, usos máximos)
  inventory/               existencias, movimientos, alertas de bajo stock, reserva atómica
  sales/                   checkout, reembolsos, cancelación, envío, reportes
  httpapi/                 REST: auth, CORS, rate limit, request-id, errores
  platform/database/       pool pgx y migraciones
  config/ apperr/          configuración validada y errores de negocio
docs/openapi.yaml          contrato de la API
```

## Patrones de diseño

**Composite** — `internal/catalog/composite.go`. `Component` es la interfaz común; `Simple` es la hoja y `Bundle`
el compuesto (un combo puede contener productos u otros combos). `Price()`, `Available()` y `Requirements()` se resuelven
recursivamente, así el checkout descuenta inventario de los productos simples sin importar el anidamiento.

**Decorator** — dos usos:
- `pricing`: `Chain(Base, WithTax, WithShipping, WithCoupons)`; cada decorador agrega su renglón al desglose.
- `payment`: `Decorate(pasarela, WithLogging, WithRetry, WithTimeout, WithAmountLimit)`. Una pasarela nueva solo implementa `Gateway`.

## Flujo de una venta (consistencia)

1. Se arma el árbol Composite y se calcula el precio (cadena Decorator).
2. **Transacción:** crea la orden `pending`, consume el cupón y descuenta inventario con `UPDATE ... WHERE quantity >= n` (atómico, sin sobreventa; bloqueo en orden fijo para evitar deadlocks).
3. Cobro en la pasarela (con reintentos solo ante errores temporales; Stripe recibe `Idempotency-Key`).
4. Aprobado → `paid`. Rechazado → compensación: se devuelve stock y cupón, orden `failed` (HTTP 402).
5. Si el cobro se aprueba pero falla guardarlo, se intenta reembolsar automáticamente.

`Idempotency-Key` en `POST /orders` hace seguros los reintentos del cliente (misma orden, sin doble cobro ni doble descuento).

Ciclo de vida: `pending → paid → shipped → delivered`; `paid → refunded | cancelled` (reembolso total + stock repuesto, atómico: si la pasarela falla la orden sigue `paid`).

## Roles y seguridad

| Rol | Puede |
|---|---|
| Público | ver catálogo/categorías, cotizar, ver pasarelas, registrarse, iniciar sesión |
| `customer` | comprar, ver **solo sus** órdenes, cancelar su orden pagada |
| `admin` | gestionar productos, combos, categorías, inventario, cupones; reembolsar, enviar/entregar; reportes |

JWT HS256 con expiración, contraseñas con bcrypt, login resistente a enumeración de cuentas, límite de peticiones por IP
(estricto en login/registro), CORS por lista blanca, cabeceras de seguridad, cuerpos limitados a 1 MB, campos desconocidos rechazados.
En `APP_ENV=production` la configuración se valida al arrancar (JWT_SECRET ≥ 32 caracteres, etc.).

## Pasarelas de pago

- **Stripe real:** define `STRIPE_SECRET_KEY` (usa `sk_test_...` para pruebas). El `payment_token` es el id de un PaymentMethod creado en el frontend con Stripe.js (`pm_...`; para probar: `pm_card_visa`, `pm_card_chargeDeclined`).
- **Simuladas** (`stripe`, `paypal`, `mercadopago`; no cobran): activas por defecto solo en desarrollo. Tokens: `tok_ok`, `tok_declined`, `tok_flaky` (falla 2 veces y reintenta), `tok_slow` (timeout).
- PayPal y MercadoPago reales no están implementados; agregarlos = implementar `payment.Gateway` y registrarlos en `main.go`.

## Ejemplo

```bash
# login como admin
TOKEN=$(curl -s localhost:8080/api/v1/auth/login -d '{"email":"admin@ecommerce.local","password":"Admin12345!"}' | jq -r .token)
H="Authorization: Bearer $TOKEN"

P=$(curl -s -H "$H" localhost:8080/api/v1/products -d '{"sku":"MS-1","name":"Mouse","price_cents":2000,"initial_stock":10}' | jq -r .id)
K=$(curl -s -H "$H" localhost:8080/api/v1/products -d '{"sku":"KB-1","name":"Teclado","price_cents":5000,"initial_stock":5}' | jq -r .id)
B=$(curl -s -H "$H" localhost:8080/api/v1/bundles -d "{\"sku\":\"KIT-1\",\"name\":\"Kit\",\"discount_pct\":10,\"items\":[{\"product_id\":\"$P\",\"quantity\":1},{\"product_id\":\"$K\",\"quantity\":1}]}" | jq -r .id)

# cliente
curl -s localhost:8080/api/v1/auth/register -d '{"email":"ana@mail.com","name":"Ana","password":"clave-segura-1"}'
CT=$(curl -s localhost:8080/api/v1/auth/login -d '{"email":"ana@mail.com","password":"clave-segura-1"}' | jq -r .token)

curl -s -H "Authorization: Bearer $CT" -H "Idempotency-Key: pedido-001" localhost:8080/api/v1/orders -d "{
  \"gateway\":\"stripe\",\"payment_token\":\"tok_ok\",\"coupon\":\"BIENVENIDO10\",
  \"shipping\":{\"name\":\"Ana\",\"address\":\"Calle 1 #2-3\",\"city\":\"Bogota\",\"country\":\"CO\"},
  \"items\":[{\"product_id\":\"$B\",\"quantity\":2}]}" | jq

curl -s -H "$H" "localhost:8080/api/v1/reports/sales" | jq
```

## Producción (checklist)

- `APP_ENV=production`, `JWT_SECRET` aleatorio (`openssl rand -hex 32`), `ADMIN_EMAIL`/`ADMIN_PASSWORD` fuertes, `ALLOWED_ORIGINS` con tu dominio, `STRIPE_SECRET_KEY`.
- Poner la API detrás de HTTPS (proxy/balanceador) y activar `TRUST_PROXY=true` solo si ese proxy es de confianza.
- Credenciales de Postgres propias y respaldos del volumen `pgdata`.

## Alcance y límites conocidos

- El rate limit es en memoria (por instancia); con varias réplicas usar uno compartido (Redis/gateway).
- No hay webhooks de pasarela: el cobro con Stripe es síncrono (`confirm=true`); pagos que requieren 3-D Secure quedan como rechazados.
- Reembolsos parciales, devoluciones por ítem y envío por transportadora no están implementados.

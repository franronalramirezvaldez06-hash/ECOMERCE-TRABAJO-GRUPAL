# Arquitectura (guía rápida)

## Tipo de arquitectura

- **Monolito modular** en Go: un solo binario (`cmd/api`) con módulos aislados en `internal/`.
- **Arquitectura por capas** dentro de cada módulo: `httpapi` (transporte) → servicio de negocio → almacenamiento (pgx/PostgreSQL).
- **Composition root**: `cmd/api/main.go` es el único lugar donde se crean y conectan las dependencias (inyección manual, sin framework).
- **Frontend** separado: Next.js (App Router) que consume la API REST (`/api/v1`).

```
Frontend (Next.js) ──HTTP/JSON──▶ httpapi ──▶ sales ──▶ catalog (Composite)
                                     │          ├────▶ pricing (Decorator)
                                     │          ├────▶ payment (Decorator + Registry)
                                     │          └────▶ inventory / coupon
                                     └──▶ auth, catalog, inventory, coupon
                                                   └──▶ platform/database (PostgreSQL)
```

## Patrones de diseño y para qué sirven

| Patrón | Dónde | Para qué |
|---|---|---|
| **Composite** | `catalog/composite.go` | Un producto simple (`Simple`) y un combo (`Bundle`) se tratan igual. El combo puede contener otros combos; precio, disponibilidad y requerimientos de inventario se calculan recursivamente. |
| **Decorator** (precios) | `pricing/pricing.go` | `Chain(Base, WithShipping, WithCoupons)`: cada decorador agrega su renglón (envío, cupón, impuesto) sin que los demás lo sepan. |
| **Decorator** (pasarelas) | `payment/decorators.go` | `Decorate(pasarela, WithLogging, WithRetry, WithTimeout, WithAmountLimit)`: comportamiento transversal reutilizable sobre cualquier `Gateway`. |
| **Strategy / Adapter** | `payment/stripe.go`, `simulated.go` | Cada pasarela implementa la interfaz `Gateway`; el checkout no sabe cuál usa. |
| **Registry** | `payment.Registry` | Busca la pasarela por nombre elegida por el cliente. |
| **Dependency Inversion** | `sales.Catalog`, `pricing.CouponLookup`, `catalog.StockReader` | El módulo consumidor define la interfaz que necesita; el otro módulo la implementa. |
| **Unit of Work / Saga simple** | `sales/service.go` (Checkout) | Transacción para reservar stock y cupón; el cobro va fuera; si falla se compensa (devuelve stock y cupón). |
| **Idempotency Key** | `sales` + header `Idempotency-Key` | Reintentos seguros sin doble cobro. |
| **Middleware chain** | `httpapi/middleware.go` | request-id, seguridad, recover, logs, CORS, rate limit, autenticación. |

## Módulos del backend

| Módulo | Responsabilidad |
|---|---|
| `auth` | Usuarios, bcrypt, JWT, roles admin/customer |
| `catalog` | Productos, combos y categorías |
| `pricing` | Cálculo del total (subtotal, cupón, envío, impuesto) |
| `payment` | Pasarelas y decoradores |
| `coupon` | Cupones en BD |
| `inventory` | Existencias, movimientos, reserva atómica |
| `sales` | Orquestación del checkout y ciclo de vida de la orden |
| `httpapi` | Rutas REST, middlewares y errores HTTP |
| `platform/database`, `config`, `apperr` | Infraestructura compartida |

## Mapa de archivos (tras ordenar)

`sales/`
- `service.go` – `Service`, cotización y **Checkout** (crear orden, cobrar, compensar)
- `refund.go` – reembolso / cancelación
- `fulfillment.go` – aprobar, enviar, entregar
- `model.go` – tipos y estados · `store.go` – lectura de órdenes y reportes

`httpapi/`
- `api.go` – rutas y cadena de middlewares
- `handlers_auth.go`, `handlers_catalog.go`, `handlers_inventory.go`, `handlers_coupons.go`, `handlers_orders.go`, `handlers_reports.go` – un archivo por dominio
- `helpers.go`, `middleware.go`

## Ciclo de vida de una orden

`pending → paid → approved → shipped → delivered`
`pending → failed` (pago rechazado) · `paid/approved → refunded | cancelled`

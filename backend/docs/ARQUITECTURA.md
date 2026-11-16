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
| **Repository** | `auth/repository.go`, `catalog/repository.go`, `*/store*.go` | Aísla el acceso a datos: cambiar de base de datos solo toca esta capa. |
| **Dependency Injection** | `cmd/api/main.go`, `sales.Deps` | Las dependencias se pasan desde fuera; facilita las pruebas. |
| **Singleton** (con cuidado) | pool de conexiones `pgxpool` | Una sola instancia creada en `main` e inyectada, sin variables globales. |
| **Factory** | `newGateways` + `payment.Registry` | Elige la pasarela por nombre sin `if-else` gigantes. |
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

## Capas (regla: cada capa hace una sola cosa)

| Capa | Archivos | Qué hace | Qué NO hace |
|---|---|---|---|
| **Presentación** (handlers) | `httpapi/handlers_<dominio>.go` | Recibe HTTP, decodifica/valida el formato, llama al servicio y responde con `a.ok` / `a.created` | Lógica de negocio ni SQL |
| **Negocio** (servicios) | `<módulo>/service.go` (+ `refund.go`, `fulfillment.go`, `listing.go`) | Reglas: precios, permisos de estado, idempotencia, compensaciones | SQL ni HTTP |
| **Datos** (repositorios) | `<módulo>/repository.go` o `store*.go` | Único lugar con SQL y transacciones | Decisiones de negocio |
| **Modelos / DTOs** | `<módulo>/model.go` | Entidades y datos de entrada/salida (`User` nunca expone el hash de la contraseña) | — |

`cmd/api/main.go` es la raíz de composición: crea cada pieza y la inyecta (`newAPI`, `newGateways`, `newServer`, `serve`).

## Reglas de código (Clean Code)

- **Funciones:** máximo 40 líneas; si crece, se extrae una función con nombre que diga qué hace.
- **Archivos:** máximo ~300 líneas; al crecer se dividen por responsabilidad.
- **Parámetros:** máximo 3 (sin contar `ctx context.Context`, que en Go siempre va primero). Si hacen falta más, se agrupan en un struct: `sales.Deps`, `sales.ListFilter`, `inventory.StockMove`, `catalog.Info`, `paymentRecord`.
- **Errores:** nunca un 500 genérico por un error esperado. Los módulos devuelven errores de `apperr` y `httpapi/helpers.go` los traduce a 400/401/403/404/409/502.
- **DRY:** lo repetido se extrae (`Store.inTx`, `insertPayment`, `releaseStock`, `releaseCoupon`).
- **Nombres:** verbos para acciones (`createPending`, `recordPayment`), sustantivos para datos (`pendingOrder`, `approvedPayment`).

Para auditar: funciones largas, archivos largos y exceso de parámetros se revisan antes de cada PR.

## Mapa de archivos

`sales/`
- `service.go` – `Service`, cotización y **Checkout** (orquesta: idempotencia → precio → orden → cobro)
- `refund.go` – reembolso / cancelación · `fulfillment.go` – aprobar, enviar, entregar
- `model.go` – tipos y estados
- `store.go` – lectura de órdenes · `store_checkout.go` – transacciones de la venta · `store_refund.go` – transacción del reembolso · `store_reports.go` – reportes

`auth/` – `service.go` (negocio), `repository.go` (SQL de usuarios), `model.go`, `token.go` (JWT)

`catalog/` – `composite.go` (patrón), `service.go`, `listing.go`, `repository.go`

`coupon/` – `model.go`, `store.go` · `inventory/` – `store.go`

`httpapi/`
- `api.go` – rutas agrupadas por dominio (`authRoutes`, `catalogRoutes`, ...) y cadena de middlewares
- `handlers_<dominio>.go` – un archivo por dominio
- `helpers.go`, `middleware.go`

## Ciclo de vida de una orden

`pending → paid → approved → shipped → delivered`
`pending → failed` (pago rechazado) · `paid/approved → refunded | cancelled`

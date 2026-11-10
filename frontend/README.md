# Ecommerce — Frontend (Grupo 3)

Tienda y panel de administración en **Next.js 15 (App Router) + React 19 + TypeScript + Tailwind CSS**,
conectados al backend en Go de [`../backend`](../backend).

## Arranque

```bash
cp .env.example .env.local     # NEXT_PUBLIC_API_URL=http://localhost:8080
npm install
npm run dev                    # http://localhost:3000
```

Primero levanta el backend (`cd ../backend && docker compose up -d --build`). En desarrollo el backend crea el admin
`admin@ecommerce.local` / `Admin12345!`. El backend permite CORS desde cualquier origen solo en desarrollo; en producción
define `ALLOWED_ORIGINS` con el dominio del frontend.

```bash
npm run typecheck   # TypeScript estricto
npm test            # pruebas unitarias (carrito, formato de dinero)
npm run build       # build de producción
```

## Qué incluye

**Tienda (público / cliente)**
- Inicio con novedades y categorías; catálogo con búsqueda (debounce), filtro por categoría y paginación en la URL.
- Detalle de producto; los **combos muestran su árbol de componentes** y el ahorro (Composite del backend).
- Carrito persistente (localStorage, sincronizado entre pestañas) con **cotización en vivo** del backend: descuento por cupón y envío, sin impuesto adicional.
- Checkout con dirección, pasarela y medio de pago. Usa `Idempotency-Key`: un reintento por falla de red no duplica la compra; un pago rechazado (402) se informa sin perder el carrito.
- Registro / login, pedidos propios, detalle y **cancelación con reembolso**.

**Panel de administración (`/admin`, solo rol admin)**
- Resumen: ingresos, reembolsos, ticket promedio, ingresos por día, más vendidos, pedidos por estado, poco stock.
- Productos y combos: crear, editar, activar/desactivar; el formulario de combos calcula el precio con descuento.
- Inventario: alertas de poco stock, ajustes con motivo e historial de movimientos.
- Pedidos: filtro por estado, marcar enviado/entregado y reembolsar.
- Cupones (vigencia, usos máximos) y categorías.

## Arquitectura

```
src/
  app/                  rutas (App Router): (shop), (auth) y admin con su propio layout
  components/
    ui/                 Button, formularios accesibles, Modal, Badge, Pagination...
    shop/ admin/        componentes de cada área
    auth/guards.tsx     RequireAuth / RequireAdmin
  providers/            sesión (JWT), carrito, notificaciones, React Query
  lib/
    api/                cliente fetch tipado (ApiError, 401 -> cierra sesión), tipos y servicios
    cart/               lógica pura del carrito (reducer) con pruebas
    format.ts           dinero en centavos, fechas, estados
```

Decisiones:
- **Dinero en centavos** de punta a punta; `parseMoneyToCents` evita errores de punto flotante.
- **El backend es la fuente de verdad** de precios y stock; el carrito solo guarda una referencia visual.
- **React Query** para datos del servidor (caché, reintentos solo en errores no-4xx, invalidación tras mutaciones).
- El JWT se guarda en `localStorage` (simple y válido para este alcance). Para endurecerlo en producción, mover el token a una cookie `HttpOnly` mediante un proxy/route handler de Next.
- Los guards del frontend son solo UX: **la autorización real la hace el backend** (401/403).
- `?next=` solo acepta rutas internas (evita redirecciones abiertas).

## Docker

```bash
docker build -t ecommerce-web --build-arg NEXT_PUBLIC_API_URL=https://api.tudominio.com .
docker run -p 3000:3000 ecommerce-web
```

`NEXT_PUBLIC_API_URL` se incrusta en el build, así que debe darse al construir la imagen.

## Límites conocidos

- Con Stripe real hace falta Stripe.js para crear el `PaymentMethod`; hoy el formulario recibe el token como texto (lo normal con las pasarelas simuladas).
- Sin imágenes subidas: el admin pega una URL.
- No hay recuperación de contraseña ni edición de perfil (el backend tampoco los tiene).

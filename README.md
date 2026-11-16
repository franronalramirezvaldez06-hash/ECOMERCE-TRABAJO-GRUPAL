# Ecommerce — Trabajo grupal (Grupo 3)

Sistema ecommerce de **ventas, inventarios y pasarelas de pago**.

| Carpeta | Contenido | Documentación |
|---|---|---|
| [`backend/`](backend) | API REST en Go (monolito modular) + PostgreSQL | [README](backend/README.md) · [Arquitectura](backend/docs/ARQUITECTURA.md) · [OpenAPI](backend/docs/openapi.yaml) |
| [`frontend/`](frontend) | Tienda y panel de administración en Next.js 15 + TypeScript | [README](frontend/README.md) |
| [`deploy/`](deploy) | Scripts de despliegue en AWS EC2 | [README](deploy/README.md) |
| [`.github/workflows/`](.github/workflows) | Integración continua (formato, vet y pruebas del backend) | — |

## Arranque rápido

Backend (API en <http://localhost:8080>):

```bash
cd backend && cp .env.example .env && docker compose up -d --build
```

Frontend (tienda en <http://localhost:3000>):

```bash
cd frontend && cp .env.example .env.local && npm install && npm run dev
```

## Estructura

```
backend/
  cmd/api/          punto de entrada y composición de dependencias
  internal/         módulos de negocio (auth, catalog, pricing, payment, coupon, inventory, sales, httpapi)
  migrations/       esquema SQL versionado
  docs/             arquitectura y contrato OpenAPI
frontend/
  src/app/          rutas: (shop), (auth) y admin
  src/components/   ui/ (un componente por archivo), shop/, admin/, auth/
  src/providers/    sesión, carrito, notificaciones
  src/lib/          cliente de la API, lógica del carrito, formato
deploy/             scripts de despliegue
```

## Convenciones

- **Capas:** handler (HTTP) → servicio (negocio) → repositorio/store (SQL). Un handler no tiene lógica de negocio ni SQL; un servicio no tiene SQL.
- **Tamaños:** funciones ≤ 40 líneas, archivos ≤ ~300 líneas, máximo 3 parámetros (se agrupan en un struct/objeto). Detalle en [Arquitectura](backend/docs/ARQUITECTURA.md).
- **Backend:** un paquete por dominio en `internal/`; dentro, `service.go` (negocio), `store*.go`/`repository.go` (SQL), `model.go` (tipos); pruebas `*_test.go` junto al código. `httpapi` tiene un `handlers_<dominio>.go` por dominio.
- **Frontend:** archivos en `kebab-case`; un componente de UI por archivo en `components/ui/`; la lógica de una pantalla va en un hook `use-*.ts` y el marcado largo en subcomponentes; componentes de área en `components/shop|admin|auth`; pruebas `*.test.ts` junto al código.
- **Dinero** siempre en centavos; el backend es la fuente de verdad de precios y stock.

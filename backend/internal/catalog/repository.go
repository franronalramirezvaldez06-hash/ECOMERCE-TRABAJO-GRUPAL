package catalog

import (
	"context"
	"time"

	"ecommerce/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Record es la fila de la tabla products.
type Record struct {
	ID                string    `json:"id"`
	SKU               string    `json:"sku"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	PriceCents        int64     `json:"price_cents"`
	Kind              string    `json:"kind"`
	BundleDiscountPct int       `json:"bundle_discount_pct"`
	Active            bool      `json:"active"`
	CategoryID        *string   `json:"category_id"`
	ImageURL          string    `json:"image_url"`
	CreatedAt         time.Time `json:"created_at"`
}

// BundleItem es una fila de bundle_items.
type BundleItem struct {
	ComponentID string `json:"product_id"`
	Quantity    int    `json:"quantity"`
}

type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type ListFilter struct {
	Query           string
	CategoryID      string
	Kind            string
	Sort            string
	InStock         bool
	IncludeInactive bool
	Limit, Offset   int
}

type UpdateInput struct {
	Name              *string `json:"name"`
	Description       *string `json:"description"`
	PriceCents        *int64  `json:"price_cents"`
	CategoryID        *string `json:"category_id"` // "" quita la categoria
	ImageURL          *string `json:"image_url"`
	Active            *bool   `json:"active"`
	BundleDiscountPct *int    `json:"bundle_discount_pct"`
}

const columns = `id::text, sku, name, description, price_cents, kind, bundle_discount_pct, active, category_id::text, image_url, created_at`

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func scan(row pgx.Row) (Record, error) {
	var r Record
	err := row.Scan(&r.ID, &r.SKU, &r.Name, &r.Description, &r.PriceCents, &r.Kind, &r.BundleDiscountPct,
		&r.Active, &r.CategoryID, &r.ImageURL, &r.CreatedAt)
	return r, database.MapError(err)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// CreateSimple inserta un producto y su fila de inventario en una transaccion.
func (r *Repository) CreateSimple(ctx context.Context, rec Record, initialStock int) (Record, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Record{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	out, err := scan(tx.QueryRow(ctx,
		`INSERT INTO products (sku, name, description, price_cents, kind, category_id, image_url)
		 VALUES ($1, $2, $3, $4, 'simple', NULLIF($5::text, '')::uuid, $6) RETURNING `+columns,
		rec.SKU, rec.Name, rec.Description, rec.PriceCents, deref(rec.CategoryID), rec.ImageURL))
	if err != nil {
		return Record{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO inventory (product_id, quantity) VALUES ($1::uuid, $2)`, out.ID, initialStock); err != nil {
		return Record{}, database.MapError(err)
	}
	if initialStock > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO inventory_movements (product_id, delta, reason) VALUES ($1::uuid, $2, 'stock_inicial')`,
			out.ID, initialStock); err != nil {
			return Record{}, database.MapError(err)
		}
	}
	return out, database.MapError(tx.Commit(ctx))
}

// CreateBundle inserta un combo y sus componentes en una transaccion.
func (r *Repository) CreateBundle(ctx context.Context, rec Record, items []BundleItem) (Record, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Record{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	out, err := scan(tx.QueryRow(ctx,
		`INSERT INTO products (sku, name, description, price_cents, kind, bundle_discount_pct, category_id, image_url)
		 VALUES ($1, $2, $3, 0, 'bundle', $4, NULLIF($5::text, '')::uuid, $6) RETURNING `+columns,
		rec.SKU, rec.Name, rec.Description, rec.BundleDiscountPct, deref(rec.CategoryID), rec.ImageURL))
	if err != nil {
		return Record{}, err
	}
	for _, it := range items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO bundle_items (bundle_id, component_id, quantity) VALUES ($1::uuid, $2::uuid, $3)`,
			out.ID, it.ComponentID, it.Quantity); err != nil {
			return Record{}, database.MapError(err)
		}
	}
	return out, database.MapError(tx.Commit(ctx))
}

func (r *Repository) Update(ctx context.Context, id string, in UpdateInput) (Record, error) {
	return scan(r.pool.QueryRow(ctx,
		`UPDATE products SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			price_cents = COALESCE($4, price_cents),
			image_url = COALESCE($5, image_url),
			active = COALESCE($6, active),
			bundle_discount_pct = COALESCE($7, bundle_discount_pct),
			category_id = CASE WHEN $8::text IS NULL THEN category_id ELSE NULLIF($8::text, '')::uuid END
		 WHERE id = $1::uuid RETURNING `+columns,
		id, in.Name, in.Description, in.PriceCents, in.ImageURL, in.Active, in.BundleDiscountPct, in.CategoryID))
}

func (r *Repository) Get(ctx context.Context, id string) (Record, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+columns+` FROM products WHERE id = $1::uuid`, id))
}

func (r *Repository) List(ctx context.Context, f ListFilter) ([]Record, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+columns+` FROM products
		 WHERE ($1::bool OR active)
		   AND ($2 = '' OR category_id::text = $2)
		   AND ($3 = '' OR name ILIKE '%' || $3 || '%' OR sku ILIKE '%' || $3 || '%')
		   AND ($6 = '' OR kind = $6)
		 ORDER BY CASE WHEN $7 = 'name' THEN lower(name) END ASC, created_at DESC, id DESC LIMIT $4 OFFSET $5`,
		f.IncludeInactive, f.CategoryID, f.Query, f.Limit, f.Offset, f.Kind, f.Sort)
	if err != nil {
		return nil, database.MapError(err)
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		rec, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repository) BundleItems(ctx context.Context, bundleID string) ([]BundleItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT component_id::text, quantity FROM bundle_items WHERE bundle_id = $1::uuid ORDER BY component_id`, bundleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BundleItem
	for rows.Next() {
		var it BundleItem
		if err := rows.Scan(&it.ComponentID, &it.Quantity); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCategory(ctx context.Context, name string) (Category, error) {
	var c Category
	err := r.pool.QueryRow(ctx,
		`INSERT INTO categories (name) VALUES ($1) RETURNING id::text, name, created_at`, name).
		Scan(&c.ID, &c.Name, &c.CreatedAt)
	return c, database.MapError(err)
}

func (r *Repository) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text, name, created_at FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

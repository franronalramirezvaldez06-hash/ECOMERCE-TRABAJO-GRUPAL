package catalog

import (
	"context"
	"strings"

	"ecommerce/internal/apperr"
)

const maxBundleDepth = 8

// StockReader abstrae el modulo de inventario.
type StockReader interface {
	Levels(ctx context.Context, productIDs []string) (map[string]int, error)
}

type Service struct {
	repo  *Repository
	stock StockReader
}

func NewService(repo *Repository, stock StockReader) *Service {
	return &Service{repo: repo, stock: stock}
}

type CreateSimpleInput struct {
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	PriceCents   int64  `json:"price_cents"`
	InitialStock int    `json:"initial_stock"`
	CategoryID   string `json:"category_id"`
	ImageURL     string `json:"image_url"`
}

type CreateBundleInput struct {
	SKU         string       `json:"sku"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	DiscountPct int          `json:"discount_pct"`
	CategoryID  string       `json:"category_id"`
	ImageURL    string       `json:"image_url"`
	Items       []BundleItem `json:"items"`
}

// Detail es la vista completa de un producto o combo.
type Detail struct {
	Record
	Price     int64 `json:"price_cents_total"`
	Available int   `json:"available"`
	Tree      Node  `json:"tree"`
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *Service) CreateSimple(ctx context.Context, in CreateSimpleInput) (Record, error) {
	in.SKU, in.Name = strings.TrimSpace(in.SKU), strings.TrimSpace(in.Name)
	switch {
	case in.SKU == "":
		return Record{}, apperr.Validation("sku es obligatorio")
	case in.Name == "":
		return Record{}, apperr.Validation("name es obligatorio")
	case in.PriceCents < 0:
		return Record{}, apperr.Validation("price_cents no puede ser negativo")
	case in.InitialStock < 0:
		return Record{}, apperr.Validation("initial_stock no puede ser negativo")
	}
	return s.repo.CreateSimple(ctx, Record{
		SKU: in.SKU, Name: in.Name, Description: in.Description, PriceCents: in.PriceCents,
		CategoryID: optional(in.CategoryID), ImageURL: in.ImageURL,
	}, in.InitialStock)
}

func (s *Service) CreateBundle(ctx context.Context, in CreateBundleInput) (Record, error) {
	in.SKU, in.Name = strings.TrimSpace(in.SKU), strings.TrimSpace(in.Name)
	switch {
	case in.SKU == "":
		return Record{}, apperr.Validation("sku es obligatorio")
	case in.Name == "":
		return Record{}, apperr.Validation("name es obligatorio")
	case in.DiscountPct < 0 || in.DiscountPct > 100:
		return Record{}, apperr.Validation("discount_pct debe estar entre 0 y 100")
	case len(in.Items) == 0:
		return Record{}, apperr.Validation("un combo necesita al menos un componente")
	}
	seen := map[string]bool{}
	for _, it := range in.Items {
		if it.Quantity <= 0 {
			return Record{}, apperr.Validation("la cantidad de cada componente debe ser mayor a cero")
		}
		if seen[it.ComponentID] {
			return Record{}, apperr.Validation("componente repetido: %s", it.ComponentID)
		}
		seen[it.ComponentID] = true
	}
	// Los combos solo referencian productos ya existentes y sus componentes no
	// se editan, por lo que el grafo no puede tener ciclos.
	return s.repo.CreateBundle(ctx, Record{
		SKU: in.SKU, Name: in.Name, Description: in.Description, BundleDiscountPct: in.DiscountPct,
		CategoryID: optional(in.CategoryID), ImageURL: in.ImageURL,
	}, in.Items)
}

// Update modifica datos de un producto o combo (el precio solo aplica a
// productos simples y el descuento solo a combos).
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (Record, error) {
	rec, err := s.repo.Get(ctx, id)
	if err != nil {
		return Record{}, err
	}
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return Record{}, apperr.Validation("name no puede estar vacio")
		}
		in.Name = &n
	}
	if in.PriceCents != nil {
		if rec.Kind == KindBundle {
			return Record{}, apperr.Validation("el precio de un combo se calcula de sus componentes")
		}
		if *in.PriceCents < 0 {
			return Record{}, apperr.Validation("price_cents no puede ser negativo")
		}
	}
	if in.BundleDiscountPct != nil {
		if rec.Kind != KindBundle {
			return Record{}, apperr.Validation("bundle_discount_pct solo aplica a combos")
		}
		if *in.BundleDiscountPct < 0 || *in.BundleDiscountPct > 100 {
			return Record{}, apperr.Validation("bundle_discount_pct debe estar entre 0 y 100")
		}
	}
	return s.repo.Update(ctx, id, in)
}

// ListItem es un producto del listado con su precio final y disponibilidad
// (para los combos el precio se calcula a partir de sus componentes).
type ListItem struct {
	Record
	Price     int64 `json:"price_cents_total"`
	Available int   `json:"available"`
}

// List devuelve la pagina pedida con precio y disponibilidad ya calculados.
// Hace una sola consulta de inventario para toda la pagina.
func (s *Service) List(ctx context.Context, f ListFilter) ([]ListItem, error) {
	recs, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}
	comps := make([]Component, len(recs))
	leafSet := map[string]struct{}{}
	for i, rec := range recs {
		comp, err := s.build(ctx, rec.ID, 0, true)
		if err != nil {
			return nil, err
		}
		comps[i] = comp
		for leaf := range comp.Requirements(1) {
			leafSet[leaf] = struct{}{}
		}
	}
	ids := make([]string, 0, len(leafSet))
	for id := range leafSet {
		ids = append(ids, id)
	}
	levels, err := s.stock.Levels(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, len(recs))
	for i, rec := range recs {
		out[i] = ListItem{Record: rec, Price: comps[i].Price(), Available: comps[i].Available(StockView(levels))}
	}
	return out, nil
}

func (s *Service) CreateCategory(ctx context.Context, name string) (Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Category{}, apperr.Validation("name es obligatorio")
	}
	return s.repo.CreateCategory(ctx, name)
}

func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	return s.repo.ListCategories(ctx)
}

// Build arma el arbol Composite de un producto leyendo la base de datos.
// Rechaza productos inactivos (se usa al vender).
func (s *Service) Build(ctx context.Context, id string) (Component, error) {
	return s.build(ctx, id, 0, false)
}

func (s *Service) build(ctx context.Context, id string, depth int, allowInactive bool) (Component, error) {
	if depth > maxBundleDepth {
		return nil, apperr.Validation("combo demasiado anidado (maximo %d niveles)", maxBundleDepth)
	}
	rec, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !rec.Active && !allowInactive {
		return nil, apperr.Validation("el producto %q no esta disponible para la venta", rec.Name)
	}
	if rec.Kind == KindSimple {
		return NewSimple(rec.ID, rec.SKU, rec.Name, rec.PriceCents), nil
	}
	bundle, err := NewBundle(rec.ID, rec.SKU, rec.Name, rec.BundleDiscountPct)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.BundleItems(ctx, rec.ID)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		child, err := s.build(ctx, it.ComponentID, depth+1, allowInactive)
		if err != nil {
			return nil, err
		}
		if err := bundle.Add(child, it.Quantity); err != nil {
			return nil, err
		}
	}
	return bundle, nil
}

// Detail devuelve el producto con precio, disponibilidad y arbol. Los
// productos inactivos solo los ve quien tenga includeInactive (administradores).
func (s *Service) Detail(ctx context.Context, id string, includeInactive bool) (*Detail, error) {
	rec, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !rec.Active && !includeInactive {
		return nil, apperr.ErrNotFound
	}
	comp, err := s.build(ctx, id, 0, true)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0)
	for leaf := range comp.Requirements(1) {
		ids = append(ids, leaf)
	}
	levels, err := s.stock.Levels(ctx, ids)
	if err != nil {
		return nil, err
	}
	return &Detail{
		Record:    rec,
		Price:     comp.Price(),
		Available: comp.Available(StockView(levels)),
		Tree:      Describe(comp, 1),
	}, nil
}

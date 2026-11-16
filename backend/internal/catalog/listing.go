package catalog

// Listado del catalogo con precio y disponibilidad calculados.

import "context"

// ListItem es un producto del listado con su precio final y disponibilidad
// (para los combos el precio se calcula a partir de sus componentes).
type ListItem struct {
	Record
	Price     int64 `json:"price_cents_total"`
	Available int   `json:"available"`
}

// List devuelve la pagina pedida con precio y disponibilidad ya calculados.
func (s *Service) List(ctx context.Context, f ListFilter) ([]ListItem, error) {
	if f.InStock {
		return s.listInStock(ctx, f)
	}
	recs, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}
	comps := make([]Component, len(recs))
	for i, rec := range recs {
		if comps[i], err = s.build(ctx, rec.ID, 0, true); err != nil {
			return nil, err
		}
	}
	// Una sola consulta de inventario para toda la pagina.
	levels, err := s.stock.Levels(ctx, leafIDs(comps))
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, len(recs))
	for i, rec := range recs {
		out[i] = ListItem{Record: rec, Price: comps[i].Price(), Available: comps[i].Available(StockView(levels))}
	}
	return out, nil
}

// listInStock aplica el filtro de disponibilidad antes de paginar, incluso
// para combos anidados: recorre el catalogo por lotes hasta llenar la pagina.
func (s *Service) listInStock(ctx context.Context, f ListFilter) ([]ListItem, error) {
	wanted, skip := f.Limit, f.Offset
	f.InStock, f.Limit, f.Offset = false, 100, 0
	out := make([]ListItem, 0, wanted)
	for len(out) < wanted {
		batch, err := s.List(ctx, f)
		if err != nil {
			return nil, err
		}
		for _, item := range batch {
			if item.Available <= 0 {
				continue
			}
			if skip > 0 {
				skip--
				continue
			}
			if out = append(out, item); len(out) == wanted {
				break
			}
		}
		if len(batch) < f.Limit {
			break
		}
		f.Offset += f.Limit
	}
	return out, nil
}

// leafIDs devuelve, sin repetir, los productos simples que componen comps.
func leafIDs(comps []Component) []string {
	seen := map[string]struct{}{}
	ids := []string{}
	for _, comp := range comps {
		for leaf := range comp.Requirements(1) {
			if _, ok := seen[leaf]; !ok {
				seen[leaf] = struct{}{}
				ids = append(ids, leaf)
			}
		}
	}
	return ids
}

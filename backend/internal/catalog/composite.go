package catalog

import (
	"fmt"
	"math"
)

// PATRON COMPOSITE
//
// Component es la interfaz comun: un producto simple (hoja) y un combo
// (compuesto) se tratan igual. Un combo puede contener productos simples u
// otros combos, y el calculo de precio, disponibilidad y requerimientos de
// inventario se resuelve recursivamente sin que el cliente distinga el tipo.

const (
	KindSimple = "simple"
	KindBundle = "bundle"
)

// StockView es una foto del inventario: id de producto simple -> unidades.
type StockView map[string]int

// Component es la interfaz del patron Composite.
type Component interface {
	ID() string
	SKU() string
	Name() string
	Kind() string
	// Price devuelve el precio unitario en centavos.
	Price() int64
	// Available devuelve cuantas unidades completas se pueden vender.
	Available(stock StockView) int
	// Requirements devuelve cuantas unidades de cada producto simple
	// (id -> cantidad) hacen falta para entregar qty unidades.
	Requirements(qty int) map[string]int
	// Children devuelve los hijos directos (vacio en una hoja).
	Children() []Entry
}

// Entry es un hijo de un combo con su cantidad.
type Entry struct {
	Component Component
	Quantity  int
}

// ---------------------------------------------------------------- Hoja

// Simple es la hoja del arbol: un producto individual.
type Simple struct {
	id, sku, name string
	priceCents    int64
}

func NewSimple(id, sku, name string, priceCents int64) *Simple {
	return &Simple{id: id, sku: sku, name: name, priceCents: priceCents}
}

func (p *Simple) ID() string                              { return p.id }
func (p *Simple) SKU() string                             { return p.sku }
func (p *Simple) Name() string                            { return p.name }
func (p *Simple) Kind() string                            { return KindSimple }
func (p *Simple) Price() int64                            { return p.priceCents }
func (p *Simple) Available(stock StockView) int           { return stock[p.id] }
func (p *Simple) Requirements(qty int) map[string]int     { return map[string]int{p.id: qty} }
func (p *Simple) Children() []Entry                       { return nil }

// ----------------------------------------------------------- Compuesto

// Bundle es el compuesto: agrupa componentes y aplica un descuento propio.
type Bundle struct {
	id, sku, name string
	discountPct   int
	entries       []Entry
}

func NewBundle(id, sku, name string, discountPct int) (*Bundle, error) {
	if discountPct < 0 || discountPct > 100 {
		return nil, fmt.Errorf("descuento de combo invalido: %d", discountPct)
	}
	return &Bundle{id: id, sku: sku, name: name, discountPct: discountPct}, nil
}

// Add agrega un componente (simple o combo) con su cantidad.
func (b *Bundle) Add(c Component, qty int) error {
	if c == nil {
		return fmt.Errorf("componente nulo")
	}
	if qty <= 0 {
		return fmt.Errorf("la cantidad de %q debe ser mayor a cero", c.Name())
	}
	b.entries = append(b.entries, Entry{Component: c, Quantity: qty})
	return nil
}

func (b *Bundle) ID() string       { return b.id }
func (b *Bundle) SKU() string      { return b.sku }
func (b *Bundle) Name() string     { return b.name }
func (b *Bundle) Kind() string     { return KindBundle }
func (b *Bundle) Children() []Entry { return b.entries }

// Price suma los hijos (recursivo) y aplica el descuento del combo.
func (b *Bundle) Price() int64 {
	var sum int64
	for _, e := range b.entries {
		sum += e.Component.Price() * int64(e.Quantity)
	}
	return sum * int64(100-b.discountPct) / 100
}

// Available es el minimo de combos completos que permite cada hijo.
func (b *Bundle) Available(stock StockView) int {
	if len(b.entries) == 0 {
		return 0
	}
	min := math.MaxInt
	for _, e := range b.entries {
		if n := e.Component.Available(stock) / e.Quantity; n < min {
			min = n
		}
	}
	return min
}

// Requirements expande el combo (y sus sub-combos) a productos simples.
func (b *Bundle) Requirements(qty int) map[string]int {
	out := make(map[string]int)
	for _, e := range b.entries {
		for id, n := range e.Component.Requirements(qty * e.Quantity) {
			out[id] += n
		}
	}
	return out
}

// ------------------------------------------------------------ Descripcion

// Node es la representacion serializable del arbol para la API.
type Node struct {
	ID         string `json:"id"`
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Quantity   int    `json:"quantity"`
	PriceCents int64  `json:"price_cents"`
	Children   []Node `json:"children,omitempty"`
}

// Describe convierte un componente (y su subarbol) en un Node.
func Describe(c Component, qty int) Node {
	n := Node{ID: c.ID(), SKU: c.SKU(), Name: c.Name(), Kind: c.Kind(), Quantity: qty, PriceCents: c.Price()}
	for _, e := range c.Children() {
		n.Children = append(n.Children, Describe(e.Component, e.Quantity))
	}
	return n
}

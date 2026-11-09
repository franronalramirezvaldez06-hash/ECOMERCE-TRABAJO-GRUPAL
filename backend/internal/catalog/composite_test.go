package catalog

import "testing"

func buildTree(t *testing.T) (*Bundle, StockView) {
	t.Helper()
	mouse := NewSimple("mouse", "MS-1", "Mouse", 2000)
	teclado := NewSimple("teclado", "KB-1", "Teclado", 5000)
	pad := NewSimple("pad", "PD-1", "Pad", 1000)

	kit, _ := NewBundle("kit", "KIT-1", "Kit escritorio", 10)
	if err := kit.Add(mouse, 1); err != nil {
		t.Fatal(err)
	}
	if err := kit.Add(teclado, 1); err != nil {
		t.Fatal(err)
	}

	// Combo dentro de combo.
	pro, _ := NewBundle("pro", "PRO-1", "Pack pro", 0)
	_ = pro.Add(kit, 2)
	_ = pro.Add(pad, 1)

	return pro, StockView{"mouse": 10, "teclado": 3, "pad": 100}
}

func TestSimpleLeaf(t *testing.T) {
	p := NewSimple("a", "A", "A", 1500)
	if p.Price() != 1500 || p.Available(StockView{"a": 7}) != 7 {
		t.Fatal("hoja devuelve valores incorrectos")
	}
}

func TestBundlePriceAppliesDiscountRecursively(t *testing.T) {
	pro, _ := buildTree(t)
	// kit = (2000+5000)*0.90 = 6300 ; pro = 2*6300 + 1000 = 13600
	if got := pro.Price(); got != 13600 {
		t.Fatalf("precio esperado 13600, obtuvo %d", got)
	}
}

func TestBundleAvailabilityIsLimitedByScarcestChild(t *testing.T) {
	pro, stock := buildTree(t)
	// teclado: 3 unidades, cada pro consume 2 -> 1 pack completo.
	if got := pro.Available(stock); got != 1 {
		t.Fatalf("disponibilidad esperada 1, obtuvo %d", got)
	}
}

func TestBundleRequirementsFlattenNestedBundles(t *testing.T) {
	pro, _ := buildTree(t)
	req := pro.Requirements(3)
	want := map[string]int{"mouse": 6, "teclado": 6, "pad": 3}
	for id, n := range want {
		if req[id] != n {
			t.Fatalf("%s: esperado %d, obtuvo %d", id, n, req[id])
		}
	}
}

func TestBundleValidation(t *testing.T) {
	if _, err := NewBundle("x", "X", "X", 150); err == nil {
		t.Fatal("debio rechazar descuento > 100")
	}
	b, _ := NewBundle("x", "X", "X", 0)
	if err := b.Add(NewSimple("a", "A", "A", 1), 0); err == nil {
		t.Fatal("debio rechazar cantidad 0")
	}
	if b.Available(StockView{}) != 0 {
		t.Fatal("combo vacio no debe tener disponibilidad")
	}
}

func TestDescribeBuildsTree(t *testing.T) {
	pro, _ := buildTree(t)
	n := Describe(pro, 1)
	if len(n.Children) != 2 || len(n.Children[0].Children) != 2 {
		t.Fatalf("arbol inesperado: %+v", n)
	}
}

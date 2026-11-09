-- Usuarios y roles, categorias, cupones en BD y ciclo de vida completo de ordenes.

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text        NOT NULL UNIQUE,
    password_hash text        NOT NULL,
    name          text        NOT NULL,
    role          text        NOT NULL DEFAULT 'customer' CHECK (role IN ('admin', 'customer')),
    active        boolean     NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE categories (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE products ADD COLUMN category_id uuid REFERENCES categories (id) ON DELETE SET NULL;
ALTER TABLE products ADD COLUMN image_url   text NOT NULL DEFAULT '';
CREATE INDEX idx_products_category ON products (category_id);
CREATE INDEX idx_products_active ON products (active);

CREATE TABLE coupons (
    code         text PRIMARY KEY CHECK (code = upper(code)),
    discount_pct int         NOT NULL CHECK (discount_pct BETWEEN 1 AND 100),
    max_uses     int         NOT NULL DEFAULT 0 CHECK (max_uses >= 0), -- 0 = ilimitado
    used_count   int         NOT NULL DEFAULT 0 CHECK (used_count >= 0),
    expires_at   timestamptz,
    active       boolean     NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now()
);
INSERT INTO coupons (code, discount_pct) VALUES ('BIENVENIDO10', 10), ('VERANO20', 20);

ALTER TABLE orders ADD COLUMN user_id         uuid REFERENCES users (id);
ALTER TABLE orders ADD COLUMN idempotency_key text;
ALTER TABLE orders ADD COLUMN ship_name       text NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN ship_address    text NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN ship_city       text NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN ship_country    text NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN ship_phone      text NOT NULL DEFAULT '';
CREATE INDEX idx_orders_user ON orders (user_id, created_at DESC);
CREATE UNIQUE INDEX idx_orders_idempotency ON orders (user_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'paid', 'failed', 'cancelled', 'refunded', 'shipped', 'delivered'));

-- Unidades de productos simples descontadas por cada orden (los combos ya
-- expandidos). Permite revertir inventario aunque el catalogo cambie.
CREATE TABLE order_stock_items (
    order_id   uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    product_id uuid NOT NULL REFERENCES products (id),
    quantity   int  NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (order_id, product_id)
);

-- Esquema inicial: catalogo (con combos), inventario, ventas y pagos.

CREATE TABLE products (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    sku                  text        NOT NULL UNIQUE,
    name                 text        NOT NULL,
    description          text        NOT NULL DEFAULT '',
    price_cents          bigint      NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    kind                 text        NOT NULL CHECK (kind IN ('simple', 'bundle')),
    bundle_discount_pct  int         NOT NULL DEFAULT 0 CHECK (bundle_discount_pct BETWEEN 0 AND 100),
    active               boolean     NOT NULL DEFAULT true,
    created_at           timestamptz NOT NULL DEFAULT now()
);

-- Composite: un combo (bundle) contiene productos simples u otros combos.
CREATE TABLE bundle_items (
    bundle_id    uuid NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    component_id uuid NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    quantity     int  NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (bundle_id, component_id),
    CHECK (bundle_id <> component_id)
);

CREATE TABLE inventory (
    product_id uuid PRIMARY KEY REFERENCES products (id) ON DELETE CASCADE,
    quantity   int         NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE inventory_movements (
    id         bigserial PRIMARY KEY,
    product_id uuid        NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    delta      int         NOT NULL,
    reason     text        NOT NULL,
    reference  text        NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_movements_product ON inventory_movements (product_id, created_at DESC);

CREATE TABLE orders (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_email  text        NOT NULL,
    status          text        NOT NULL CHECK (status IN ('pending', 'paid', 'failed', 'cancelled')),
    coupon          text        NOT NULL DEFAULT '',
    subtotal_cents  bigint      NOT NULL,
    discount_cents  bigint      NOT NULL DEFAULT 0,
    shipping_cents  bigint      NOT NULL DEFAULT 0,
    tax_cents       bigint      NOT NULL DEFAULT 0,
    total_cents     bigint      NOT NULL,
    currency        text        NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_orders_created ON orders (created_at DESC);
CREATE INDEX idx_orders_email ON orders (customer_email);

CREATE TABLE order_items (
    id               bigserial PRIMARY KEY,
    order_id         uuid   NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    product_id       uuid   NOT NULL REFERENCES products (id),
    name             text   NOT NULL,
    unit_price_cents bigint NOT NULL,
    quantity         int    NOT NULL CHECK (quantity > 0)
);
CREATE INDEX idx_order_items_order ON order_items (order_id);

CREATE TABLE payments (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      uuid        NOT NULL REFERENCES orders (id),
    gateway       text        NOT NULL,
    status        text        NOT NULL CHECK (status IN ('approved', 'declined', 'error', 'refunded')),
    amount_cents  bigint      NOT NULL,
    external_ref  text        NOT NULL DEFAULT '',
    error_message text        NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_payments_order ON payments (order_id);

ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'paid', 'approved', 'failed', 'cancelled', 'refunded', 'shipped', 'delivered'));

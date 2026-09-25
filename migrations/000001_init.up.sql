DO $$
BEGIN
    CREATE TYPE user_order_status AS ENUM ('created', 'rejected', 'accepted', 'preparing', 'completed');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$$;

CREATE TABLE IF NOT EXISTS companies(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    company_source_url VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS company_products(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    partner_product_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    price BIGINT NOT NULL CHECK ( price > 0),
    description VARCHAR(255),
    current_count INTEGER NOT NULL CHECK ( current_count >= 0 ) DEFAULT 0,
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (company_id, partner_product_id)
);

CREATE TABLE IF NOT EXISTS users(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid()
);

CREATE TABLE IF NOT EXISTS user_baskets(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    user_id UUID NOT NULL REFERENCES users(id),

    UNIQUE (company_id,user_id)
);
CREATE TABLE IF NOT EXISTS user_basket_products(
    basket_id UUID NOT NULL REFERENCES user_baskets(id),
    product_id UUID NOT NULL REFERENCES company_products(id),
    count INTEGER NOT NULL CHECK ( count > 0 ),
    confirmed_price BIGINT NOT NULL CHECK ( confirmed_price > 0 ),

    PRIMARY KEY (basket_id, product_id)
);

CREATE TABLE IF NOT EXISTS user_orders(
    id UUID PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    company_id UUID NOT NULL REFERENCES companies(id),

    status user_order_status NOT NULL DEFAULT 'created',
    rejection_reason TEXT,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_order_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES user_orders(id),
    status user_order_status NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (order_id, status)
);
CREATE TABLE IF NOT EXISTS user_order_items(

    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES user_orders(id),
    product_id UUID NOT NULL REFERENCES company_products(id),

    count INTEGER NOT NULL CHECK ( count > 0 ),
    product_name VARCHAR(255) NOT NULL,
    product_price BIGINT NOT NULL CHECK ( product_price > 0 ),

    UNIQUE (order_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_products_company
    ON company_products(company_id);

CREATE INDEX IF NOT EXISTS idx_orders_user_created
    ON user_orders(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_orders_company_status
    ON user_orders(company_id, status);

CREATE INDEX IF NOT EXISTS idx_status_history_order
    ON user_order_status_history(order_id, created_at);
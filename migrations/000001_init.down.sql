DROP INDEX IF EXISTS idx_status_history_order;
DROP INDEX IF EXISTS idx_orders_company_status;
DROP INDEX IF EXISTS idx_orders_user_created;
DROP INDEX IF EXISTS idx_products_company;

DROP TABLE IF EXISTS user_order_items;
DROP TABLE IF EXISTS user_order_status_history;
DROP TABLE IF EXISTS user_orders;
DROP TABLE IF EXISTS user_basket_products;
DROP TABLE IF EXISTS user_baskets;
DROP TABLE IF EXISTS company_products;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS companies;

DROP TYPE IF EXISTS user_order_status;

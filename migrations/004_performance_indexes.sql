-- 004_performance_indexes.sql
-- Adds indexes for hot query paths flagged in the production audit.
-- All IF NOT EXISTS so the migration is idempotent. Apply via:
--   psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/004_performance_indexes.sql
--
-- Notes:
--  * orders.user_id is already covered by idx_orders_user (001).
--  * food_products is filtered by outlet_id (idx_food_products_outlet, 001) —
--    there is no `restaurant` column, so no index is added there.

-- Login / user lookup by email (repository.GetUserByEmail)
CREATE INDEX IF NOT EXISTS idx_users_email ON public.users(email);

-- Order-detail JOINs and item fetches by order
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON public.order_items(order_id);

-- Order list filtering (GetOrdersWithFilters) by outlet / restaurant / status
CREATE INDEX IF NOT EXISTS idx_orders_outlet_id ON public.orders(outlet_id);
CREATE INDEX IF NOT EXISTS idx_orders_restaurant_id ON public.orders(restaurant_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON public.orders(status);

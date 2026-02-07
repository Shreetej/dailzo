-- Migration: 001_new_tables.sql
-- Description: Add tables for delivery, grocery, admin modules and enhance existing tables
-- Date: 2026-02-04

BEGIN;

-- ============================================
-- DELIVERY MODULE TABLES
-- ============================================

-- Delivery Profiles (extended partner information)
CREATE TABLE IF NOT EXISTS public.delivery_profiles (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES public.users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    city VARCHAR(100),
    vehicle_type VARCHAR(50),
    vehicle_number VARCHAR(50),
    license_number VARCHAR(100),
    kyc_status VARCHAR(20) DEFAULT 'pending' CHECK (kyc_status IN ('pending', 'submitted', 'verified', 'rejected')),
    kyc_documents JSONB,
    rating DECIMAL(3,2) DEFAULT 0.00,
    total_trips INT DEFAULT 0,
    is_online BOOLEAN DEFAULT false,
    current_lat DECIMAL(10,8),
    current_lng DECIMAL(11,8),
    last_location_update TIMESTAMP,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    last_modified_by VARCHAR(255)
);

-- Delivery Tasks (active delivery assignments)
CREATE TABLE IF NOT EXISTS public.delivery_tasks (
    id VARCHAR(50) PRIMARY KEY,
    order_id VARCHAR(255) REFERENCES public.orders(id) ON DELETE CASCADE,
    delivery_person_id VARCHAR(50) REFERENCES public.delivery_profiles(id) ON DELETE SET NULL,
    status VARCHAR(30) DEFAULT 'assigned' CHECK (status IN ('assigned', 'accepted', 'picked_up', 'in_transit', 'delivered', 'cancelled')),
    pickup_lat DECIMAL(10,8),
    pickup_lng DECIMAL(11,8),
    dropoff_lat DECIMAL(10,8),
    dropoff_lng DECIMAL(11,8),
    eta_mins INT,
    pickup_point TEXT,
    drop_point TEXT,
    distance_km DECIMAL(6,2),
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    accepted_at TIMESTAMP,
    picked_up_at TIMESTAMP,
    delivered_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    cancel_reason TEXT,
    notes TEXT,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Delivery Trace Events (location and status history)
CREATE TABLE IF NOT EXISTS public.delivery_trace_events (
    id VARCHAR(50) PRIMARY KEY,
    order_id VARCHAR(255) REFERENCES public.orders(id) ON DELETE CASCADE,
    delivery_task_id VARCHAR(50) REFERENCES public.delivery_tasks(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    actor_type VARCHAR(20) CHECK (actor_type IN ('system', 'delivery_partner', 'customer', 'restaurant', 'admin')),
    lat DECIMAL(10,8),
    lng DECIMAL(11,8),
    notes TEXT
);

-- Delivery Shifts (available work shifts)
CREATE TABLE IF NOT EXISTS public.delivery_shifts (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    zone VARCHAR(100),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    max_partners INT DEFAULT 50,
    incentive_multiplier DECIMAL(3,2) DEFAULT 1.00,
    is_active BOOLEAN DEFAULT true,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Delivery Shift Bookings (partner shift reservations)
CREATE TABLE IF NOT EXISTS public.delivery_shift_bookings (
    id VARCHAR(50) PRIMARY KEY,
    shift_id VARCHAR(50) REFERENCES public.delivery_shifts(id) ON DELETE CASCADE,
    delivery_person_id VARCHAR(50) REFERENCES public.delivery_profiles(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'booked' CHECK (status IN ('booked', 'started', 'completed', 'cancelled', 'no_show')),
    check_in_time TIMESTAMP,
    check_out_time TIMESTAMP,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shift_id, delivery_person_id, date)
);

-- Delivery Recommendations (AI suggestions for partners)
CREATE TABLE IF NOT EXISTS public.delivery_recommendations (
    id VARCHAR(50) PRIMARY KEY,
    delivery_person_id VARCHAR(50) REFERENCES public.delivery_profiles(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('zone_change', 'shift_suggestion', 'break_reminder', 'incentive_alert', 'safety_tip')),
    title VARCHAR(255),
    message TEXT NOT NULL,
    priority VARCHAR(20) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
    cta_route VARCHAR(255),
    data JSONB,
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

-- Daily Delivery KPIs (aggregated daily metrics)
CREATE TABLE IF NOT EXISTS public.daily_delivery_kpis (
    id VARCHAR(50) PRIMARY KEY,
    delivery_person_id VARCHAR(50) REFERENCES public.delivery_profiles(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    today_earnings DECIMAL(10,2) DEFAULT 0,
    completed_trips INT DEFAULT 0,
    online_minutes INT DEFAULT 0,
    acceptance_rate DECIMAL(5,2) DEFAULT 0,
    on_time_rate DECIMAL(5,2) DEFAULT 0,
    avg_delivery_time_mins INT DEFAULT 0,
    total_distance_km DECIMAL(8,2) DEFAULT 0,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(delivery_person_id, date)
);

-- ============================================
-- GROCERY MODULE TABLES
-- ============================================

-- Grocery Profiles (vendor/store information)
CREATE TABLE IF NOT EXISTS public.grocery_profiles (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES public.users(id) ON DELETE CASCADE,
    store_name VARCHAR(255) NOT NULL,
    owner_name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(20) NOT NULL,
    address TEXT,
    city VARCHAR(100),
    pincode VARCHAR(10),
    kyc_status VARCHAR(20) DEFAULT 'pending' CHECK (kyc_status IN ('pending', 'submitted', 'verified', 'rejected')),
    kyc_documents JSONB,
    fssai_license VARCHAR(100),
    gst_number VARCHAR(50),
    pan_number VARCHAR(20),
    payout_status VARCHAR(20) DEFAULT 'pending' CHECK (payout_status IN ('pending', 'active', 'suspended')),
    bank_details JSONB,
    working_hours VARCHAR(100),
    is_active BOOLEAN DEFAULT false,
    rating DECIMAL(3,2) DEFAULT 0.00,
    total_orders INT DEFAULT 0,
    commission_rate DECIMAL(5,2) DEFAULT 15.00,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    last_modified_by VARCHAR(255)
);

-- Daily Grocery KPIs (aggregated daily metrics)
CREATE TABLE IF NOT EXISTS public.daily_grocery_kpis (
    id VARCHAR(50) PRIMARY KEY,
    grocery_id VARCHAR(50) REFERENCES public.grocery_profiles(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    today_revenue DECIMAL(10,2) DEFAULT 0,
    pending_orders INT DEFAULT 0,
    completed_orders INT DEFAULT 0,
    cancelled_orders INT DEFAULT 0,
    cancel_rate DECIMAL(5,2) DEFAULT 0,
    avg_prep_time_mins INT DEFAULT 0,
    low_stock_items INT DEFAULT 0,
    expiry_risk_items INT DEFAULT 0,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(grocery_id, date)
);

-- Grocery Payouts (payout records)
CREATE TABLE IF NOT EXISTS public.grocery_payouts (
    id VARCHAR(50) PRIMARY KEY,
    grocery_id VARCHAR(50) REFERENCES public.grocery_profiles(id) ON DELETE CASCADE,
    amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    payout_date DATE,
    transaction_id VARCHAR(100),
    bank_reference VARCHAR(100),
    notes TEXT,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- ADMIN MODULE TABLES
-- ============================================

-- Approvals (KYC and onboarding approvals)
CREATE TABLE IF NOT EXISTS public.approvals (
    id VARCHAR(50) PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL CHECK (entity_type IN ('delivery', 'grocery', 'restaurant')),
    entity_id VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMP,
    reviewed_by VARCHAR(255),
    notes TEXT,
    documents JSONB,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Complaints (customer complaints and issues)
CREATE TABLE IF NOT EXISTS public.complaints (
    id VARCHAR(50) PRIMARY KEY,
    order_id VARCHAR(255) REFERENCES public.orders(id) ON DELETE SET NULL,
    user_id VARCHAR(255) REFERENCES public.users(id) ON DELETE SET NULL,
    complaint_type VARCHAR(50) CHECK (complaint_type IN ('delivery_delay', 'food_quality', 'missing_item', 'wrong_order', 'rude_behavior', 'payment_issue', 'other')),
    description TEXT,
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'investigating', 'resolved', 'closed')),
    priority VARCHAR(10) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    culprit VARCHAR(50),
    reason_code VARCHAR(50),
    confidence DECIMAL(3,2),
    evidence_timeline JSONB,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_on TIMESTAMP,
    resolved_by VARCHAR(255),
    resolution_notes TEXT,
    refund_amount DECIMAL(10,2) DEFAULT 0,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Partner Suspensions (suspension records)
CREATE TABLE IF NOT EXISTS public.partner_suspensions (
    id VARCHAR(50) PRIMARY KEY,
    partner_id VARCHAR(50) NOT NULL,
    partner_type VARCHAR(20) NOT NULL CHECK (partner_type IN ('delivery', 'grocery', 'restaurant')),
    reason TEXT,
    suspended_by VARCHAR(255),
    suspended_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    suspension_end_date TIMESTAMP,
    reinstated_on TIMESTAMP,
    reinstated_by VARCHAR(255),
    is_permanent BOOLEAN DEFAULT false
);

-- Onboarding Leads (incomplete registrations tracking)
CREATE TABLE IF NOT EXISTS public.onboarding_leads (
    id VARCHAR(50) PRIMARY KEY,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('delivery', 'grocery', 'restaurant')),
    name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(20),
    city VARCHAR(100),
    last_step VARCHAR(100),
    total_steps INT DEFAULT 5,
    step_completed INT DEFAULT 0,
    last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    days_since_start INT DEFAULT 0,
    potential_loss DECIMAL(10,2) DEFAULT 0,
    notifications_sent INT DEFAULT 0,
    last_notification_at TIMESTAMP,
    source VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Admin Activity Log (audit trail)
CREATE TABLE IF NOT EXISTS public.admin_activity_log (
    id VARCHAR(50) PRIMARY KEY,
    admin_id VARCHAR(255) REFERENCES public.users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id VARCHAR(50),
    details JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- PRODUCT ENHANCEMENTS (ALTER EXISTING TABLE)
-- ============================================

-- Add new columns to food_products for inventory management
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS stock_quantity INT DEFAULT 0;
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS low_stock_threshold INT DEFAULT 10;
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS expiry_date DATE;
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS batch_number VARCHAR(50);
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS is_promo BOOLEAN DEFAULT false;
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS promo_price DECIMAL(10,2);
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS auto_discount_enabled BOOLEAN DEFAULT false;
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS auto_discount_days_before_expiry INT DEFAULT 3;
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS auto_discount_pct DECIMAL(5,2) DEFAULT 0;
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS outlet_id VARCHAR(50);
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS storage_type VARCHAR(20);
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS brand VARCHAR(100);
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS unit VARCHAR(20);
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS weight DECIMAL(10,2);
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS shelf_life_days INT;

-- Add outlet_id to orders for filtering
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS outlet_id VARCHAR(50);
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS delivery_status VARCHAR(30);
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS pickup_otp VARCHAR(6);
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS delivery_otp VARCHAR(6);
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS estimated_delivery_time TIMESTAMP;
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS actual_delivery_time TIMESTAMP;

-- ============================================
-- INDEXES FOR PERFORMANCE
-- ============================================

-- Delivery indexes
CREATE INDEX IF NOT EXISTS idx_delivery_profiles_user_id ON public.delivery_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_delivery_profiles_city ON public.delivery_profiles(city);
CREATE INDEX IF NOT EXISTS idx_delivery_profiles_is_online ON public.delivery_profiles(is_online);
CREATE INDEX IF NOT EXISTS idx_delivery_tasks_status ON public.delivery_tasks(status);
CREATE INDEX IF NOT EXISTS idx_delivery_tasks_delivery_person ON public.delivery_tasks(delivery_person_id);
CREATE INDEX IF NOT EXISTS idx_delivery_tasks_order ON public.delivery_tasks(order_id);
CREATE INDEX IF NOT EXISTS idx_delivery_trace_order ON public.delivery_trace_events(order_id);
CREATE INDEX IF NOT EXISTS idx_delivery_trace_timestamp ON public.delivery_trace_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_daily_delivery_kpis_date ON public.daily_delivery_kpis(date);

-- Grocery indexes
CREATE INDEX IF NOT EXISTS idx_grocery_profiles_user_id ON public.grocery_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_grocery_profiles_city ON public.grocery_profiles(city);
CREATE INDEX IF NOT EXISTS idx_daily_grocery_kpis_date ON public.daily_grocery_kpis(date);

-- Admin indexes
CREATE INDEX IF NOT EXISTS idx_approvals_status ON public.approvals(status);
CREATE INDEX IF NOT EXISTS idx_approvals_entity ON public.approvals(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_complaints_status ON public.complaints(status);
CREATE INDEX IF NOT EXISTS idx_complaints_order ON public.complaints(order_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_leads_type ON public.onboarding_leads(entity_type);

-- Orders indexes (additional)
CREATE INDEX IF NOT EXISTS idx_orders_status ON public.orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_outlet ON public.orders(outlet_id);
CREATE INDEX IF NOT EXISTS idx_orders_delivery_person ON public.orders(delivery_person_id);
CREATE INDEX IF NOT EXISTS idx_orders_user ON public.orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_date ON public.orders(order_date);

-- Products indexes (additional)
CREATE INDEX IF NOT EXISTS idx_food_products_expiry ON public.food_products(expiry_date);
CREATE INDEX IF NOT EXISTS idx_food_products_stock ON public.food_products(stock_quantity);
CREATE INDEX IF NOT EXISTS idx_food_products_outlet ON public.food_products(outlet_id);
CREATE INDEX IF NOT EXISTS idx_food_products_category ON public.food_products(category);
CREATE INDEX IF NOT EXISTS idx_food_products_promo ON public.food_products(is_promo);

COMMIT;

DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plans;


CREATE TABLE IF NOT EXISTS subscription_products (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS subscription_product_attributes (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    value VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES subscription_products(id)
);


CREATE TABLE IF NOT EXISTS subscription_plans (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    price_monthly DECIMAL(10, 2) NOT NULL,
    price_yearly DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES subscription_products(id)
);


CREATE TABLE IF NOT EXISTS subscription_plan_attributes (
    plan_id VARCHAR(36) NOT NULL,
    attribute_id VARCHAR(36) NOT NULL,
    PRIMARY KEY (plan_id, attribute_id),
    FOREIGN KEY (plan_id) REFERENCES subscription_plans(id),
    FOREIGN KEY (attribute_id) REFERENCES subscription_product_attributes(id)
);


CREATE TABLE IF NOT EXISTS subscription_transactions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    product_id VARCHAR(36) NOT NULL,
    plan_id VARCHAR(36) NOT NULL,
    card_id VARCHAR(36) NOT NULL,
    is_renewal BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    payment_type VARCHAR(20) NOT NULL, 
    amount DECIMAL(10, 2) NOT NULL,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    next_renewal_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES subscription_products(id),
    FOREIGN KEY (plan_id) REFERENCES subscription_plans(id),
    FOREIGN KEY (card_id) REFERENCES cards(id)
);


INSERT INTO subscription_products (id, name)
VALUES ('prod-netflix-001', 'Netflix');


INSERT INTO subscription_product_attributes (id, product_id, name, value)
VALUES 
('attr-001', 'prod-netflix-001', 'screens', '1'),
('attr-002', 'prod-netflix-001', 'screens', '2'),
('attr-003', 'prod-netflix-001', 'quality', 'HDTV'),
('attr-004', 'prod-netflix-001', 'quality', '4K'),
('attr-005', 'prod-netflix-001', 'quality', 'Ultra HD');


INSERT INTO subscription_plans (id, product_id, name, price_monthly, price_yearly)
VALUES 
('plan-001', 'prod-netflix-001', 'Budget', 9.99, 99.99),
('plan-002', 'prod-netflix-001', 'Standard', 14.99, 149.99),
('plan-003', 'prod-netflix-001', 'Premium', 19.99, 199.99);


INSERT INTO subscription_plan_attributes (plan_id, attribute_id)
VALUES 
-- Budget plan: 1 screen + HDTV
('plan-001', 'attr-001'),
('plan-001', 'attr-003'),
-- Standard plan: 1 screen + 4K
('plan-002', 'attr-001'),
('plan-002', 'attr-004'),
-- Premium plan: 2 screens + Ultra HD
('plan-003', 'attr-002'),
('plan-003', 'attr-005');
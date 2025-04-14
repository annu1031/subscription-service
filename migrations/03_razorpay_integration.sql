ALTER TABLE subscription_transactions
ADD COLUMN razorpay_order_id VARCHAR(100) NULL,
ADD COLUMN razorpay_payment_id VARCHAR(100) NULL,
ADD COLUMN razorpay_subscription_id VARCHAR(100) NULL,
ADD COLUMN razorpay_key_id VARCHAR(100) NULL,
ADD COLUMN auto_renewal BOOLEAN DEFAULT false;


CREATE TABLE IF NOT EXISTS razorpay_webhook_events (
    id VARCHAR(36) PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload TEXT NOT NULL,
    processed BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE SCHEMA account;

CREATE TABLE account.customer (
    customerid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customerstatusid INT CHECK (customerstatusid BETWEEN 1 AND 6),
    amount NUMERIC(10,2),
    updated_at TIMESTAMP,
    customer_email TEXT UNIQUE
);

INSERT INTO account.customer (customerstatusid, amount, updated_at, customer_email)
VALUES
    (1, 500.00, NOW(), 'customer1@example.com'),
    (2, -150.75, NOW(), 'customer2@example.com'),
    (3, 1200.00, NOW(), 'customer3@example.com');

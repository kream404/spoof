CREATE SCHEMA account;

CREATE TABLE account.customer (
    customer_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customerstatusid INT CHECK (customerstatusid BETWEEN 1 AND 6),
    amount NUMERIC(10,2),
    updated_at TIMESTAMP,
    customer_email TEXT UNIQUE
);

DO $$
BEGIN
  INSERT INTO account.customer (customerstatusid, amount, updated_at, customer_email)
  SELECT
    FLOOR(random() * 6 + 1)::int AS customerstatusid,
    ROUND((random() * 2000 - 1000)::numeric, 2) AS amount, -- range: -1000 to 1000
    NOW() - (random() * interval '365 days') AS updated_at,
    'customer' || gs::text || '@example.com' AS customer_email
  FROM generate_series(4, 5003) AS gs; -- skip 1-3 (already inserted)
END
$$;

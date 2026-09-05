
-- Shops table (one user → one shop)
CREATE TABLE IF NOT EXISTS shops (
    id SERIAL PRIMARY KEY,
    user_id INT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    address_id INT NOT NULL REFERENCES addresses(id) ON DELETE CASCADE,
    timing VARCHAR(100),
    category VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
	
CREATE TABLE shops (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    address_id int references addresses(id) on delete cascade,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    category TEXT[]
);

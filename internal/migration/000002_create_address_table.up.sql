CREATE TABLE IF NOT EXISTS addresses (
    id SERIAL PRIMARY KEY,
    village   VARCHAR(100),
    post      VARCHAR(100),
    district  VARCHAR(100),
    state     VARCHAR(100),
    pincode   VARCHAR(20),
    landmark  VARCHAR(150),
    latitude  DECIMAL(9,6), -- for distance calculation
    longitude DECIMAL(9,6)
);

CREATE INDEX idx_addresses_pincode ON addresses(pincode);
CREATE INDEX idx_addresses_district ON addresses(district);


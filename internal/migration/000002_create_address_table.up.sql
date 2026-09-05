CREATE TABLE IF NOT EXISTS addresses (
    id SERIAL PRIMARY KEY,
    village   VARCHAR(100) not null,
    post      VARCHAR(100),
    district  VARCHAR(100) not null,
    state     VARCHAR(100) not null,
    pincode   VARCHAR(20) check (pincode ~ '^[0-9]{6}$'),
    landmark  VARCHAR(150) check (landmark ~ '^[a-zA-Z0-9\s,.-]*$'),
    latitude  DECIMAL(9,6) check (latitude >= -90 and latitude <= 90), -- for distance calculation
    longitude DECIMAL(9,6) check (longitude >= -180 and longitude <= 180)
);



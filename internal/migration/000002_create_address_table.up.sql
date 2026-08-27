
CREATE TABLE addresses (
    id SERIAL PRIMARY KEY,
    village   VARCHAR(100),
    post      VARCHAR(100),
    district  VARCHAR(100),
    state     VARCHAR(100),
    pincode   VARCHAR(20),
    landmark  VARCHAR(150)
);

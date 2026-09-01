CREATE TABLE users (
    id SERIAL PRIMARY KEY,
	name varchar(100),
    email VARCHAR(100) unique,
    password VARCHAR(100)
);
	
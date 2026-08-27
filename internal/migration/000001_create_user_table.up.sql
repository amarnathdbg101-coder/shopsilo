CREATE TABLE IF NOT EXISTS users(
	id serial primary key,
	name text null,
	email text null,
	password text null
);
CREATE TABLE if NOT EXISTS  products(
	id serial primary key,
	name text null,
	title text null,
	price float,
	create_at timestamp default now()
);
CREATE TABLE IF NOT EXISTS prices(
    id serial PRIMARY KEY, 
    name TEXT NOT NULL, 
    category TEXT NOT NULL, 
    price NUMERIC(10, 2) NOT NULL, 
    created_at DATE NOT NULL
);
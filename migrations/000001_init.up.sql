CREATE TABLE IF NOT EXISTS prices(
    id          INTEGER NOT NULL, 
    name        VARCHAR(50),
    category    VARCHAR(50),
    price       FLOAT,
    create_date DATE
);
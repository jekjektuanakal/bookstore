CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    "user" VARCHAR(255) NOT NULL,
    date TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(255)
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    "user" VARCHAR(255) NOT NULL,
    order_id INT NOT NULL REFERENCES orders(id),
    book_id INT NOT NULL,
    quantity INT NOT NULL,
    UNIQUE ("user", order_id)
);

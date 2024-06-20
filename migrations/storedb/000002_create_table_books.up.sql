CREATE TABLE IF NOT EXISTS books (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author VARCHAR(255) NOT NULL,
    price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO books (title, author, price)
    VALUES 
    ('The Great Gatsby', 'F. Scott Fitzgerald', 19.99),
    ('To Kill a Mockingbird', 'Harper Lee', 14.99),
    ('1984', 'George Orwell', 9.99),
    ('Pride and Prejudice', 'Jane Austen', 12.99),
    ('The Catcher in the Rye', 'J.D. Salinger', 11.99),
    ('The Hobbit', 'J.R.R. Tolkien', 15.99),
    ('The Lord of the Rings', 'J.R.R. Tolkien', 29.99),
    ('Animal Farm', 'George Orwell', 8.99),
    ('Brave New World', 'Aldous Huxley', 10.99),
    ('The Grapes of Wrath', 'John Steinbeck', 13.99);

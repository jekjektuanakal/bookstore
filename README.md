# Book Store Backend Example

## Overview

Bookstore example to demonstrate backend API using Go and PostgreSQL.

## Features

1. First Party Registration and Login
2. Getting all books
3. Creating order for books

## Testing

```bash
# Starting the database
docker-compose up --detach --build 

# Running the tests
go test ./internal/test/... -v
```

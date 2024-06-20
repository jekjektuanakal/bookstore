package store

import (
	"database/sql"
	"time"
)

func insertLogin(db *sql.DB, login Login) (err error) {
	_, err = db.Exec("INSERT INTO logins (email, hash) VALUES ($1, $2)", login.Email, login.Hash)

	return err
}

func getLoginByEmail(db *sql.DB, email string) (login Login, err error) {
	err = db.
		QueryRow("SELECT email, hash FROM logins WHERE email = $1", email).
		Scan(&login.Email, &login.Hash)

	return login, err
}

func getBooks(db *sql.DB) (books []Book, err error) {
	rows, err := db.Query("SELECT id, title, author FROM books")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var book Book
		if err := rows.Scan(&book.ID, &book.Title, &book.Author); err != nil {
			return nil, err
		}
		books = append(books, book)
	}
	return books, nil
}

func insertOrders(db *sql.DB, orderRequest OrderRequest) (order Order, err error) {
	tx, err := db.Begin()
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()

	err = tx.
		QueryRow(
			`INSERT INTO orders ("user", date, status) VALUES ($1, $2, $3) RETURNING id, "user", date, status`,
			orderRequest.User,
			time.Now(),
			"pending",
		).
		Scan(&order.ID, &order.User, &order.Date, &order.Status)
	if err != nil {
		return Order{}, err
	}

	stmt, err := tx.Prepare(`INSERT INTO order_items ("user", order_id, book_id, quantity) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		return Order{}, err
	}
	defer stmt.Close()

	for _, item := range orderRequest.Items {
		_, err := stmt.Exec(orderRequest.User, order.ID, item.BookID, item.Quantity)
		if err != nil {
			return Order{}, err
		}
	}
	err = tx.Commit()

	return order, err
}

func getOrdersByUser(db *sql.DB, user string) (orders []OrderDetail, err error) {
	query := `
    SELECT o.id, o.user, o.date, o.status, oi.id, oi.book_id, oi.quantity
    FROM orders o
    JOIN order_items oi ON o.id = oi.order_id
    WHERE o.user = $1
    ORDER BY o.date DESC
    `
	lastOrder := OrderDetail{}

	rows, err := db.Query(query, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var order OrderDetail
		var item OrderItem

		if err := rows.Scan(
			&order.ID,
			&order.User,
			&order.Date,
			&order.Status,
			&item.ID,
			&item.BookID,
			&item.Quantity,
		); err != nil {
			return nil, err
		}
		if lastOrder.ID != order.ID {
			orders = append(orders, order)
			lastOrder = order
		}

		orders[len(orders)-1].Items = append(orders[len(orders)-1].Items, item)
	}

	return orders, nil
}

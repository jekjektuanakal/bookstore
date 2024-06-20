package store

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

type Secrets interface {
	GetAuthKey() string
}

type App struct {
	authKey ed25519.PrivateKey
	db      *sql.DB
}

func NewApp(secrets Secrets, db *sql.DB) App {
	seed, err := hex.DecodeString(secrets.GetAuthKey())
	if err != nil {
		panic(err)
	}

	return App{
		authKey: ed25519.NewKeyFromSeed(seed),
		db:      db,
	}
}

type User struct {
	Email    string
	Password string
}

type Login struct {
	Email string
	Hash  string
}

func (a *App) RegisterUser(user User) error {
	if _, err := mail.ParseAddress(user.Email); err != nil {
		return fmt.Errorf("invalid email: %w %s", err, user.Email)
	}

	if user.Password == "" {
		return fmt.Errorf("password is required: %w", ErrInvalid)
	}

	hash := hashPassword(user.Password)

	err := insertLogin(a.db, Login{Email: user.Email, Hash: hash})
	if err != nil {
		return fmt.Errorf("failed to insert login: %w: %w", err, ErrConflict)
	}

	return nil
}

func (a *App) LoginUser(user User) (token jwt.RegisteredClaims, err error) {
	login, err := getLoginByEmail(a.db, user.Email)
	if err != nil {
		return jwt.RegisteredClaims{}, fmt.Errorf("login failed: %w", ErrUnauthorized)
	}

	if !verifyPassword(user.Password, login.Hash) {
		return jwt.RegisteredClaims{}, fmt.Errorf("login failed: %w", ErrUnauthorized)
	}

	return jwt.RegisteredClaims{
		Issuer:    "gotu",
		Subject:   user.Email,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}, nil
}

func hashPassword(password string) string {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		panic(err)
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 1, 32)

	return base64.StdEncoding.EncodeToString(hash) + "." + base64.StdEncoding.EncodeToString(salt)
}

func verifyPassword(password, hash string) bool {
	parts := strings.Split(hash, ".")
	if len(parts) != 2 {
		return false
	}
	decodedHash, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	decodedSalt, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	comparisonHash := argon2.IDKey([]byte(password), decodedSalt, 1, 64*1024, 1, 32)
	return subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1
}

type Book struct {
	ID     int
	Title  string
	Author string
}

func (a *App) GetBooks() ([]Book, error) {
	return getBooks(a.db)
}

type OrderStatus string

const OrderStatusPending = OrderStatus("pending")

type Order struct {
	ID     int
	User   string
	Date   time.Time
	Status OrderStatus
}

type OrderItem struct {
	ID       int
	User     string
	OrderID  int
	BookID   int
	Quantity int
}

type OrderDetail struct {
	Order
	Items []OrderItem
}

func (a *App) GetOrdersByUser(user string) ([]OrderDetail, error) {
	return getOrdersByUser(a.db, user)
}

type OrderItemRequest struct {
	BookID   int
	Quantity int
}

type OrderRequest struct {
	User  string
	Items []OrderItemRequest
}

func (a *App) CreateOrder(orderRequest OrderRequest) (Order, error) {
	if len(orderRequest.Items) == 0 {
		return Order{}, fmt.Errorf("order must have at least one item: %w", ErrInvalid)
	}

	order, err := insertOrders(a.db, orderRequest)
	if err != nil {
		return Order{}, fmt.Errorf("failed to insert order: %w: %w", err, ErrConflict)
	}

	return order, nil
}

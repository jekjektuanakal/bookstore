package store

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kelseyhightower/envconfig"

	jwtware "github.com/gofiber/contrib/jwt"
)

type ServerConfig struct {
	Port     int
	BasePath string
}

func ParseServerConfig() ServerConfig {
	cfg := ServerConfig{}
	envconfig.MustProcess("BOOKSTORE_SERVER", &cfg)

	return cfg
}

type Server struct {
	router  *fiber.App
	port    int
	app     App
	authKey ed25519.PrivateKey
}

func NewServer(cfg ServerConfig, secrets Secrets, db *sql.DB) Server {
	router := fiber.New()
	server := Server{
		router: router,
		port:   cfg.Port,
		app:    NewApp(secrets, db),
	}

	v1 := router.Group("/v1" + cfg.BasePath)
	v1.Get("/health", server.getHealth)
	v1.Post("/users", server.postUsers)
	v1.Post("/login", server.login)

	seed, err := hex.DecodeString(secrets.GetAuthKey())
	if err != nil {
		panic(err)
	}

	server.authKey = ed25519.NewKeyFromSeed(seed)

	v1.Use(
		jwtware.New(
			jwtware.Config{
				SigningKey: jwtware.SigningKey{
					JWTAlg: "EdDSA",
					Key:    server.authKey.Public(),
				},
				ErrorHandler: func(c *fiber.Ctx, err error) error {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
				},
			},
		),
	)

	v1.Get("/books", server.getBooks)
	v1.Get("/orders", server.getOrders)
	v1.Post("/orders", server.createOrder)
	return server
}

func (s *Server) Start() {
	s.router.Listen(":" + strconv.Itoa(s.port))
}

func (s *Server) Stop() error {
	return s.router.Shutdown()
}

func (s *Server) Test(req *http.Request, msTimeout ...int) (*http.Response, error) {
	return s.router.Test(req, msTimeout...)
}

func (s *Server) getHealth(c *fiber.Ctx) error {
	c.SendStatus(fiber.StatusOK)

	return nil
}

func (s *Server) postUsers(c *fiber.Ctx) error {
	user := &User{}
	err := c.BodyParser(user)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	err = s.app.RegisterUser(*user)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, ErrInternal) {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": user.Email})
}

func (s *Server) login(c *fiber.Ctx) error {
	var user User
	var err error
	user.Email, user.Password, err = decodeBasicAuth(c.Get("Authorization"))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	token, err := s.app.LoginUser(user)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	signedString, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, token).SignedString(s.authKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"token": signedString})
}

func decodeBasicAuth(authHeader string) (string, string, error) {
	if authHeader == "" {
		return "", "", errors.New("missing authorization header")
	}
	if len(authHeader) < 6 || authHeader[:6] != "Basic " {
		return "", "", errors.New("invalid authorization header")
	}
	decoded, err := base64.StdEncoding.DecodeString(authHeader[6:])
	if err != nil {
		return "", "", errors.New("failed to decode authorization header")
	}
	creds := strings.SplitN(string(decoded), ":", 2)
	if len(creds) != 2 {
		return "", "", errors.New("invalid authorization header")
	}
	return creds[0], creds[1], nil
}

func (s *Server) getBooks(c *fiber.Ctx) error {
	books, err := s.app.GetBooks()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"books": books})
}

func (s *Server) getOrders(c *fiber.Ctx) error {
	user := c.Locals("user").(*jwt.Token)
	userSubject, err := user.Claims.GetSubject()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	orders, err := s.app.GetOrdersByUser(userSubject)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"orders": orders})
}

func (s *Server) createOrder(c *fiber.Ctx) error {
	user := c.Locals("user").(*jwt.Token)
	userSubject, err := user.Claims.GetSubject()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	orderRequest := OrderRequest{}
	err = c.BodyParser(&orderRequest)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	orderRequest.User = userSubject

	order, err := s.app.CreateOrder(orderRequest)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(order)
}

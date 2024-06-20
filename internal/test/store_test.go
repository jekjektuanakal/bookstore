package test

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"bookstore.example/store/internal/infra"
	"bookstore.example/store/internal/store"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/stretchr/testify/suite"
)

type StoreTestSuite struct {
	suite.Suite
	server  *store.Server
	secrets store.Secrets
	db      *sql.DB
}

type testSecret struct{}

func (s *testSecret) GetAuthKey() string {
	return "64D4D1E3ABE3C7A2EB09305A1C8A7B896110674735C671E5586D968ED0561415"
}

func (s *StoreTestSuite) SetupSuite() {
	godotenv.Load("../../.env")

	s.secrets = &testSecret{}

	dbCfg := infra.ParsePostgresDBConfig()
	s.T().Logf("dbCfg: %+v", dbCfg)
	s.db = infra.NewDB(infra.ParsePostgresDBConfig())
	infra.Migrate(s.db, "../../migrations/storedb")

	cfg := store.ParseServerConfig()
	server := store.NewServer(cfg, s.secrets, s.db)
	s.server = &server

	go s.server.Start()
}

func (s *StoreTestSuite) TearDownSuite() {
	s.server.Stop()
	s.db.Close()
}

func (s *StoreTestSuite) TestRegistration() {
	s.Run("register a new user without email, expect 400", func() {
		req := httptest.NewRequest(
			"POST",
			"/v1/users",
			strings.NewReader(`{"password": "password"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		rsp, _ := s.server.Test(req)

		s.Equal(400, rsp.StatusCode)
	})

	s.Run("register a new user without password, expect 400", func() {
		req := httptest.NewRequest(
			"POST",
			"/v1/users",
			strings.NewReader(`{"email": "email@domain.example"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		rsp, _ := s.server.Test(req)

		s.Equal(400, rsp.StatusCode)
	})

	s.Run("register a new user with invalid email, expect 400", func() {
		req := httptest.NewRequest(
			"POST",
			"/v1/users",
			strings.NewReader(`{"email": "invalid", "password": "password"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		rsp, _ := s.server.Test(req)

		s.Equal(400, rsp.StatusCode)
	})

	s.Run("register a new user with valid email and password, expect 201", func() {
		s.db.Exec("DELETE FROM logins WHERE email = 'email@domain.example'")

		req := httptest.NewRequest(
			"POST",
			"/v1/users",
			strings.NewReader(`{"email": "email@domain.example", "password": "password"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		rsp, _ := s.server.Test(req)

		s.Equal(201, rsp.StatusCode)
	})

	s.Run("register a new user with duplicated email, expect 400", func() {
		req := httptest.NewRequest(
			"POST",
			"/v1/users",
			strings.NewReader(`{"email": "email@domain.example", "password": "password"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		rsp, _ := s.server.Test(req)

		_, err := s.db.Exec("DELETE FROM logins WHERE email = 'email@domain.example'")
		if err != nil {
			s.T().Errorf("failed to delete user: %v", err)
		}

		s.Equal(409, rsp.StatusCode)
	})
}

func (s *StoreTestSuite) TestLogin() {
	s.Run("login with unregistered email, expect 401", func() {
		s.db.Exec("DELETE FROM logins WHERE email = 'budi@domain.example'")

		req := httptest.NewRequest(
			"POST",
			"/v1/login",
			nil,
		)
		req.Header.Set("Content-Type", "application/json")
		basicAuth := base64.StdEncoding.EncodeToString([]byte("budi@domain.example:password1"))
		req.Header.Set("Authorization", "Basic "+basicAuth)

		rsp, _ := s.server.Test(req)

		s.Equal(401, rsp.StatusCode)
	})

	req := httptest.NewRequest(
		"POST",
		"/v1/users",
		strings.NewReader(`{"email": "budi@domain.example", "password": "password1"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rsp, _ := s.server.Test(req)

	s.Assert().Equal(201, rsp.StatusCode)

	s.Run("login with registered email", func() {
		req := httptest.NewRequest(
			"POST",
			"/v1/login",
			nil,
		)

		req.Header.Set("Content-Type", "application/json")
		basicAuth := base64.StdEncoding.EncodeToString([]byte("budi@domain.example:password1"))
		req.Header.Set("Authorization", "Basic "+basicAuth)

		rsp, _ := s.server.Test(req)

		s.Run("expect 200", func() {
			s.Equal(200, rsp.StatusCode)
		})

		s.Run("expect valid token in response", func() {
			var rspBody struct{ Token string }
			json.NewDecoder(rsp.Body).Decode(&rspBody)
			s.NotEmpty(rspBody.Token)
		})
	})

	s.Run("login with invalid password, expect 401", func() {
		req := httptest.NewRequest(
			"POST",
			"/v1/login",
			nil,
		)
		req.Header.Set("Content-Type", "application/json")
		basicAuth := base64.StdEncoding.EncodeToString([]byte("budi@domain.example:password2"))
		req.Header.Set("Authorization", "Basic "+basicAuth)

		rsp, _ := s.server.Test(req)

		s.Equal(401, rsp.StatusCode)
	})
}

func (s *StoreTestSuite) TestOrders() {
	s.Run("get books without token, expect 401", func() {
		req := httptest.NewRequest("GET", "/v1/books", nil)
		req.Header.Set("Content-Type", "application/json")
		rsp, _ := s.server.Test(req)

		s.Equal(401, rsp.StatusCode)
	})

	req := httptest.NewRequest(
		"POST",
		"/v1/users",
		strings.NewReader(`{"email": "cahyo@domain.example", "password": "password"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rsp, _ := s.server.Test(req)

	req = httptest.NewRequest(
		"POST",
		"/v1/login",
		nil,
	)
	req.Header.Set("Content-Type", "application/json")
	basicAuth := base64.StdEncoding.EncodeToString([]byte("budi@domain.example:password1"))
	req.Header.Set("Authorization", "Basic "+basicAuth)

	rsp, _ = s.server.Test(req)

	s.Equal(200, rsp.StatusCode)

	var token struct{ Token string }
	json.NewDecoder(rsp.Body).Decode(&token)

	s.Run("get books with valid token", func() {
		req := httptest.NewRequest(
			"GET",
			"/v1/books",
			nil,
		)
		req.Header.Set("Authorization", "Bearer "+token.Token)
		req.Header.Set("Content-Type", "application/json")

		rsp, _ = s.server.Test(req)

		s.Run("expect 200", func() {
			s.Equal(200, rsp.StatusCode)
		})

		s.Run("expect valid books in response", func() {
			var books struct{ Books []store.Book }
			json.NewDecoder(rsp.Body).Decode(&books)
			s.NotEmpty(books)
		})
	})

	s.Run("get orders without order created, expect empty response", func() {
		s.db.Exec("DELETE FROM orders")
		s.db.Exec("DELETE FROM order_items")
		req := httptest.NewRequest(
			"GET",
			"/v1/orders",
			nil,
		)
		req.Header.Set("Authorization", "Bearer "+token.Token)
		req.Header.Set("Content-Type", "application/json")
		rsp, _ := s.server.Test(req)
		s.Equal(200, rsp.StatusCode)
		var orderResponse struct{ Orders []store.Order }
		json.NewDecoder(rsp.Body).Decode(&orderResponse)
		s.Empty(orderResponse.Orders)
	})

	s.Run("create order without item, expect 400", func() {
		req := httptest.NewRequest(
			"POST",
			"/v1/orders",
			strings.NewReader(`{ "items": [] }`),
		)
		req.Header.Set("Authorization", "Bearer "+token.Token)
		req.Header.Set("Content-Type", "application/json")

		rsp, err := s.server.Test(req)

		s.NoError(err)
		s.Equal(400, rsp.StatusCode)
	})

	s.Run("create order with item, expect 201", func() {
		req := httptest.NewRequest(
			"POST",
			"/v1/orders",
			strings.NewReader(`{"items": [{ "bookId": 1, "quantity": 1 }]}`),
		)
		req.Header.Set("Authorization", "Bearer "+token.Token)
		req.Header.Set("Content-Type", "application/json")

		rsp, err := s.server.Test(req)

		s.NoError(err)
		s.Equal(201, rsp.StatusCode)

		s.Run("get orders after order created, expect not empty response", func() {
			req := httptest.NewRequest(
				"GET",
				"/v1/orders",
				nil,
			)
			req.Header.Set("Authorization", "Bearer "+token.Token)
			req.Header.Set("Content-Type", "application/json")

			rsp, _ := s.server.Test(req)

			s.Equal(200, rsp.StatusCode)
			var orderResponse struct{ Orders []store.Order }
			json.NewDecoder(rsp.Body).Decode(&orderResponse)

			s.NotEmpty(orderResponse.Orders)

			s.db.Exec("DELETE FROM orders WHERE user = 'cahyo@domain.example'")
			s.db.Exec("DELETE FROM order_items WHERE user = 'cahyo@domain.example'")
		})
	})
}

func TestStore(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}

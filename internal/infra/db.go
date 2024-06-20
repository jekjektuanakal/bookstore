package infra

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type PostgresDBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

func ParsePostgresDBConfig() *PostgresDBConfig {
	dbConfig := PostgresDBConfig{}
	envconfig.MustProcess("BOOKSTORE_DB", &dbConfig)
	return &dbConfig
}

func (c *PostgresDBConfig) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", c.User, c.Password, c.Host, c.Port, c.Name)
}

func NewDB(cfg *PostgresDBConfig) *sql.DB {
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to open db: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping db: %v", err)
	}

	return db
}

func Migrate(db *sql.DB, sourcePath string) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+sourcePath,
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to migrate: %v", err)
	}
}

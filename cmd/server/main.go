package main

import (
	"bookstore.example/store/internal/infra"
	"bookstore.example/store/internal/store"
)

func main() {
	db := infra.NewDB(infra.ParsePostgresDBConfig())
	infra.Migrate(db, "../../migrations/storedb/")
	secrets := infra.NewEnvSecrets()

	cfg := store.ParseServerConfig()
	server := store.NewServer(cfg, &secrets, db)

	server.Start()
}

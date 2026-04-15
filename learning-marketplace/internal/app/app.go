package app

import (
	"database/sql"

	"learning-marketplace/internal/analytics"
	"learning-marketplace/internal/config"
	"learning-marketplace/internal/coordination"
	"learning-marketplace/internal/store"
)

type App struct {
	Config      config.Config
	DB          *sql.DB
	ReaderDB    *sql.DB
	Store       *store.Store
	ReaderStore *store.Store
	UserShards  *store.UserShards
	LeaseStore  *coordination.LeaseStore
	Reporter    *analytics.Reporter
}

func New(cfg config.Config, db *sql.DB, readerDB *sql.DB, userShardStores []store.NamedStore, leaseStore *coordination.LeaseStore) *App {
	primaryStore := store.New(db)
	application := &App{Config: cfg, DB: db, ReaderDB: readerDB, Store: primaryStore, Reporter: analytics.NewReporter(db)}
	if readerDB != nil {
		application.ReaderStore = store.New(readerDB)
	}
	if len(userShardStores) == 0 {
		userShardStores = []store.NamedStore{{Name: "primary", Store: primaryStore}}
	}
	application.UserShards = store.NewUserShards(cfg.PostgresShards.VirtualShards, userShardStores)
	application.LeaseStore = leaseStore

	return application
}

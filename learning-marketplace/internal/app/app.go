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
	LeaseStore  *coordination.LeaseStore
	Reporter    *analytics.Reporter
}

func New(cfg config.Config, db *sql.DB, readerDB *sql.DB, leaseStore *coordination.LeaseStore) *App {
	application := &App{Config: cfg, DB: db, ReaderDB: readerDB, Store: store.New(db), Reporter: analytics.NewReporter(db)}
	if readerDB != nil {
		application.ReaderStore = store.New(readerDB)
	}
	application.LeaseStore = leaseStore

	return application
}

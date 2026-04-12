package db

import "embed"

// Migrations contains all SQL migrations shipped with the service binary.
//
//go:embed migrations/*.sql
var Migrations embed.FS

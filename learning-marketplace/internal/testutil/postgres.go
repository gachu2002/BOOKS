package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	appconfig "learning-marketplace/internal/config"
	apppostgres "learning-marketplace/internal/postgres"
)

// TestDatabase is a disposable migrated database for integration tests.
type TestDatabase struct {
	DB      *sql.DB
	adminDB *sql.DB
	name    string
}

type containerInfo struct {
	host     string
	port     string
	user     string
	password string
}

var (
	postgresContainerOnce sync.Once
	postgresContainerInfo containerInfo
	postgresContainerErr  error
	postgresContainerSkip string
)

// NewMigratedPostgres creates a throwaway database and applies all migrations.
func NewMigratedPostgres(t *testing.T, namePrefix string) *TestDatabase {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is required for integration tests")
	}

	info := startPostgresContainer(t)
	dbName := fmt.Sprintf("%s_%d", namePrefix, time.Now().UnixNano())

	adminCfg := appconfig.PostgresConfig{
		Host:     info.host,
		Port:     info.port,
		DB:       "postgres",
		User:     info.user,
		Password: info.password,
		SSLMode:  "disable",
	}

	adminDB, err := sql.Open("pgx", adminCfg.DSN())
	require.NoError(t, err)
	require.NoError(t, adminDB.PingContext(context.Background()))

	_, err = adminDB.ExecContext(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbName))
	require.NoError(t, err)

	appCfg := adminCfg
	appCfg.DB = dbName
	appDB, err := sql.Open("pgx", appCfg.DSN())
	require.NoError(t, err)
	require.NoError(t, appDB.PingContext(context.Background()))
	require.NoError(t, apppostgres.Migrate(context.Background(), appDB))

	testDB := &TestDatabase{DB: appDB, adminDB: adminDB, name: dbName}
	t.Cleanup(func() {
		require.NoError(t, appDB.Close())
		_, err := adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", dbName))
		require.NoError(t, err)
		require.NoError(t, adminDB.Close())
	})

	return testDB
}

// LooksLikeMissingDockerProvider lets integration tests skip cleanly outside Docker-enabled environments.
func LooksLikeMissingDockerProvider(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "failed to create Docker provider") ||
		strings.Contains(message, "Cannot connect to the Docker daemon") ||
		strings.Contains(message, "docker socket")
}

func startPostgresContainer(t *testing.T) containerInfo {
	t.Helper()

	postgresContainerOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		container, err := tcpostgres.Run(ctx,
			"postgres:16-alpine",
			tcpostgres.WithDatabase("postgres"),
			tcpostgres.WithUsername("postgres"),
			tcpostgres.WithPassword("postgres"),
		)
		if err != nil {
			if LooksLikeMissingDockerProvider(err) {
				postgresContainerSkip = err.Error()
				return
			}
			postgresContainerErr = err
			return
		}

		host, err := container.Host(ctx)
		if err != nil {
			postgresContainerErr = err
			return
		}

		mappedPort, err := container.MappedPort(ctx, "5432/tcp")
		if err != nil {
			postgresContainerErr = err
			return
		}

		postgresContainerInfo = containerInfo{host: host, port: mappedPort.Port(), user: "postgres", password: "postgres"}
	})

	if postgresContainerSkip != "" {
		t.Skipf("docker provider unavailable for integration tests: %s", postgresContainerSkip)
	}
	if postgresContainerErr != nil {
		t.Fatalf("start postgres container: %v", postgresContainerErr)
	}

	return postgresContainerInfo
}

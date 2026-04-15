package store_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/testutil"
)

func TestReadCommitted_AllowsReadSkew(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_test")
	ctx := context.Background()

	createAccountsFixture(t, testDB.DB)

	tx1, err := testDB.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	require.NoError(t, err)
	defer func() { _ = tx1.Rollback() }()

	accountOneBefore := readBalance(t, tx1, 1)

	tx2, err := testDB.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	require.NoError(t, err)
	require.NoError(t, transferBalance(ctx, tx2, 1, 2, 100))
	require.NoError(t, tx2.Commit())

	accountTwoAfter := readBalance(t, tx1, 2)
	require.NoError(t, tx1.Commit())

	require.Equal(t, 500, accountOneBefore)
	require.Equal(t, 400, accountTwoAfter)
	require.Equal(t, 900, accountOneBefore+accountTwoAfter)
	require.Equal(t, 1000, currentTotalBalance(t, testDB.DB))
}

func TestRepeatableRead_KeepsConsistentSnapshot(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_test")
	ctx := context.Background()

	createAccountsFixture(t, testDB.DB)

	tx1, err := testDB.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	require.NoError(t, err)
	defer func() { _ = tx1.Rollback() }()

	accountOneBefore := readBalance(t, tx1, 1)

	tx2, err := testDB.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	require.NoError(t, err)
	require.NoError(t, transferBalance(ctx, tx2, 1, 2, 100))
	require.NoError(t, tx2.Commit())

	accountTwoAfter := readBalance(t, tx1, 2)
	require.NoError(t, tx1.Commit())

	require.Equal(t, 500, accountOneBefore)
	require.Equal(t, 500, accountTwoAfter)
	require.Equal(t, 1000, accountOneBefore+accountTwoAfter)
	require.Equal(t, 1000, currentTotalBalance(t, testDB.DB))
}

func TestReadCommitted_AllowsLostUpdate(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_test")
	ctx := context.Background()

	createCounterFixture(t, testDB.DB)

	tx1, err := testDB.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	require.NoError(t, err)
	defer func() { _ = tx1.Rollback() }()

	tx2, err := testDB.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()

	firstRead := readCounter(t, tx1)
	secondRead := readCounter(t, tx2)

	require.Equal(t, 42, firstRead)
	require.Equal(t, 42, secondRead)

	writeCounter(t, tx1, firstRead+1)
	require.NoError(t, tx1.Commit())

	writeCounter(t, tx2, secondRead+1)
	require.NoError(t, tx2.Commit())

	require.Equal(t, 43, currentCounter(t, testDB.DB))
}

func TestSerializable_PreventsLostUpdate(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_test")
	ctx := context.Background()

	createCounterFixture(t, testDB.DB)

	tx1, err := testDB.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	require.NoError(t, err)
	defer func() { _ = tx1.Rollback() }()

	tx2, err := testDB.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()

	firstRead := readCounter(t, tx1)
	secondRead := readCounter(t, tx2)

	require.Equal(t, 42, firstRead)
	require.Equal(t, 42, secondRead)

	writeCounter(t, tx1, firstRead+1)
	require.NoError(t, tx1.Commit())

	_, err = tx2.ExecContext(ctx, `UPDATE chapter7_counters SET value = $1 WHERE id = 1`, secondRead+1)
	if err == nil {
		err = tx2.Commit()
	}
	require.Error(t, err)

	require.Equal(t, 43, currentCounter(t, testDB.DB))
}

func createAccountsFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `
CREATE TABLE chapter7_accounts (
    id INTEGER PRIMARY KEY,
    balance INTEGER NOT NULL
);
INSERT INTO chapter7_accounts (id, balance) VALUES (1, 500), (2, 500);`)
	require.NoError(t, err)
}

func createCounterFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `
CREATE TABLE chapter7_counters (
    id INTEGER PRIMARY KEY,
    value INTEGER NOT NULL
);
INSERT INTO chapter7_counters (id, value) VALUES (1, 42);`)
	require.NoError(t, err)
}

func readBalance(t *testing.T, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, accountID int) int {
	t.Helper()

	var balance int
	require.NoError(t, q.QueryRowContext(context.Background(), `SELECT balance FROM chapter7_accounts WHERE id = $1`, accountID).Scan(&balance))
	return balance
}

func currentTotalBalance(t *testing.T, db *sql.DB) int {
	t.Helper()

	var total int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT SUM(balance) FROM chapter7_accounts`).Scan(&total))
	return total
}

func transferBalance(ctx context.Context, tx *sql.Tx, fromID, toID, amount int) error {
	if _, err := tx.ExecContext(ctx, `UPDATE chapter7_accounts SET balance = balance - $1 WHERE id = $2`, amount, fromID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE chapter7_accounts SET balance = balance + $1 WHERE id = $2`, amount, toID); err != nil {
		return err
	}
	return nil
}

func readCounter(t *testing.T, tx *sql.Tx) int {
	t.Helper()

	var value int
	require.NoError(t, tx.QueryRowContext(context.Background(), `SELECT value FROM chapter7_counters WHERE id = 1`).Scan(&value))
	return value
}

func writeCounter(t *testing.T, tx *sql.Tx, value int) {
	t.Helper()

	_, err := tx.ExecContext(context.Background(), `UPDATE chapter7_counters SET value = $1 WHERE id = 1`, value)
	require.NoError(t, err)
}

func currentCounter(t *testing.T, db *sql.DB) int {
	t.Helper()

	var value int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT value FROM chapter7_counters WHERE id = 1`).Scan(&value))
	return value
}

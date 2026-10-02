package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSafeLocalURL(t *testing.T) {
	assert.True(t, isSafeLocalURL("postgres://user:pass@localhost:5432/db"))
	assert.True(t, isSafeLocalURL("postgres://user:pass@127.0.0.1:5432/db"))
	assert.True(t, isSafeLocalURL("postgres://user:pass@host.docker.internal:5432/db"))
	assert.False(t, isSafeLocalURL("postgres://user:pass@db.supabase.co:5432/db"))
}

func TestFindEnvFileAndGetEnvVar(t *testing.T) {
	t.Setenv("TEST_SHADOW_KEY", "direct_value")
	assert.Equal(t, "direct_value", getEnvVar("", "TEST_SHADOW_KEY"))

	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte("TEST_FILE_KEY=\"quoted_value\"\n# comment\nOTHER_KEY=123\n"), 0644)
	require.NoError(t, err)

	assert.Equal(t, "quoted_value", getEnvVar(envPath, "TEST_FILE_KEY"))
	assert.Equal(t, "123", getEnvVar(envPath, "OTHER_KEY"))
	assert.Equal(t, "", getEnvVar(envPath, "NON_EXISTENT"))
	assert.Equal(t, "", getEnvVar("/non/existent/path/.env", "ANY"))
	assert.Equal(t, "", getEnvVar("", "ANY"))

	// Test scanner error with a token longer than 64K
	longEnvPath := filepath.Join(tmpDir, "long.env")
	longLine := "LONG_KEY=" + strings.Repeat("A", 70000) + "\n"
	os.WriteFile(longEnvPath, []byte(longLine), 0644)
	assert.Equal(t, "", getEnvVar(longEnvPath, "LONG_KEY"))

	_ = findEnvFile()
}

func TestGetCommonColumns(t *testing.T) {
	dbProd, mockProd, err := sqlmock.New()
	require.NoError(t, err)
	defer dbProd.Close()

	dbLocal, mockLocal, err := sqlmock.New()
	require.NoError(t, err)
	defer dbLocal.Close()

	mockProd.ExpectQuery("SELECT column_name").
		WithArgs("users").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).
			AddRow("uuid").
			AddRow("email").
			AddRow("supabase_extra"))

	mockLocal.ExpectQuery("SELECT column_name").
		WithArgs("users").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).
			AddRow("uuid").
			AddRow("email").
			AddRow("local_extra"))

	cols, err := getCommonColumns(dbProd, dbLocal, "users")
	require.NoError(t, err)
	assert.Equal(t, []string{"uuid", "email"}, cols)

	// Test prod error
	mockProd.ExpectQuery("SELECT column_name").
		WithArgs("users").
		WillReturnError(errors.New("db error"))
	_, err = getCommonColumns(dbProd, dbLocal, "users")
	assert.Error(t, err)

	// Test prod row iteration error
	mockProd.ExpectQuery("SELECT column_name").
		WithArgs("users").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("uuid").RowError(0, errors.New("row err")))
	_, err = getCommonColumns(dbProd, dbLocal, "users")
	assert.Error(t, err)

	// Test local error
	mockProd.ExpectQuery("SELECT column_name").
		WithArgs("users").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("uuid"))
	mockLocal.ExpectQuery("SELECT column_name").
		WithArgs("users").
		WillReturnError(errors.New("local error"))
	_, err = getCommonColumns(dbProd, dbLocal, "users")
	assert.Error(t, err)

	// Test local row iteration error
	mockProd.ExpectQuery("SELECT column_name").
		WithArgs("users").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("uuid"))
	mockLocal.ExpectQuery("SELECT column_name").
		WithArgs("users").
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("uuid").RowError(0, errors.New("local row err")))
	_, err = getCommonColumns(dbProd, dbLocal, "users")
	assert.Error(t, err)
}

func TestRebaseDatabases_DryRun(t *testing.T) {
	dbProd, mockProd, err := sqlmock.New()
	require.NoError(t, err)
	defer dbProd.Close()

	dbLocal, mockLocal, err := sqlmock.New()
	require.NoError(t, err)
	defer dbLocal.Close()

	tables := []string{
		"users", "user_tokens", "envelope_group", "envelope",
		"subscriptions", "shortcut_intent", "wishlist_items",
		"transactionrows", "allocation", "wishlist_allocations",
	}

	for _, tbl := range tables {
		mockProd.ExpectQuery("SELECT column_name").
			WithArgs(tbl).
			WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))

		mockLocal.ExpectQuery("SELECT column_name").
			WithArgs(tbl).
			WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))

		if tbl == "users" {
			mockProd.ExpectQuery("SELECT \"id\" FROM \"users\"").
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("test-uuid"))
		} else {
			mockProd.ExpectQuery("SELECT \"id\" FROM \"" + tbl + "\"").
				WillReturnRows(sqlmock.NewRows([]string{"id"}))
		}
	}

	for _, tbl := range tables {
		mockProd.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \"" + tbl + "\"").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mockLocal.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \"" + tbl + "\"").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	}

	err = RebaseDatabases(dbProd, dbLocal, true)
	assert.NoError(t, err)
}

func TestRebaseDatabases_Success(t *testing.T) {
	dbProd, mockProd, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer dbProd.Close()

	dbLocal, mockLocal, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer dbLocal.Close()

	mockLocal.ExpectExec("SET session_replication_role = 'replica';").
		WillReturnResult(sqlmock.NewResult(0, 0))

	tables := []string{
		"users", "user_tokens", "envelope_group", "envelope",
		"subscriptions", "shortcut_intent", "wishlist_items",
		"transactionrows", "allocation", "wishlist_allocations",
	}
	for i := len(tables) - 1; i >= 0; i-- {
		mockLocal.ExpectExec("TRUNCATE TABLE \"" + tables[i] + "\" CASCADE;").
			WillReturnResult(sqlmock.NewResult(0, 0))
	}

	colQuery := "\n\t\tSELECT column_name \n\t\tFROM information_schema.columns \n\t\tWHERE table_schema = 'public' AND table_name = $1 \n\t\tORDER BY ordinal_position;\n\t"
	for _, tbl := range tables {
		if tbl == "users" {
			mockProd.ExpectQuery(colQuery).
				WithArgs(tbl).
				WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("uuid").AddRow("email"))

			mockLocal.ExpectQuery(colQuery).
				WithArgs(tbl).
				WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("uuid").AddRow("email"))

			mockProd.ExpectQuery("SELECT \"uuid\", \"email\" FROM \"users\"").
				WillReturnRows(sqlmock.NewRows([]string{"uuid", "email"}).AddRow("uuid-123", ""))

			mockLocal.ExpectPrepare("INSERT INTO \"users\" (\"uuid\", \"email\") VALUES ($1, $2)").
				ExpectExec().
				WithArgs("uuid-123", nil).
				WillReturnResult(sqlmock.NewResult(1, 1))
		} else if tbl == "wishlist_allocations" {
			mockProd.ExpectQuery(colQuery).
				WithArgs(tbl).
				WillReturnRows(sqlmock.NewRows([]string{"column_name"}))
			mockLocal.ExpectQuery(colQuery).
				WithArgs(tbl).
				WillReturnRows(sqlmock.NewRows([]string{"column_name"}))
		} else {
			mockProd.ExpectQuery(colQuery).
				WithArgs(tbl).
				WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))

			mockLocal.ExpectQuery(colQuery).
				WithArgs(tbl).
				WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))

			mockProd.ExpectQuery("SELECT \"id\" FROM \"" + tbl + "\"").
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("uuid-123"))

			mockLocal.ExpectPrepare("INSERT INTO \"" + tbl + "\" (\"id\") VALUES ($1)").
				ExpectExec().
				WithArgs("uuid-123").
				WillReturnResult(sqlmock.NewResult(1, 1))
		}
	}

	mockLocal.ExpectQuery("SELECT uuid FROM users WHERE email IS NOT NULL AND email != '' ORDER BY created_at ASC LIMIT 1").
		WillReturnRows(sqlmock.NewRows([]string{"uuid"}))
	mockLocal.ExpectQuery("SELECT uuid FROM users ORDER BY created_at ASC LIMIT 1").
		WillReturnRows(sqlmock.NewRows([]string{"uuid"}).AddRow("user-fallback-1"))

	mockLocal.ExpectExec(`
				INSERT INTO user_tokens (user_id, token_uuid, prefix, name, scopes, expires_at, last_used_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, NULL, NULL, $6, $7)
				ON CONFLICT (token_uuid) DO NOTHING;
			`).
		WithArgs("user-fallback-1", DevTokenAlias, "dev_token", "local_shadow_alias", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	for _, tbl := range tables {
		if tbl == "user_tokens" {
			mockProd.ExpectQuery("SELECT COUNT(*) FROM \"user_tokens\"").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
			mockLocal.ExpectQuery("SELECT COUNT(*) FROM \"user_tokens\"").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
		} else if tbl == "wishlist_items" {
			mockProd.ExpectQuery("SELECT COUNT(*) FROM \"wishlist_items\"").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mockLocal.ExpectQuery("SELECT COUNT(*) FROM \"wishlist_items\"").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		} else {
			mockProd.ExpectQuery("SELECT COUNT(*) FROM \"" + tbl + "\"").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mockLocal.ExpectQuery("SELECT COUNT(*) FROM \"" + tbl + "\"").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		}
	}

	mockLocal.ExpectExec("SET session_replication_role = 'origin';").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = RebaseDatabases(dbProd, dbLocal, false)
	assert.NoError(t, err)
}

func TestRebaseDatabases_Errors(t *testing.T) {
	// Replica error
	dbProd, _, _ := sqlmock.New()
	dbLocal, mockLocal, _ := sqlmock.New()
	mockLocal.ExpectExec("SET session_replication_role = 'replica';").WillReturnError(errors.New("replica err"))
	err := RebaseDatabases(dbProd, dbLocal, false)
	assert.Error(t, err)

	// Truncate error
	dbLocal, mockLocal, _ = sqlmock.New()
	mockLocal.ExpectExec("SET session_replication_role = 'replica';").WillReturnResult(sqlmock.NewResult(0, 0))
	mockLocal.ExpectExec("TRUNCATE TABLE").WillReturnError(errors.New("truncate err"))
	err = RebaseDatabases(dbProd, dbLocal, false)
	assert.Error(t, err)

	// Column resolution error
	dbProd, mockProd, _ := sqlmock.New()
	dbLocal, mockLocal, _ = sqlmock.New()
	mockLocal.ExpectExec("SET session_replication_role = 'replica';").WillReturnResult(sqlmock.NewResult(0, 0))
	for i := 0; i < 10; i++ {
		mockLocal.ExpectExec("TRUNCATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mockProd.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnError(errors.New("col err"))
	err = RebaseDatabases(dbProd, dbLocal, false)
	assert.Error(t, err)

	// Query prod error
	dbProd, mockProd, _ = sqlmock.New()
	dbLocal, mockLocal, _ = sqlmock.New()
	mockProd.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))
	mockLocal.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))
	mockProd.ExpectQuery("SELECT \"id\" FROM \"users\"").WillReturnError(errors.New("prod query err"))
	err = RebaseDatabases(dbProd, dbLocal, true)
	assert.Error(t, err)

	// Prepare error
	dbProd, mockProd, _ = sqlmock.New()
	dbLocal, mockLocal, _ = sqlmock.New()
	mockLocal.ExpectExec("SET session_replication_role = 'replica';").WillReturnResult(sqlmock.NewResult(0, 0))
	for i := 0; i < 10; i++ {
		mockLocal.ExpectExec("TRUNCATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mockProd.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))
	mockLocal.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))
	mockProd.ExpectQuery("SELECT \"id\" FROM \"users\"").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mockLocal.ExpectPrepare("INSERT INTO \"users\"").WillReturnError(errors.New("prepare err"))
	err = RebaseDatabases(dbProd, dbLocal, false)
	assert.Error(t, err)

	// Insert exec error
	dbProd, mockProd, _ = sqlmock.New()
	dbLocal, mockLocal, _ = sqlmock.New()
	mockLocal.ExpectExec("SET session_replication_role = 'replica';").WillReturnResult(sqlmock.NewResult(0, 0))
	for i := 0; i < 10; i++ {
		mockLocal.ExpectExec("TRUNCATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mockProd.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))
	mockLocal.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))
	mockProd.ExpectQuery("SELECT \"id\" FROM \"users\"").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("val1"))
	mockLocal.ExpectPrepare("INSERT INTO \"users\"").ExpectExec().WithArgs("val1").WillReturnError(errors.New("insert err"))
	err = RebaseDatabases(dbProd, dbLocal, false)
	assert.Error(t, err)

	// Rows iteration error
	dbProd, mockProd, _ = sqlmock.New()
	dbLocal, mockLocal, _ = sqlmock.New()
	mockProd.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))
	mockLocal.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))
	rowsWithErr := sqlmock.NewRows([]string{"id"}).AddRow("val").RowError(0, errors.New("row iteration err"))
	mockProd.ExpectQuery("SELECT \"id\" FROM \"users\"").WillReturnRows(rowsWithErr)
	err = RebaseDatabases(dbProd, dbLocal, true)
	assert.Error(t, err)

	// Scan error
	dbProd, mockProd, _ = sqlmock.New()
	dbLocal, mockLocal, _ = sqlmock.New()
	mockProd.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))
	mockLocal.ExpectQuery("SELECT column_name").WithArgs("users").WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id"))
	mockProd.ExpectQuery("SELECT \"id\" FROM \"users\"").WillReturnRows(sqlmock.NewRows([]string{"id", "extra"}).AddRow("1", "2"))
	err = RebaseDatabases(dbProd, dbLocal, true)
	assert.Error(t, err)
}

func TestRunCLI(t *testing.T) {
	// Flag parse error
	err := RunCLI([]string{"--unknown-flag"})
	assert.Error(t, err)

	tmpDir := t.TempDir()
	emptyEnv := filepath.Join(tmpDir, "empty.env")
	os.WriteFile(emptyEnv, []byte(""), 0644)
	err = RunCLI([]string{"-env", emptyEnv})
	assert.Error(t, err)

	prodOnlyEnv := filepath.Join(tmpDir, "prod_only.env")
	os.WriteFile(prodOnlyEnv, []byte("PROD_DATABASE_URL=postgres://user:pass@prod.db/db\n"), 0644)
	err = RunCLI([]string{"-env", prodOnlyEnv})
	assert.Error(t, err)

	unsafeEnv := filepath.Join(tmpDir, "unsafe.env")
	os.WriteFile(unsafeEnv, []byte("PROD_DATABASE_URL=postgres://user:pass@prod.db/db\nDATABASE_URL=postgres://user:pass@remote.db/db\n"), 0644)
	err = RunCLI([]string{"-env", unsafeEnv})
	assert.Error(t, err)

	// Mock DB openers
	oldOpener := dbOpener
	defer func() { dbOpener = oldOpener }()

	safeEnv := filepath.Join(tmpDir, "safe.env")
	os.WriteFile(safeEnv, []byte("PROD_DATABASE_URL=postgres://user:pass@localhost:5432/prod\nDATABASE_URL=postgres://user:pass@localhost:5432/local\n"), 0644)

	// Opener error on prod
	dbOpener = func(driverName, dataSourceName string) (*sql.DB, error) {
		return nil, errors.New("open error")
	}
	err = RunCLI([]string{"-env", safeEnv})
	assert.Error(t, err)

	// Opener error on local
	dbProd, _, _ := sqlmock.New()
	dbOpener = func(driverName, dataSourceName string) (*sql.DB, error) {
		if dataSourceName == "postgres://user:pass@localhost:5432/prod" {
			return dbProd, nil
		}
		return nil, errors.New("local open error")
	}
	err = RunCLI([]string{"-env", safeEnv})
	assert.Error(t, err)

	// Local ping error
	dbProd, _, _ = sqlmock.New()
	dbLocal, _, _ := sqlmock.New()
	dbLocal.Close()
	dbOpener = func(driverName, dataSourceName string) (*sql.DB, error) {
		if dataSourceName == "postgres://user:pass@localhost:5432/prod" {
			return dbProd, nil
		}
		return dbLocal, nil
	}
	err = RunCLI([]string{"-env", safeEnv})
	assert.Error(t, err)

	// Success run via RunCLI with dry-run
	dbProd, mockProd, _ := sqlmock.New()
	dbLocal, mockLocal, _ := sqlmock.New()
	dbOpener = func(driverName, dataSourceName string) (*sql.DB, error) {
		if dataSourceName == "postgres://user:pass@localhost:5432/prod" {
			return dbProd, nil
		}
		return dbLocal, nil
	}

	tables := []string{
		"users", "user_tokens", "envelope_group", "envelope",
		"subscriptions", "shortcut_intent", "wishlist_items",
		"transactionrows", "allocation", "wishlist_allocations",
	}
	for _, tbl := range tables {
		mockProd.ExpectQuery("SELECT column_name").WithArgs(tbl).WillReturnRows(sqlmock.NewRows([]string{"column_name"}))
		mockLocal.ExpectQuery("SELECT column_name").WithArgs(tbl).WillReturnRows(sqlmock.NewRows([]string{"column_name"}))
	}
	for _, tbl := range tables {
		mockProd.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \"" + tbl + "\"").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mockLocal.ExpectQuery("SELECT COUNT\\(\\*\\) FROM \"" + tbl + "\"").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	}

	err = RunCLI([]string{"-env", safeEnv, "-dry-run", "-force"})
	assert.NoError(t, err)
}

func TestMainFunction(t *testing.T) {
	oldExit := exitFunc
	defer func() { exitFunc = oldExit }()

	called := false
	exitFunc = func(format string, v ...any) {
		called = true
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"dbshadow", "--invalid-flag"}

	main()
	assert.True(t, called)
}

func TestDefaultOpener(t *testing.T) {
	_, err := dbOpener("postgres", "invalid-dsn")
	_ = err
}

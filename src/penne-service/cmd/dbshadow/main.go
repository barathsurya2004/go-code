package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"
)

// DevTokenAlias is the token configured in the local dashboard's LoginPage for quick access
const DevTokenAlias = "f66dcebd-e275-4b22-83bd-e446e0a45624"

func findEnvFile() string {
	candidates := []string{
		".env",
		"src/penne-service/.env",
		"../penne-service/.env",
		"/home/barath/Codes/penne-server/src/penne-service/.env",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func getEnvVar(envFile, key string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	if envFile == "" {
		return ""
	}
	f, err := os.Open(envFile)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, key+"=") {
			val := strings.TrimSpace(line[len(key)+1:])
			return strings.Trim(val, "\"")
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("scanner error: %v", err)
	}
	return ""
}

func getCommonColumns(dbProd, dbLocal *sql.DB, tableName string) ([]string, error) {
	query := `
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_schema = 'public' AND table_name = $1 
		ORDER BY ordinal_position;
	`
	prodRows, err := dbProd.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("prod columns query error: %w", err)
	}
	defer prodRows.Close()

	prodCols := make(map[string]bool)
	for prodRows.Next() {
		var col string
		if err := prodRows.Scan(&col); err == nil {
			prodCols[col] = true
		}
	}
	if err := prodRows.Err(); err != nil {
		return nil, fmt.Errorf("prodRows iteration error: %w", err)
	}

	localRows, err := dbLocal.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("local columns query error: %w", err)
	}
	defer localRows.Close()

	var common []string
	for localRows.Next() {
		var col string
		if err := localRows.Scan(&col); err == nil {
			if prodCols[col] {
				common = append(common, col)
			}
		}
	}
	if err := localRows.Err(); err != nil {
		return nil, fmt.Errorf("localRows iteration error: %w", err)
	}

	return common, nil
}

func isSafeLocalURL(url string) bool {
	lower := strings.ToLower(url)
	return strings.Contains(lower, "localhost") ||
		strings.Contains(lower, "127.0.0.1") ||
		strings.Contains(lower, "host.docker.internal")
}

// RebaseDatabases performs the full data copy from dbProd to dbLocal
func RebaseDatabases(dbProd, dbLocal *sql.DB, dryRun bool) error {
	tables := []string{
		"users",
		"user_tokens",
		"envelope_group",
		"envelope",
		"subscriptions",
		"shortcut_intent",
		"wishlist_items",
		"transactionrows",
		"allocation",
		"wishlist_allocations",
	}

	if !dryRun {
		if _, err := dbLocal.Exec("SET session_replication_role = 'replica';"); err != nil {
			return fmt.Errorf("set replica mode error: %w", err)
		}
		defer func() {
			dbLocal.Exec("SET session_replication_role = 'origin';")
		}()

		fmt.Println("🧹 Truncating local tables...")
		for i := len(tables) - 1; i >= 0; i-- {
			t := tables[i]
			if _, err := dbLocal.Exec(fmt.Sprintf("TRUNCATE TABLE \"%s\" CASCADE;", t)); err != nil {
				return fmt.Errorf("truncate table %s error: %w", t, err)
			}
		}
	}

	fmt.Println("\n📥 Copying production tables to local shadow DB...")
	for _, t := range tables {
		cols, err := getCommonColumns(dbProd, dbLocal, t)
		if err != nil {
			return fmt.Errorf("column resolution error for %s: %w", t, err)
		}
		if len(cols) == 0 {
			continue
		}

		colList := "\"" + strings.Join(cols, "\", \"") + "\""
		placeholders := make([]string, len(cols))
		for i := range cols {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}

		selectSQL := fmt.Sprintf("SELECT %s FROM \"%s\"", colList, t)
		rows, err := dbProd.Query(selectSQL)
		if err != nil {
			return fmt.Errorf("query prod %s error: %w", t, err)
		}

		insertSQL := fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s)", t, colList, strings.Join(placeholders, ", "))
		var stmt *sql.Stmt
		if !dryRun {
			stmt, err = dbLocal.Prepare(insertSQL)
			if err != nil {
				rows.Close()
				return fmt.Errorf("prepare insert for %s error: %w", t, err)
			}
		}

		count := 0
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range cols {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				if stmt != nil {
					stmt.Close()
				}
				return fmt.Errorf("scan row from %s error: %w", t, err)
			}

			if t == "users" {
				for i, col := range cols {
					if col == "email" {
						if s, ok := vals[i].(string); ok && strings.TrimSpace(s) == "" {
							vals[i] = nil
						}
					}
				}
			}

			if !dryRun {
				if _, err := stmt.Exec(vals...); err != nil {
					rows.Close()
					stmt.Close()
					return fmt.Errorf("insert error on table %s: %w", t, err)
				}
			}
			count++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			if stmt != nil {
				stmt.Close()
			}
			return fmt.Errorf("rows iteration error on %s: %w", t, err)
		}
		rows.Close()
		if stmt != nil {
			stmt.Close()
		}
	}

	if !dryRun {
		var primaryUserUUID string
		dbLocal.QueryRow("SELECT uuid FROM users WHERE email IS NOT NULL AND email != '' ORDER BY created_at ASC LIMIT 1").Scan(&primaryUserUUID)
		if primaryUserUUID == "" {
			dbLocal.QueryRow("SELECT uuid FROM users ORDER BY created_at ASC LIMIT 1").Scan(&primaryUserUUID)
		}

		if primaryUserUUID != "" {
			_, err := dbLocal.Exec(`
				INSERT INTO user_tokens (user_id, token_uuid, prefix, name, scopes, expires_at, last_used_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, NULL, NULL, $6, $7)
				ON CONFLICT (token_uuid) DO NOTHING;
			`, primaryUserUUID, DevTokenAlias, "dev_token", "local_shadow_alias", pq.StringArray{"read", "write", "all"}, time.Now(), time.Now())
			if err == nil {
				fmt.Printf("🔑 Configured local dev token alias (%s) for user %s\n", DevTokenAlias[:8]+"...", primaryUserUUID)
			}
		}
	}

	// Verification table
	fmt.Println("\n=================================================================")
	fmt.Printf("%-24s | %-12s | %-12s | %s\n", "Table", "Prod Count", "Local Count", "Status")
	fmt.Println("-----------------------------------------------------------------")

	allSynced := true
	for _, t := range tables {
		var pCount, lCount int
		dbProd.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", t)).Scan(&pCount)
		dbLocal.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", t)).Scan(&lCount)

		status := "✅ MATCH"
		if t == "user_tokens" && lCount == pCount+1 {
			status = "✅ MATCH (+1 dev alias)"
		} else if pCount != lCount && !dryRun {
			status = "⚠️ MISMATCH"
			allSynced = false
		}
		fmt.Printf("%-24s | %-12d | %-12d | %s\n", t, pCount, lCount, status)
	}
	fmt.Println("=================================================================")

	if allSynced {
		fmt.Println("🎉 Local database successfully rebased with production data!")
	}
	return nil
}

var dbOpener = func(driverName, dataSourceName string) (*sql.DB, error) {
	return sql.Open(driverName, dataSourceName)
}

var exitFunc = func(format string, v ...any) {
	log.Fatalf(format, v...)
}

func RunCLI(args []string) error {
	fs := flag.NewFlagSet("dbshadow", flag.ContinueOnError)
	force := fs.Bool("force", false, "Force sync even if target DB does not appear to be localhost")
	dryRun := fs.Bool("dry-run", false, "Simulate sync without modifying local database")
	envPath := fs.String("env", "", "Path to custom .env file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	envFile := *envPath
	if envFile == "" {
		envFile = findEnvFile()
	}
	prodURL := getEnvVar(envFile, "PROD_DATABASE_URL")
	localURL := getEnvVar(envFile, "DATABASE_URL")

	if prodURL == "" {
		return fmt.Errorf("PROD_DATABASE_URL is not set in environment or .env file")
	}
	if localURL == "" {
		return fmt.Errorf("DATABASE_URL (local target) is not set in environment or .env file")
	}

	if !*force && !isSafeLocalURL(localURL) {
		return fmt.Errorf("safety abort: target DATABASE_URL (%s) does not look like a local instance; use --force to override", localURL)
	}

	dbProd, err := dbOpener("postgres", prodURL)
	if err != nil {
		return fmt.Errorf("failed to open PROD database: %w", err)
	}
	defer dbProd.Close()

	if err := dbProd.Ping(); err != nil {
		return fmt.Errorf("cannot reach PROD database: %w", err)
	}

	dbLocal, err := dbOpener("postgres", localURL)
	if err != nil {
		return fmt.Errorf("failed to open LOCAL database: %w", err)
	}
	defer dbLocal.Close()

	if err := dbLocal.Ping(); err != nil {
		return fmt.Errorf("cannot reach LOCAL database: %w", err)
	}

	return RebaseDatabases(dbProd, dbLocal, *dryRun)
}

func main() {
	if err := RunCLI(os.Args[1:]); err != nil {
		exitFunc("❌ Error: %v", err)
	}
}

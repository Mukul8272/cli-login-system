package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/example/containerized-cli-login/internal/config"
)

func Open(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)
	for i := 0; i < 30; i++ {
		if err := database.PingContext(ctx); err == nil {
			return database, nil
		}
		time.Sleep(time.Second)
	}
	database.Close()
	return nil, fmt.Errorf("could not connect to MySQL")
}

func RunMigrations(database *sql.DB) error {
	content, err := os.ReadFile("/app/migrations/001_init.sql")
	if err != nil {
		content, err = os.ReadFile("migrations/001_init.sql")
		if err != nil {
			return err
		}
	}
	for _, statement := range splitSQL(string(content)) {
		if statement == "" {
			continue
		}
		if _, err := database.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func splitSQL(s string) []string {
	// This migration contains only simple CREATE statements, so splitting on
	// semicolons is sufficient and keeps startup dependency-free.
	var statements []string
	for _, part := range stringsSplit(s, ";") {
		trimmed := trimSpace(part)
		if trimmed != "" {
			statements = append(statements, trimmed)
		}
	}
	return statements
}

func stringsSplit(s, sep string) []string {
	var result []string
	for {
		idx := indexString(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}
func indexString(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\n' || s[start] == '\r' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\r' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

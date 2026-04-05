// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASSWORD", "admin")
	name := getEnv("DB_NAME", "recurin")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, name)
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("DB connect failed:", err)
	}
	defer pool.Close()

	fmt.Println("=== ALL TABLES ===")
	rows, err := pool.Query(ctx, `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name`)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var schema, table string
		rows.Scan(&schema, &table)
		fmt.Printf("  %s.%s\n", schema, table)
	}
	rows.Close()

	// Now show the payments table columns
	fmt.Println("\n=== users.payments COLUMNS ===")
	rows2, err := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'users' AND table_name = 'payments'
		ORDER BY ordinal_position`)
	if err != nil {
		log.Fatal(err)
	}
	for rows2.Next() {
		var colName, dataType, nullable string
		var colDefault *string
		rows2.Scan(&colName, &dataType, &nullable, &colDefault)
		def := "NULL"
		if colDefault != nil {
			def = *colDefault
		}
		fmt.Printf("  %-25s %-20s nullable=%-5s default=%s\n", colName, dataType, nullable, def)
	}
	rows2.Close()

	// Show subscription.subscriptions columns
	fmt.Println("\n=== subscription.subscriptions COLUMNS ===")
	rows3, err := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'subscription' AND table_name = 'subscriptions'
		ORDER BY ordinal_position`)
	if err != nil {
		log.Fatal(err)
	}
	for rows3.Next() {
		var colName, dataType, nullable string
		var colDefault *string
		rows3.Scan(&colName, &dataType, &nullable, &colDefault)
		def := "NULL"
		if colDefault != nil {
			def = *colDefault
		}
		fmt.Printf("  %-25s %-20s nullable=%-5s default=%s\n", colName, dataType, nullable, def)
	}
	rows3.Close()

	fmt.Println("\nDone.")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

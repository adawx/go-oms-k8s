package main

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"time"

	"go-oms/shared/env"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

var (
	dbHost = env.GetEnv("DB_HOST", "localhost")
	dbPort = env.GetEnv("DB_PORT", "5432")
	dbUser = env.GetEnv("DB_USER", "app")
	dbPass = env.GetEnv("DB_PASSWORD", "")
	dbName = env.GetEnv("DB_NAME", "app")
)

func main() {
	log.Print("STARTING ORDER SERVICE")

	db := connect()
	defer db.Close()

	runMigrations(db)

	log.Print("Order service ready (migrations applied); idling")
	select {} //TODO: write a http server so we don't deadlock and crash loop
}

func connect() *sql.DB {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=require",
		dbUser, dbPass, dbHost, dbPort, dbName,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	for attempt := 1; attempt <= 10; attempt++ {
		if err = db.Ping(); err == nil {
			log.Printf("connected to postgres at %s:%s/%s", dbHost, dbPort, dbName)
			return db
		}
		log.Printf("waiting for postgres (attempt %d/10): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("could not connect to postgres after retries: %v", err)
	return nil
}

func runMigrations(db *sql.DB) {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("goose set dialect: %v", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatalf("goose up: %v", err)
	}

	log.Print("migrations applied")
}

package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"net/http"
	"time"

	"go-oms/shared/env"
	"go-oms/shared/httpserver"
	"go-oms/shared/metrics"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// serviceName labels every metric this service exports.
const serviceName = "order-service"

//go:embed migrations/*.sql
var migrations embed.FS

var (
	httpAddr  = env.GetEnv("HTTP_ADDR", ":8085")
	adminAddr = env.GetEnv("ADMIN_HTTP_ADDR", ":9090")

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

	m := metrics.New(serviceName)

	// Expose database/sql pool stats (open, idle, in-use, wait counts) so the
	// dashboard can show pool saturation next to request latency.
	//
	// The collector only labels its series with db_name, so it is wrapped to
	// carry the same `service` label as the RED metrics; without it the
	// dashboard's per-service filter would exclude these series entirely.
	prometheus.WrapRegistererWith(
		prometheus.Labels{"service": serviceName},
		m.Registry(),
	).MustRegister(collectors.NewDBStatsCollector(db, dbName))

	mux := http.NewServeMux()
	mux.Handle("/", m.Middleware("/", http.HandlerFunc(rootHandler)))

	srv := httpserver.New(httpAddr, adminAddr, mux, m, func() error {
		// Readiness follows the database: if Postgres is unreachable this pod
		// cannot serve orders, so it should be pulled out of the Service.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return db.PingContext(ctx)
	})

	httpserver.ExitOnError(srv.Run())
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello from Order Service"))
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

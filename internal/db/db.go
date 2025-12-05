package db

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Conn is the global database handle reused across the service.
var Conn *sql.DB

// InitPostgres initializes a PostgreSQL connection using the provided DSN.
func InitPostgres(dsn string) {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	Conn, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("DB connection error: %v", err)
	}

	if err = Conn.PingContext(ctx); err != nil {
		log.Fatalf("DB ping failed: %v", err)
	}

	log.Println("PostgreSQL connected")
}

// CloseDB closes the shared database connection.
func CloseDB() {
	if Conn == nil {
		return
	}
	if err := Conn.Close(); err != nil {
		log.Printf("Error closing DB: %v", err)
	}
}

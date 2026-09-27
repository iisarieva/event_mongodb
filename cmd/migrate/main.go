package main

import (
	"context"
	"event_mongodb/internal/database"
	"event_mongodb/internal/migrate"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("failed to load .env file")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("MONGODB_URI is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := database.NewMongoClient(ctx, uri)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("disconnect mongodb: %v", err)
		}
	}()

	db := client.Database("eventflow")
	if err := migrate.Up(ctx, db); err != nil {
		log.Fatal(err)
	}

	log.Println("migrations applied successfully")
}

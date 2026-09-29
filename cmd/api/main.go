package main

import (
	"context"
	"event_mongodb/internal/database"
	"event_mongodb/internal/event"
	"event_mongodb/internal/migrate"
	"event_mongodb/internal/server"
	"event_mongodb/internal/user"
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

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	client, err := database.NewMongoClient(ctx, uri)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf(
				"disconnect mongodb: %v",
				err,
			)
		}
	}()

	log.Println("MongoDB connected successfully")

	db := client.Database("eventflow")
	if err := migrate.Up(ctx, db); err != nil {
		log.Fatal(err)
	}
	log.Println("database migrations applied")

	eventsCollection := db.Collection("events")
	usersCollection := db.Collection("users")

	eventRepository := event.NewRepository(eventsCollection)
	userRepository := user.NewRepository(usersCollection)
	userHandler := user.NewHandler(userRepository)
	eventService := event.NewService(
		client,
		eventRepository,
		userRepository,
	)

	eventHandler := event.NewHandler(
		eventRepository,
		eventService,
		userRepository,
	)

	e := server.New(eventHandler, userHandler)

	if err := e.Start(":8080"); err != nil {
		log.Printf(
			"server stopped: %v",
			err,
		)
	}
}

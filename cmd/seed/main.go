package main

import (
	"context"
	"event_mongodb/internal/database"
	"event_mongodb/internal/event"
	"event_mongodb/internal/migrate"
	"event_mongodb/internal/user"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	usersCount  = 10_000
	eventsCount = 200_000

	userBatchSize  = 1_000
	eventBatchSize = 5_000
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("failed to load .env")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("MONGODB_URI is not set")
	}

	ctx := context.Background()

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

	usersCollection := db.Collection("users")
	eventsCollection := db.Collection("events")

	if _, err := usersCollection.DeleteMany(ctx, bson.M{}); err != nil {
		log.Fatal(err)
	}

	if _, err := eventsCollection.DeleteMany(ctx, bson.M{}); err != nil {
		log.Fatal(err)
	}

	if err := seedUsers(ctx, usersCollection); err != nil {
		log.Fatal(err)
	}

	if err := seedEvents(ctx, eventsCollection); err != nil {
		log.Fatal(err)
	}

	log.Println("Seed completed")
}

func seedUsers(
	ctx context.Context,
	collection *mongo.Collection,
) error {
	batch := make([]any, 0, userBatchSize)

	for i := 1; i <= usersCount; i++ {
		id := fmt.Sprintf("u%05d", i)

		country := []string{
			"UA",
			"DE",
			"PL",
			"FR",
		}[i%4]

		plan := []string{
			"free",
			"basic",
			"pro",
		}[i%3]

		tags := []string{"mobile"}

		if i%2 == 0 {
			tags = append(tags, "beta")
		}

		if i%5 == 0 {
			tags = append(tags, "vip")
		}

		u := user.User{
			ID:      id,
			Name:    fmt.Sprintf("User %d", i),
			Country: country,
			Plan:    plan,
			Tags:    tags,
		}

		batch = append(batch, u)

		if len(batch) == userBatchSize {
			if _, err := collection.InsertMany(ctx, batch); err != nil {
				return fmt.Errorf(
					"insert users batch: %w",
					err,
				)
			}

			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if _, err := collection.InsertMany(ctx, batch); err != nil {
			return fmt.Errorf(
				"insert last users batch: %w",
				err,
			)
		}
	}

	log.Printf("Inserted %d users", usersCount)

	return nil
}

func seedEvents(
	ctx context.Context,
	collection *mongo.Collection,
) error {
	batch := make([]any, 0, eventBatchSize)

	now := time.Now().UTC()

	for i := 1; i <= eventsCount; i++ {
		userNumber := (i % usersCount) + 1
		userID := fmt.Sprintf(
			"u%05d",
			userNumber,
		)

		country := []string{
			"UA",
			"DE",
			"PL",
			"FR",
		}[userNumber%4]

		plan := []string{
			"free",
			"basic",
			"pro",
		}[userNumber%3]

		timestamp := now.Add(
			-time.Duration(i%720) * time.Hour,
		)

		var eventType string
		var properties bson.M

		switch i % 3 {
		case 0:
			eventType = "user_registered"

			properties = bson.M{
				"country": country,
				"source": []string{
					"google",
					"direct",
					"facebook",
				}[i%3],
			}

		case 1:
			eventType = "message_sent"

			properties = bson.M{
				"chat_id": fmt.Sprintf(
					"chat_%d",
					i%5000,
				),
				"channel": []string{
					"telegram",
					"web",
					"mobile",
				}[i%3],
				"length": (i % 500) + 1,
			}

		case 2:
			eventType = "payment_completed"

			properties = bson.M{
				"amount": float64(
					(i % 500) + 1,
				),
				"currency": "USD",
				"plan":     plan,
			}
		}

		e := event.Event{
			EventType: eventType,
			UserID:    userID,

			UserSnapshot: &event.UserSnapshot{
				Country: country,
				Plan:    plan,
			},

			Timestamp:  timestamp,
			Properties: properties,
		}

		batch = append(batch, e)

		if len(batch) == eventBatchSize {
			if _, err := collection.InsertMany(
				ctx,
				batch,
			); err != nil {
				return fmt.Errorf(
					"insert events batch: %w",
					err,
				)
			}

			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if _, err := collection.InsertMany(
			ctx,
			batch,
		); err != nil {
			return fmt.Errorf(
				"insert last events batch: %w",
				err,
			)
		}
	}

	log.Printf(
		"Inserted %d events",
		eventsCount,
	)

	return nil
}

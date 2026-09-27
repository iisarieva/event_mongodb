package migrate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type migration struct {
	name string
	fn   func(context.Context, *mongo.Database) error
}

var migrations = []migration{
	{name: "001_create_events_collection", fn: createEventsCollection},
	{name: "002_events_json_schema", fn: eventsValidation},
	{name: "003_users_tags_index", fn: usersTagsIndex},
}

func Up(
	ctx context.Context,
	db *mongo.Database,
) error {
	applied := db.Collection("schema_migrations")

	for _, m := range migrations {
		var existing bson.M

		err := applied.
			FindOne(
				ctx,
				bson.M{"_id": m.name},
			).
			Decode(&existing)

		if err == nil {
			continue
		}

		if !errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf(
				"check migration %s: %w",
				m.name,
				err,
			)
		}

		if err := m.fn(ctx, db); err != nil {
			return fmt.Errorf(
				"migration %s: %w",
				m.name,
				err,
			)
		}

		_, err = applied.InsertOne(
			ctx,
			bson.M{
				"_id":        m.name,
				"applied_at": time.Now().UTC(),
			},
		)
		if err != nil {
			return fmt.Errorf(
				"record migration %s: %w",
				m.name,
				err,
			)
		}
	}

	return nil
}

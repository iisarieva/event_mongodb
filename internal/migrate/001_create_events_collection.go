package migrate

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func createEventsCollection(
	ctx context.Context,
	db *mongo.Database,
) error {
	names, err := db.ListCollectionNames(
		ctx,
		bson.M{"name": "events"},
	)
	if err != nil {
		return err
	}

	if len(names) > 0 {
		return nil
	}

	return db.CreateCollection(ctx, "events")
}

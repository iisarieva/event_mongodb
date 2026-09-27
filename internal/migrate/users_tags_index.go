package migrate

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func usersTagsIndex(
	ctx context.Context,
	db *mongo.Database,
) error {
	collection := db.Collection("users")

	model := mongo.IndexModel{
		Keys: bson.D{
			{Key: "tags", Value: 1},
		},
	}

	_, err := collection.
		Indexes().
		CreateOne(ctx, model)

	return err
}

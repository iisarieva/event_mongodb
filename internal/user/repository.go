package user

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var ErrNotFound = errors.New("user not found")

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(collection *mongo.Collection) *Repository {
	return &Repository{
		collection: collection,
	}
}

func (r *Repository) Create(
	ctx context.Context,
	user *User,
) error {
	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	id string,
) (*User, error) {
	var result User

	err := r.collection.
		FindOne(
			ctx,
			bson.M{
				"_id": id,
			},
		).
		Decode(&result)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf(
			"find user: %w",
			err,
		)
	}

	return &result, nil
}

func (r *Repository) Update(
	ctx context.Context,
	id string,
	fields bson.M,
) (*User, error) {
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id": id,
		},
		bson.M{
			"$set": fields,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"update user: %w",
			err,
		)
	}

	if result.MatchedCount == 0 {
		return nil, ErrNotFound
	}

	return r.GetByID(ctx, id)
}

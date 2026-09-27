package event

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrNotFound  = errors.New("event not found")
	ErrInvalidID = errors.New("invalid event id")
)

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
	event *Event,
) error {
	result, err := r.collection.InsertOne(ctx, event)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	id, ok := result.InsertedID.(bson.ObjectID)
	if ok {
		event.ID = id
	}

	return nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	id string,
) (*Event, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidID, err)
	}

	var result Event

	err = r.collection.
		FindOne(
			ctx,
			bson.M{"_id": objectID},
		).
		Decode(&result)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("find event: %w", err)
	}

	return &result, nil
}

func (r *Repository) List(
	ctx context.Context,
	params ListOptions,
) ([]Event, error) {
	filter := bson.M{}

	if params.UserID != "" {
		filter["user_id"] = params.UserID
	}

	if params.EventType != "" {
		filter["event_type"] = params.EventType
	}

	if params.From != nil || params.To != nil {
		dateFilter := bson.M{}

		if params.From != nil {
			dateFilter["$gte"] = *params.From
		}

		if params.To != nil {
			dateFilter["$lte"] = *params.To
		}

		filter["timestamp"] = dateFilter
	}

	opts := options.Find().
		SetSort(
			bson.D{
				{Key: "timestamp", Value: params.SortDirection},
			},
		).
		SetLimit(params.Limit)

	if params.ExcludeProperties {
		opts.SetProjection(
			bson.M{
				"properties": 0,
			},
		)
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find events: %w", err)
	}

	defer cursor.Close(ctx)

	var events []Event

	if err := cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}

	return events, nil
}

func (r *Repository) Update(
	ctx context.Context,
	id string,
	fields bson.M,
) (*Event, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidID, err)
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
		bson.M{
			"$set": fields,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, ErrNotFound
	}

	return r.GetByID(ctx, id)
}

func (r *Repository) Delete(
	ctx context.Context,
	id string,
) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidID, err)
	}

	result, err := r.collection.DeleteOne(
		ctx,
		bson.M{
			"_id": objectID,
		},
	)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	if result.DeletedCount == 0 {
		return ErrNotFound
	}

	return nil
}

func IsValidationError(err error) bool {

	if writeException, ok := errors.AsType[mongo.WriteException](err); ok {
		return writeException.HasErrorCode(121)
	}

	return false
}

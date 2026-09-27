package migrate

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func eventsValidation(
	ctx context.Context,
	db *mongo.Database,
) error {
	schema := bson.M{
		"bsonType": "object",

		"required": bson.A{
			"event_type",
			"user_id",
			"timestamp",
			"properties",
		},

		"properties": bson.M{
			"event_type": bson.M{
				"enum": bson.A{
					"user_registered",
					"message_sent",
					"payment_completed",
				},
			},

			"user_id": bson.M{
				"bsonType":  "string",
				"minLength": 1,
			},

			"timestamp": bson.M{
				"bsonType": "date",
			},

			"properties": bson.M{
				"bsonType": "object",
			},
		},

		"oneOf": bson.A{
			bson.M{
				"properties": bson.M{
					"event_type": bson.M{
						"enum": bson.A{
							"user_registered",
						},
					},
					"properties": bson.M{
						"bsonType": "object",
						"required": bson.A{
							"country",
							"source",
						},
						"properties": bson.M{
							"country": bson.M{
								"bsonType": "string",
							},
							"source": bson.M{
								"bsonType": "string",
							},
						},
					},
				},
			},

			bson.M{
				"properties": bson.M{
					"event_type": bson.M{
						"enum": bson.A{
							"message_sent",
						},
					},
					"properties": bson.M{
						"bsonType": "object",
						"required": bson.A{
							"chat_id",
							"channel",
							"length",
						},
						"properties": bson.M{
							"chat_id": bson.M{
								"bsonType": "string",
							},
							"channel": bson.M{
								"bsonType": "string",
							},
							"length": bson.M{
								"bsonType": "number",
								"minimum":  0,
							},
						},
					},
				},
			},

			bson.M{
				"properties": bson.M{
					"event_type": bson.M{
						"enum": bson.A{
							"payment_completed",
						},
					},
					"properties": bson.M{
						"bsonType": "object",
						"required": bson.A{
							"amount",
							"currency",
							"plan",
						},
						"properties": bson.M{
							"amount": bson.M{
								"bsonType": "number",
								"minimum":  0,
							},
							"currency": bson.M{
								"bsonType": "string",
							},
							"plan": bson.M{
								"bsonType": "string",
							},
						},
					},
				},
			},
		},
	}

	command := bson.D{
		{Key: "collMod", Value: "events"},
		{
			Key: "validator",
			Value: bson.M{
				"$jsonSchema": schema,
			},
		},
		{Key: "validationLevel", Value: "strict"},
		{Key: "validationAction", Value: "error"},
	}

	var result bson.M

	if err := db.RunCommand(ctx, command).Decode(&result); err != nil {
		return fmt.Errorf("apply events validation: %w", err)
	}

	return nil
}

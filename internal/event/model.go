package event

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Event struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	EventType    string        `bson:"event_type" json:"event_type"`
	UserID       string        `bson:"user_id" json:"user_id"`
	UserSnapshot *UserSnapshot `bson:"user_snapshot,omitempty" json:"user_snapshot,omitempty"`
	Timestamp    time.Time     `bson:"timestamp" json:"timestamp"`
	Properties   bson.M        `bson:"properties" json:"properties,omitempty"`
}

type CreateEventRequest struct {
	EventType  string     `json:"event_type"`
	UserID     string     `json:"user_id"`
	Timestamp  *time.Time `json:"timestamp,omitempty"`
	Properties bson.M     `json:"properties"`
}

type UpdateEventRequest struct {
	EventType  *string    `json:"event_type,omitempty"`
	UserID     *string    `json:"user_id,omitempty"`
	Timestamp  *time.Time `json:"timestamp,omitempty"`
	Properties bson.M     `json:"properties,omitempty"`
}

type ListOptions struct {
	UserID            string
	EventType         string
	From              *time.Time
	To                *time.Time
	Limit             int64
	SortDirection     int
	ExcludeProperties bool
}

type UserSnapshot struct {
	Country string `bson:"country" json:"country"`
	Plan    string `bson:"plan" json:"plan"`
}

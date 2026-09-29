package event

import (
	"context"
	"event_mongodb/internal/user"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Service struct {
	client          *mongo.Client
	eventRepository *Repository
	userRepository  *user.Repository
}

func NewService(
	client *mongo.Client,
	eventRepository *Repository,
	userRepository *user.Repository,
) *Service {
	return &Service{
		client:          client,
		eventRepository: eventRepository,
		userRepository:  userRepository,
	}
}

func (s *Service) Create(
	ctx context.Context,
	event *Event,
) error {
	session, err := s.client.StartSession()
	if err != nil {
		return fmt.Errorf("start mongo session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(
		ctx,
		func(txCtx context.Context) (any, error) {
			// 1. Создаём event.
			if err := s.eventRepository.Create(txCtx, event); err != nil {
				return nil, fmt.Errorf("create event: %w", err)
			}

			// 2. Увеличиваем events_count пользователя.
			if err := s.userRepository.IncrementEventsCount(
				txCtx,
				event.UserID,
			); err != nil {
				return nil, fmt.Errorf("increment events count: %w", err)
			}

			return nil, nil
		},
	)
	if err != nil {
		return fmt.Errorf("create event transaction: %w", err)
	}

	return nil
}

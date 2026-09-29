package event

import (
	"errors"
	"event_mongodb/internal/user"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Handler struct {
	repository     *Repository
	service        *Service
	userRepository *user.Repository
}

func NewHandler(
	repository *Repository,
	service *Service,
	userRepository *user.Repository,
) *Handler {
	return &Handler{
		repository:     repository,
		service:        service,
		userRepository: userRepository,
	}
}

func (h *Handler) Create(c *echo.Context) error {
	var request CreateEventRequest

	if err := c.Bind(&request); err != nil {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid request body",
			},
		)
	}

	if request.EventType == "" {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "event_type is required",
			},
		)
	}

	if request.UserID == "" {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "user_id is required",
			},
		)
	}

	existingUser, err := h.userRepository.GetByID(
		c.Request().Context(),
		request.UserID,
	)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return c.JSON(
				http.StatusBadRequest,
				map[string]string{
					"error": "user does not exist",
				},
			)
		}

		return err
	}

	timestamp := time.Now().UTC()

	if request.Timestamp != nil {
		timestamp = request.Timestamp.UTC()
	}

	event := Event{
		EventType: request.EventType,
		UserID:    request.UserID,
		UserSnapshot: &UserSnapshot{
			Country: existingUser.Country,
			Plan:    existingUser.Plan,
		},
		Timestamp:  timestamp,
		Properties: request.Properties,
	}

	if err := h.service.Create(
		c.Request().Context(),
		&event,
	); err != nil {
		if IsValidationError(err) {
			return c.JSON(
				http.StatusBadRequest,
				map[string]string{
					"error": "event does not match schema",
				},
			)
		}

		return err
	}

	return c.JSON(
		http.StatusCreated,
		event,
	)
}

func (h *Handler) GetByID(c *echo.Context) error {
	event, err := h.repository.GetByID(
		c.Request().Context(),
		c.Param("id"),
	)
	if err != nil {
		return handleRepositoryError(c, err)
	}

	return c.JSON(
		http.StatusOK,
		event,
	)
}

func (h *Handler) List(c *echo.Context) error {
	params := ListOptions{
		UserID:        c.QueryParam("user_id"),
		EventType:     c.QueryParam("event_type"),
		Limit:         50,
		SortDirection: -1,
	}

	if from := c.QueryParam("from"); from != "" {
		value, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return c.JSON(
				http.StatusBadRequest,
				map[string]string{
					"error": "invalid from date",
				},
			)
		}

		params.From = &value
	}

	if to := c.QueryParam("to"); to != "" {
		value, err := time.Parse(time.RFC3339, to)
		if err != nil {
			return c.JSON(
				http.StatusBadRequest,
				map[string]string{
					"error": "invalid to date",
				},
			)
		}

		params.To = &value
	}

	if limit := c.QueryParam("limit"); limit != "" {
		value, err := strconv.ParseInt(limit, 10, 64)
		if err != nil || value < 1 || value > 100 {
			return c.JSON(
				http.StatusBadRequest,
				map[string]string{
					"error": "limit must be between 1 and 100",
				},
			)
		}

		params.Limit = value
	}

	switch c.QueryParam("sort") {
	case "", "desc":
		params.SortDirection = -1

	case "asc":
		params.SortDirection = 1

	default:
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "sort must be asc or desc",
			},
		)
	}

	if exclude := c.QueryParam("exclude_properties"); exclude != "" {
		value, err := strconv.ParseBool(exclude)
		if err != nil {
			return c.JSON(
				http.StatusBadRequest,
				map[string]string{
					"error": "exclude_properties must be boolean",
				},
			)
		}

		params.ExcludeProperties = value
	}

	events, err := h.repository.List(
		c.Request().Context(),
		params,
	)
	if err != nil {
		return err
	}

	return c.JSON(
		http.StatusOK,
		events,
	)
}

func (h *Handler) Update(c *echo.Context) error {
	var request UpdateEventRequest

	if err := c.Bind(&request); err != nil {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid request body",
			},
		)
	}

	fields := bson.M{}

	if request.EventType != nil {
		fields["event_type"] = *request.EventType
	}

	if request.UserID != nil {
		fields["user_id"] = *request.UserID
	}

	if request.Timestamp != nil {
		fields["timestamp"] = request.Timestamp.UTC()
	}

	if request.Properties != nil {
		fields["properties"] = request.Properties
	}

	if len(fields) == 0 {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "nothing to update",
			},
		)
	}

	event, err := h.repository.Update(
		c.Request().Context(),
		c.Param("id"),
		fields,
	)
	if err != nil {
		return handleRepositoryError(c, err)
	}

	return c.JSON(
		http.StatusOK,
		event,
	)
}

func (h *Handler) Delete(c *echo.Context) error {
	err := h.repository.Delete(
		c.Request().Context(),
		c.Param("id"),
	)
	if err != nil {
		return handleRepositoryError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func handleRepositoryError(
	c *echo.Context,
	err error,
) error {
	switch {
	case errors.Is(err, ErrInvalidID):
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid event id",
			},
		)

	case errors.Is(err, ErrNotFound):
		return c.JSON(
			http.StatusNotFound,
			map[string]string{
				"error": "event not found",
			},
		)

	case IsValidationError(err):
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "event does not match schema",
			},
		)

	default:
		return err
	}
}

package user

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Handler struct {
	repository *Repository
}

func NewHandler(repository *Repository) *Handler {
	return &Handler{
		repository: repository,
	}
}

func (h *Handler) Create(c *echo.Context) error {
	var request CreateUserRequest

	if err := c.Bind(&request); err != nil {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid request body",
			},
		)
	}

	if request.ID == "" ||
		request.Name == "" ||
		request.Country == "" ||
		request.Plan == "" {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "id, name, country and plan are required",
			},
		)
	}

	user := User{
		ID:      request.ID,
		Name:    request.Name,
		Country: request.Country,
		Plan:    request.Plan,
	}

	if err := h.repository.Create(
		c.Request().Context(),
		&user,
	); err != nil {
		return err
	}

	return c.JSON(
		http.StatusCreated,
		user,
	)
}

func (h *Handler) GetByID(c *echo.Context) error {
	user, err := h.repository.GetByID(
		c.Request().Context(),
		c.Param("id"),
	)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(
				http.StatusNotFound,
				map[string]string{
					"error": "user not found",
				},
			)
		}

		return err
	}

	return c.JSON(
		http.StatusOK,
		user,
	)
}

func (h *Handler) Update(c *echo.Context) error {
	var request UpdateUserRequest

	if err := c.Bind(&request); err != nil {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid request body",
			},
		)
	}

	fields := bson.M{}

	if request.Name != nil {
		fields["name"] = *request.Name
	}

	if request.Country != nil {
		fields["country"] = *request.Country
	}

	if request.Plan != nil {
		fields["plan"] = *request.Plan
	}

	if len(fields) == 0 {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{
				"error": "nothing to update",
			},
		)
	}

	user, err := h.repository.Update(
		c.Request().Context(),
		c.Param("id"),
		fields,
	)

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.JSON(
				http.StatusNotFound,
				map[string]string{
					"error": "user not found",
				},
			)
		}

		return err
	}

	return c.JSON(
		http.StatusOK,
		user,
	)
}

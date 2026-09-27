package server

import (
	"event_mongodb/internal/event"
	"event_mongodb/internal/user"

	"github.com/labstack/echo/v5"
)

func New(
	eventHandler *event.Handler,
	userHandler *user.Handler,
) *echo.Echo {
	e := echo.New()

	e.POST("/events", eventHandler.Create)
	e.GET("/events", eventHandler.List)
	e.GET("/events/:id", eventHandler.GetByID)
	e.PATCH("/events/:id", eventHandler.Update)
	e.DELETE("/events/:id", eventHandler.Delete)

	e.POST("/users", userHandler.Create)
	e.GET("/users/:id", userHandler.GetByID)
	e.PATCH("/users/:id", userHandler.Update)
	e.GET("/users", userHandler.List)

	return e
}

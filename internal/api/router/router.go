package router

import (
	"github.com/labstack/echo/v4"
	"suratnesia/internal/api/handler"
)

func New() *echo.Echo {
	e := echo.New()
	e.GET("/health", handler.HealthCheck)
	return e
}

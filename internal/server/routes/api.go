package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// APIRoutes registers API endpoints.
type APIRoutes struct{}

// RegisterRoutes registers API endpoints.
func (a *APIRoutes) RegisterRoutes(s *echo.Echo) {
	api := s.Group("/api/v1")

	api.GET("/services", handleAPIGetServices)
	api.POST("/deployments", handleAPICreateDeployment)
}

func handleAPIGetServices(c echo.Context) error {
	// Replace with DB logic.
	return c.JSON(http.StatusOK, []string{"service1", "service2"})
}

func handleAPICreateDeployment(c echo.Context) error {
	// Implementation for creating a deployment via API.
	return c.NoContent(http.StatusCreated)
}

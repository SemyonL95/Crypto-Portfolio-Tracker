package server

import (
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// registerRoutes registers all HTTP routes using Echo
func registerRoutes(e *echo.Echo, handler *HandlerAdapter) {
	// Health check
	e.GET("/health", handler.HealthCheck)

	// Swagger documentation
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// API v1 group
	v1 := e.Group("/api/v1")

	// Portfolio endpoints
	portfolio := v1.Group("/portfolio")
	portfolio.GET("", handler.GetPortfolioList)
	portfolio.POST("", handler.CreatePortfolio)
	portfolio.GET("/:portfolioID", handler.GetPortfolio)
	portfolio.GET("/:portfolioID/assets", handler.GetPortfolioAssets)
	portfolio.POST("/holdings", handler.AddHolding)
	portfolio.PUT("/holdings/:holdingID", handler.UpdateHolding)
	portfolio.DELETE("/holdings/:holdingID", handler.DeleteHolding)

	//Transaction endpoints
	transactions := v1.Group("/transactions")
	transactions.GET("/:portfolioID", handler.GetTransactions)
}

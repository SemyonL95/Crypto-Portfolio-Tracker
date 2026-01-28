package server

import (
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"testtask/internal/adapters/logger"
	"testtask/internal/domain/holding"
	"testtask/internal/domain/portfolio"
	"time"

	httpports "testtask/internal/ports/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// HandlerAdapter adapts holding services to HTTP handlers
type HandlerAdapter struct {
	transactionService httpports.TransactionService
	portfolioService   httpports.PortfolioService
	priceService       httpports.PriceService
	tokensService      httpports.TokensService
	logger             *logger.Logger
}

// NewHandlerAdapter creates a new handler adapter
func NewHandlerAdapter(
	transactionService httpports.TransactionService,
	portfolioService httpports.PortfolioService,
	priceService httpports.PriceService,
	tokensService httpports.TokensService,
	logger *logger.Logger,
) *HandlerAdapter {
	return &HandlerAdapter{
		transactionService: transactionService,
		portfolioService:   portfolioService,
		priceService:       priceService,
		tokensService:      tokensService,
		logger:             logger,
	}
}

// CreatePortfolioRequest represents the request body for creating a portfolio
type CreatePortfolioRequest struct {
	Address string `json:"address"`
}

func (h *HandlerAdapter) CreatePortfolio(c echo.Context) error {
	var req CreatePortfolioRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "invalid request body",
		})
	}

	if req.Address == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "address is required",
		})
	}

	// Create holding portfolio
	newPortfolio := portfolio.NewPortfolio("", req.Address)
	if err := h.portfolioService.CreatePortfolio(c.Request().Context(), newPortfolio); err != nil {
		if errors.Is(err, portfolio.ErrPortfolioAddressExists) {
			h.logger.Warn("Portfolio creation failed: address exists", zap.String("address", req.Address), zap.Error(err))
			return c.JSON(http.StatusConflict, httpports.ErrorResponse{
				Error:   "Conflict",
				Message: err.Error(),
			})
		}

		if errors.Is(err, portfolio.ErrPortfolioNotFound) {
			h.logger.Warn("Portfolio creation failed: not found", zap.String("portfolioID", newPortfolio.ID), zap.Error(err))
			return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
				Error:   "Bad Request",
				Message: err.Error(),
			})
		}

		h.logger.Error("Portfolio creation failed", zap.String("portfolioID", newPortfolio.ID), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	if newPortfolio.ID != "" {
		createdPortfolio, err := h.portfolioService.GetPortfolio(c.Request().Context(), newPortfolio.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
				Error:   "Internal Server Error",
				Message: "portfolio created but failed to retrieve",
			})
		}
		// Return the created portfolio without prices for creation response
		return c.JSON(http.StatusCreated, httpports.ToHTTPPortfolio(createdPortfolio))
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"address": req.Address,
		"message": "portfolio created successfully",
	})
}

func (h *HandlerAdapter) GetPortfolio(c echo.Context) error {
	portofolioID := c.Param("portfolioID")
	if portofolioID == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "portfolioID is required",
		})
	}

	p, err := h.portfolioService.GetPortfolio(c.Request().Context(), portofolioID)
	if err != nil {
		h.logger.Error("Failed to get p", zap.String("portfolioID", portofolioID), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, httpports.ToHTTPPortfolio(p))
}

func (h *HandlerAdapter) GetPortfolioList(c echo.Context) error {
	portofolioID := c.Param("portfolioID")
	if portofolioID == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "portfolioID is required",
		})
	}

	p, err := h.portfolioService.ListPortfolios(c.Request().Context())
	if err != nil {
		h.logger.Error("Failed to get p", zap.String("portfolioID", portofolioID), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, httpports.ToHTTPPortfolios(p))
}

// AddHolding handles POST /api/v1/portfolio/:portofolioID/holdingRepo
func (h *HandlerAdapter) AddHolding(c echo.Context) error {
	portofolioID := c.Param("portofolioID")
	if portofolioID == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "portofolioID is required",
		})
	}

	var req = struct {
		TokenAddress string `json:"token_address"`
		Amount       int    `json:"amount"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "invalid request body",
		})
	}

	if req.TokenAddress == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "token_address should be not nil",
		})
	}

	t, ok := h.tokensService.GetTokenByAddress(c.Request().Context(), req.TokenAddress)
	if !ok {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "not supported token address",
		})
	}

	holding := holding.NewHolding(portofolioID, "", t, big.NewInt(int64(req.Amount)))
	if err := h.portfolioService.AddHolding(c.Request().Context(), portofolioID, holding); err != nil {
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, holding)
}

// UpdateHolding handles PUT /api/v1/portfolio/:portofolioID/holdingRepo/:holdingID
func (h *HandlerAdapter) UpdateHolding(c echo.Context) error {
	portofolioID := c.Param("portofolioID")
	holdingID := c.Param("holdingID")
	if portofolioID == "" || holdingID == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "portofolioID and holdingID are required",
		})
	}

	var updateReq = struct {
		Amount int `json:"amount"`
	}{}

	if err := c.Bind(&updateReq); err != nil {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "invalid request body",
		})
	}

	if err := h.portfolioService.UpdateHolding(c.Request().Context(), portofolioID, holdingID, big.NewInt(int64(updateReq.Amount))); err != nil {
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, nil)
}

// DeleteHolding handles DELETE /api/v1/portfolio/:portofolioId/holdingRepo/:holdingID
func (h *HandlerAdapter) DeleteHolding(c echo.Context) error {
	portofolioID := c.Param("portofolioID")
	holdingID := c.Param("holdingID")
	if portofolioID == "" || holdingID == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "portofolioID and holdingID are required",
		})
	}

	if err := h.portfolioService.DeleteHolding(c.Request().Context(), portofolioID, holdingID); err != nil {
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *HandlerAdapter) GetTransactions(c echo.Context) error {
	filters := httpports.TransactionFilters{
		Page:     1,
		PageSize: 20,
	}

	// Parse query parameters
	addressParam := c.QueryParam("address")
	if addressParam == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "address is required ",
		})
	}
	filters.Address = &addressParam

	if typeParam := c.QueryParam("type"); typeParam != "" {
		filters.Type = &typeParam
	}
	if statusParam := c.QueryParam("status"); statusParam != "" {
		filters.Status = &statusParam
	}
	if tokenParam := c.QueryParam("token"); tokenParam != "" {
		filters.Token = &tokenParam
	}
	if fromDateParam := c.QueryParam("from_date"); fromDateParam != "" {
		if fromDate, err := time.Parse(time.RFC3339, fromDateParam); err == nil {
			filters.FromDate = &fromDate
		}
	}
	if toDateParam := c.QueryParam("to_date"); toDateParam != "" {
		if toDate, err := time.Parse(time.RFC3339, toDateParam); err == nil {
			filters.ToDate = &toDate
		}
	}
	if pageParam := c.QueryParam("page"); pageParam != "" {
		if page, err := strconv.Atoi(pageParam); err == nil && page > 0 {
			filters.Page = page
		}
	}
	if pageSizeParam := c.QueryParam("page_size"); pageSizeParam != "" {
		if pageSize, err := strconv.Atoi(pageSizeParam); err == nil && pageSize > 0 {
			filters.PageSize = pageSize
		}
	}

	// Map HTTP filters to domain filter options.
	opts, err := httpports.ToDomainFilterOptions(filters)
	if err != nil {
		h.logger.Error("Invalid transaction filters", zap.Any("filters", filters), zap.Error(err))
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
		})
	}

	transactions, total, err := h.transactionService.GetTransactions(c.Request().Context(), addressParam, opts)
	if err != nil {
		h.logger.Error("Failed to get transactions", zap.Any("filters", filters), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	totalPages := (total + filters.PageSize - 1) / filters.PageSize
	if totalPages < 1 {
		totalPages = 1
	}

	response := httpports.PaginatedResponse{
		Data:       httpports.ToHTTPTransactionsFromSlice(transactions),
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	return c.JSON(http.StatusOK, response)
}

func (h *HandlerAdapter) GetPortfolioAssets(c echo.Context) error {
	portfolioID := c.Param("portfolioID")
	if portfolioID == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "portfolioID is required",
		})
	}

	// Get currency from query parameter, default to "usd"
	currency := c.QueryParam("currency")
	if currency == "" {
		currency = "usd"
	}

	p, assets, err := h.portfolioService.GetPortfolioAssets(c.Request().Context(), portfolioID, currency)
	if err != nil {
		if errors.Is(err, portfolio.ErrPortfolioNotFound) {
			h.logger.Warn("Portfolio not found", zap.String("portfolioID", portfolioID), zap.Error(err))
			return c.JSON(http.StatusNotFound, httpports.ErrorResponse{
				Error:   "Not Found",
				Message: err.Error(),
			})
		}

		h.logger.Error("Failed to get portfolio assets", zap.String("portfolioID", portfolioID), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, httpports.ToHTTPPortfolioAssets(p, assets))
}

func (h *HandlerAdapter) HealthCheck(c echo.Context) error {
	status := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "crypto-portfolio-tracker",
		"version":   "1.0.0",
	}
	return c.JSON(http.StatusOK, status)
}

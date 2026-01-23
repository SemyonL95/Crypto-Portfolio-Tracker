package server

import (
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"testtask/internal/adapters/logger"
	"testtask/internal/domain/portfolio"
	"testtask/internal/domain/price"
	"time"

	httpports "testtask/internal/ports/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// HandlerAdapter adapts domain services to HTTP handlers
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

	// Get portfolioID from path parameter
	portfolioID := c.Param("portfolioID")

	// Create domain portfolio
	domainPortfolio := &portfolio.Portfolio{
		ID:      portfolioID,
		Address: req.Address,
	}

	if err := h.portfolioService.CreatePortfolio(c.Request().Context(), domainPortfolio); err != nil {
		// Check if it's a duplicate address error
		if errors.Is(err, portfolio.ErrPortfolioAddressExists) {
			h.logger.Warn("Portfolio creation failed: address exists", zap.String("address", req.Address), zap.Error(err))
			return c.JSON(http.StatusConflict, httpports.ErrorResponse{
				Error:   "Conflict",
				Message: err.Error(),
			})
		}
		// Check for invalid portfolio error
		if errors.Is(err, portfolio.ErrPortfolioNotFound) {
			h.logger.Warn("Portfolio creation failed: not found", zap.String("portfolioID", portfolioID), zap.Error(err))
			return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
				Error:   "Bad Request",
				Message: err.Error(),
			})
		}
		h.logger.Error("Portfolio creation failed", zap.String("portfolioID", portfolioID), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	// Fetch the created portfolio to return it
	// If portfolioID was provided in path, use it; otherwise service generated a new ID
	// but we don't have access to it, so we'll return a simpler response
	if portfolioID != "" {
		createdPortfolio, err := h.portfolioService.GetPortfolio(c.Request().Context(), portfolioID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
				Error:   "Internal Server Error",
				Message: "portfolio created but failed to retrieve",
			})
		}
		// Return the created portfolio without prices for creation response
		return c.JSON(http.StatusCreated, httpports.ToHTTPPortfolio(createdPortfolio, nil))
	}

	// If portfolioID was empty, service generated a new ID but we don't have access to it
	// Return a success response (in production, you might want to add GetByAddress to service)
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"address": req.Address,
		"message": "portfolio created successfully",
	})
}

func (h *HandlerAdapter) GetPortfolio(c echo.Context) error {
	portofolioID := c.Param("portofolioID")
	if portofolioID == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "portofolioID is required",
		})
	}

	portfolio, err := h.portfolioService.GetPortfolio(c.Request().Context(), portofolioID)
	if err != nil {
		h.logger.Error("Failed to get portfolio", zap.String("portfolioID", portofolioID), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	var tokens []*price.Token
	for _, holding := range portfolio.Holdings {
		tokens = append(tokens, holding.Token)
	}

	prices, err := h.priceService.GetPrices(c.Request().Context(), tokens, "usd")
	if err != nil {
		h.logger.Error("Failed to get prices", zap.String("portfolioID", portofolioID), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, httpports.ToHTTPPortfolio(portfolio, prices))
}

// AddHolding handles POST /api/v1/portfolio/:portofolioID/holdings
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

	// Check if token is supported by address or symbol
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

	holding := portfolio.NewHolding(portofolioID, "", t, big.NewInt(int64(req.Amount)))
	if err := h.portfolioService.AddHolding(c.Request().Context(), portofolioID, holding); err != nil {
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, holding)
}

// UpdateHolding handles PUT /api/v1/portfolio/:portofolioID/holdings/:holdingID
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

// DeleteHolding handles DELETE /api/v1/portfolio/:portofolioId/holdings/:holdingID
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
	if addressParam := c.QueryParam("address"); addressParam != "" {
		filters.Address = &addressParam
	}
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

	transactions, total, err := h.transactionService.GetTransactions(c.Request().Context(), filters)
	if err != nil {
		h.logger.Error("Failed to get transactions", zap.Any("filters", filters), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, httpports.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
		})
	}

	totalPages := (total + filters.PageSize - 1) / filters.PageSize
	response := httpports.PaginatedResponse{
		Data:       httpports.ToHTTPTransactions(transactions),
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	return c.JSON(http.StatusOK, response)
}

func (h *HandlerAdapter) GetTransactionByHash(c echo.Context) error {
	hash := c.Param("hash")
	if hash == "" {
		return c.JSON(http.StatusBadRequest, httpports.ErrorResponse{
			Error:   "Bad Request",
			Message: "hash is required",
		})
	}

	transaction, err := h.transactionService.GetTransactionByHash(c.Request().Context(), hash)
	if err != nil {
		h.logger.Warn("Transaction not found", zap.String("hash", hash), zap.Error(err))
		return c.JSON(http.StatusNotFound, httpports.ErrorResponse{
			Error:   "Not Found",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, httpports.ToHTTPTransaction(transaction))
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

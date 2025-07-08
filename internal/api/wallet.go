package api

import (
	"fmt"
	"github.com/Jiang-hao/walletApiService/internal/model"
	"github.com/Jiang-hao/walletApiService/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"strconv"
)

type WalletHandler struct {
	walletService service.WalletService
}

func NewWalletHandler(walletService service.WalletService) *WalletHandler {
	return &WalletHandler{walletService: walletService}
}

func (h *WalletHandler) Deposit(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req model.DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.walletService.Deposit(c.Request.Context(), userID, req.Amount, req.Currency, req.Reference)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *WalletHandler) Withdraw(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req model.WithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.walletService.Withdraw(c.Request.Context(), userID, req.Amount, req.Currency, req.Reference)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *WalletHandler) Transfer(c *gin.Context) {
	fromUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req model.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.walletService.Transfer(c.Request.Context(), fromUserID, req.ToWalletID, req.Amount, req.Currency, req.Reference)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		fmt.Errorf("user id null: %w", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	currency := c.Query("currency")
	if currency == "" {
		currency = "USD"
	}

	balance, err := h.walletService.GetBalance(c.Request.Context(), userID, currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.BalanceResponse{
		Balance:  balance,
		Currency: currency,
	})
}

func (h *WalletHandler) GetTransactionHistory(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	currency := c.Query("currency")
	if currency == "" {
		currency = "USD"
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page number"})
		return
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page size"})
		return
	}

	transactions, err := h.walletService.GetTransactionHistory(c.Request.Context(), userID, currency, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response []model.TransactionResponse
	for _, tx := range transactions {
		response = append(response, model.TransactionResponse{
			ID:            tx.ID,
			Amount:        tx.Amount,
			BalanceBefore: tx.BalanceBefore,
			BalanceAfter:  tx.BalanceAfter,
			Type:          tx.Type,
			Reference:     tx.Reference,
			CreatedAt:     tx.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}

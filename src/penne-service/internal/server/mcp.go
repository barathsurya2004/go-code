package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

func NewMCPServer(logger *zap.Logger, txnRepo core.TransactionRepository) *server.SSEServer {
	mcpServer := server.NewMCPServer(
		"Penne MCP",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// MCP Tool: Get transactions by user UUID
	getTxnsTool := mcp.NewTool("get_transactions_by_user",
		mcp.WithDescription("Get all transactions for a specific user"),
		mcp.WithString("user_uuid", mcp.Required(), mcp.Description("The UUID of the user")),
	)

	mcpServer.AddTool(getTxnsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]any)
		userUUID, ok := args["user_uuid"].(string)
		if !ok || strings.TrimSpace(userUUID) == "" {
			return mcp.NewToolResultError("user_uuid argument is required and must be a string"), nil
		}

		txns, err := txnRepo.GetTransactionsByUserUUID(userUUID)
		if err != nil {
			logger.Error("Failed to fetch transactions for MCP tool", zap.Error(err))
			return mcp.NewToolResultError("failed to fetch transactions: " + err.Error()), nil
		}

		data, err := json.MarshalIndent(txns, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to serialize transactions: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})

	// MCP Tool: Create a new transaction
	createTxnTool := mcp.NewTool("create_transaction",
		mcp.WithDescription("Create a new transaction for testing"),
		mcp.WithString("user_uuid", mcp.Required(), mcp.Description("The UUID of the user")),
		mcp.WithNumber("amount_e5", mcp.Required(), mcp.Description("Amount scaled by 1e5")),
		mcp.WithString("country_iso2", mcp.Required(), mcp.Description("2-letter Country ISO code")),
		mcp.WithString("category", mcp.Required(), mcp.Description("Category of transaction")),
		mcp.WithString("bank_name", mcp.Required(), mcp.Description("Bank name")),
		mcp.WithString("txn_type", mcp.Required(), mcp.Description("Transaction type (e.g. credit/debit)")),
	)

	mcpServer.AddTool(createTxnTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]any)
		userUUID, _ := args["user_uuid"].(string)
		countryISO, _ := args["country_iso2"].(string)
		category, _ := args["category"].(string)
		bankName, _ := args["bank_name"].(string)
		txnType, _ := args["txn_type"].(string)

		var amountE5 float64
		switch v := args["amount_e5"].(type) {
		case float64:
			amountE5 = v
		case int:
			amountE5 = float64(v)
		case int64:
			amountE5 = float64(v)
		}

		if userUUID == "" || countryISO == "" || category == "" || bankName == "" || txnType == "" {
			return mcp.NewToolResultError("missing required fields for transaction creation"), nil
		}

		txn := &core.Transaction{
			UserUUID:   userUUID,
			AmountE5:   amountE5,
			CountryISO: countryISO,
			Category:   category,
			BankName:   bankName,
			Type:       txnType,
		}

		if err := txnRepo.CreateTransaction(txn); err != nil {
			logger.Error("Failed to create transaction via MCP tool", zap.Error(err))
			return mcp.NewToolResultError(fmt.Sprintf("failed to create transaction: %v", err)), nil
		}

		data, _ := json.MarshalIndent(txn, "", "  ")
		return mcp.NewToolResultText("Transaction created successfully:\n" + string(data)), nil
	})

	sseServer := server.NewSSEServer(mcpServer)
	return sseServer
}

func MCPAuthMiddleWare(repo core.TokenRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}
			token := parts[1]

			if token == "" {
				http.Error(w, "user token is missing", http.StatusUnauthorized)
				return
			}

			if strings.HasPrefix(token, "mcp_") {
				mcpToken := strings.TrimPrefix(token, "mcp_")

				tokenObj, err := repo.GetToken(mcpToken)
				if err != nil {
					http.Error(w, "invalid token", http.StatusUnauthorized)
					return
				}
				if tokenObj.ExpiresAt != nil && tokenObj.ExpiresAt.Before(time.Now()) {
					http.Error(w, "token has expired", http.StatusUnauthorized)
					return
				}

				if tokenObj.Scope != nil {
					// match scopes
				}

				ctx := context.WithValue(r.Context(), "user_id", tokenObj)

				next.ServeHTTP(w, r.WithContext(ctx))

			} else {
				http.Error(w, "invalid token type", http.StatusUnauthorized)
				return
			}

		})
	}
}

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

func handleGetTransactions(logger *zap.Logger, txnRepo core.TransactionRepository) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	}
}

func handleCreateTransaction(logger *zap.Logger, txnRepo core.TransactionRepository) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	}
}

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
	mcpServer.AddTool(getTxnsTool, handleGetTransactions(logger, txnRepo))

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
	mcpServer.AddTool(createTxnTool, handleCreateTransaction(logger, txnRepo))

	sseServer := server.NewSSEServer(mcpServer)
	return sseServer
}

func MCPAuthMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 1. ALWAYS allow CORS and OPTIONS requests to pass
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 2. Check for the auth token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {

			// THE FIX: Mandatory WWW-Authenticate header for Gemini OAuth recognition
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp"`)

			http.Error(w, "Missing Authorization Header", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// 3. Validate token
		if token != "penne_mcp_test_token_123" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp", error="invalid_token"`)
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", "user_barath_123")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OAuthAuthorizeHandler handles Gemini's OAuth authorization redirect.
func OAuthAuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	if redirectURI == "" {
		http.Error(w, "missing redirect_uri", http.StatusBadRequest)
		return
	}

	fakeAuthCode := "fake_auth_code_999"

	redirectURL := fmt.Sprintf("%s?code=%s&state=%s", redirectURI, fakeAuthCode, state)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// OAuthTokenHandler returns the static Bearer token for OAuth token exchanges.
func OAuthTokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. OAuth requires POST.", http.StatusMethodNotAllowed)
		return
	}

	response := `{
		"access_token": "penne_mcp_test_token_123",
		"token_type": "Bearer",
		"expires_in": 31536000
	}`

	w.Write([]byte(response))
}

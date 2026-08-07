package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

func NewMCPServer(logger *zap.Logger, txnRepo core.TransactionRepository) *server.StreamableHTTPServer {
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

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = os.Getenv("MCP_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://yak-crisp-vulture.ngrok-free.app"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	streamableServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithProtectedResourceMetadata(server.ProtectedResourceMetadataConfig{
			Resource:               baseURL,
			AuthorizationServers:   []string{baseURL},
			ScopesSupported:        []string{"mcp:read", "mcp:write"},
			BearerMethodsSupported: []string{"header"},
		}),
		server.WithStreamableHTTPCORS(
			server.WithCORSAllowedOrigins("*"),
			server.WithCORSAllowCredentials(),
			server.WithCORSMaxAge(300),
		),
		server.WithDisableLocalhostProtection(true),
	)
	return streamableServer
}

func getBaseURL(r *http.Request) string {
	if envURL := os.Getenv("BASE_URL"); envURL != "" {
		return strings.TrimSuffix(envURL, "/")
	}
	if envURL := os.Getenv("MCP_BASE_URL"); envURL != "" {
		return strings.TrimSuffix(envURL, "/")
	}

	scheme := "https"
	if r.TLS == nil {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if strings.HasPrefix(r.Host, "localhost") || strings.HasPrefix(r.Host, "127.0.0.1") || strings.HasPrefix(r.Host, "[::1]") {
			scheme = "http"
		} else {
			scheme = "https"
		}
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func MCPAuthMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 1. ALWAYS allow CORS and OPTIONS requests to pass
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Mcp-Session-Id, Last-Event-ID, ngrok-skip-browser-warning")
		w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id")
		w.Header().Set("ngrok-skip-browser-warning", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		baseURL := getBaseURL(r)
		resourceMetadataURL := baseURL + "/.well-known/oauth-protected-resource"

		// 2. Check for the auth token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {

			// Mandatory WWW-Authenticate header for Gemini OAuth recognition & RFC 9728 discovery
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="mcp", resource_metadata="%s"`, resourceMetadataURL))

			http.Error(w, "Missing Authorization Header", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// 3. Validate token
		if token != "penne_mcp_test_token_123" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="mcp", error="invalid_token", resource_metadata="%s"`, resourceMetadataURL))
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", "user_barath_123")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OAuthAuthorizeHandler handles Gemini's OAuth authorization redirect.
func OAuthAuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

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
		"expires_in": 31536000,
		"refresh_token": "penne_mcp_test_refresh_123"
	}`

	w.Write([]byte(response))
}

// OAuthMetadataHandler returns RFC 8414 OAuth 2.0 Authorization Server Metadata for Gemini auto-discovery.
func OAuthMetadataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	baseURL := getBaseURL(r)

	metadata := map[string]any{
		"issuer":                                baseURL,
		"authorization_endpoint":                baseURL + "/oauth/authorize",
		"token_endpoint":                        baseURL + "/oauth/token",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"code_challenge_methods_supported":      []string{"S256"},
		"scopes_supported":                      []string{"mcp:read", "mcp:write"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "none"},
	}

	json.NewEncoder(w).Encode(metadata)
}

// ProtectedResourceMetadataHandler handles RFC 9728 OAuth Protected Resource Metadata requests.
func ProtectedResourceMetadataHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := getBaseURL(r)

	handler := server.NewProtectedResourceMetadataHandler(server.ProtectedResourceMetadataConfig{
		Resource:               baseURL,
		AuthorizationServers:   []string{baseURL},
		ScopesSupported:        []string{"mcp:read", "mcp:write"},
		BearerMethodsSupported: []string{"header"},
	})

	handler.ServeHTTP(w, r)
}

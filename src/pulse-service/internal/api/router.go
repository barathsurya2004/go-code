package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowOrigin := "*"
		if origin != "" {
			allowOrigin = origin
		}

		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-user-uuid, x-request-id, ngrok-skip-browser-warning, X-Requested-With, Accept, Origin, Cache-Control, Pragma")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func NewRouter(handler *PulseHandler) *mux.Router {
	r := mux.NewRouter()
	r.Use(CORSMiddleware)

	// Health check
	r.HandleFunc("/health", handler.HandleHealth).Methods(http.MethodGet, http.MethodOptions)

	// Pulse sync & stats endpoints
	prefix := "/api/v1/pulse"
	r.HandleFunc(prefix+"/sync", handler.HandleSync).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc(prefix+"/sync", handler.HandleDeleteUserData).Methods(http.MethodDelete, http.MethodOptions)
	r.HandleFunc(prefix+"/stats", handler.HandleGetStats).Methods(http.MethodGet, http.MethodOptions)

	// Pulse User & Auth endpoints
	r.HandleFunc(prefix+"/users", handler.HandleCreateUser).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc(prefix+"/user", handler.HandleGetUser).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc(prefix+"/users/me", handler.HandleGetUser).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc(prefix+"/auth/signup", handler.HandleSignup).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc(prefix+"/auth/login", handler.HandleLogin).Methods(http.MethodPost, http.MethodOptions)

	// Convenience direct aliases (when invoked directly on port 8082)
	r.HandleFunc("/sync", handler.HandleSync).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/sync", handler.HandleDeleteUserData).Methods(http.MethodDelete, http.MethodOptions)
	r.HandleFunc("/stats", handler.HandleGetStats).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/users", handler.HandleCreateUser).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/user", handler.HandleGetUser).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/auth/signup", handler.HandleSignup).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/auth/login", handler.HandleLogin).Methods(http.MethodPost, http.MethodOptions)

	return r
}

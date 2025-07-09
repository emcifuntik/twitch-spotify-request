package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/emcifuntik/twitch-spotify-request/internal/service"
	"github.com/gorilla/mux"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Try to get token from cookie as fallback
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				writeAPIError(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			authHeader = "Bearer " + cookie.Value
		}

		// Extract token from Bearer header
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeAPIError(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token
		claims, err := service.ValidateToken(tokenString)
		if err != nil {
			if err == service.ErrExpiredToken {
				writeAPIError(w, "Token expired", http.StatusUnauthorized)
			} else {
				writeAPIError(w, "Invalid token", http.StatusUnauthorized)
			}
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthMiddleware validates JWT tokens but doesn't require them
func OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Try to get token from cookie as fallback
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				// No auth provided, continue without setting claims
				next.ServeHTTP(w, r)
				return
			}
			authHeader = "Bearer " + cookie.Value
		}

		// Extract token from Bearer header
		if !strings.HasPrefix(authHeader, "Bearer ") {
			// Invalid format, continue without setting claims
			next.ServeHTTP(w, r)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token
		claims, err := service.ValidateToken(tokenString)
		if err != nil {
			// Invalid token, continue without setting claims
			next.ServeHTTP(w, r)
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserValidationMiddleware ensures users can only access their own data
func UserValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaimsFromContext(r)
		if !ok {
			writeAPIError(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Get userID from URL path
		vars := mux.Vars(r)
		requestedUserID := vars["userID"]

		// Check if the authenticated user matches the requested user
		if claims.ChannelID != requestedUserID {
			writeAPIError(w, "Access denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetClaimsFromContext retrieves JWT claims from request context
func GetClaimsFromContext(r *http.Request) (*service.Claims, bool) {
	claims, ok := r.Context().Value("claims").(*service.Claims)
	return claims, ok
}

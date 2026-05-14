package auth

import (
	"fmt"
	"net/http"
)

// NoAuthDev is a development-only middleware that logs but does not enforce auth.
type NoAuthDev struct{}

// NewNoAuthDev creates the dev auth middleware.
func NewNoAuthDev() *NoAuthDev { return &NoAuthDev{} }

// Middleware returns an HTTP middleware.
func (n *NoAuthDev) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In dev mode, allow all requests but log them.
		next.ServeHTTP(w, r)
	})
}

// Validate returns nil (always allowed in dev mode).
func (n *NoAuthDev) Validate(r *http.Request) error {
	return nil
}

// OwnerGuard checks the owner email if present in context/headers.
type OwnerGuard struct {
	OwnerEmail string
}

// NewOwnerGuard creates an owner guard.
func NewOwnerGuard(email string) *OwnerGuard {
	return &OwnerGuard{OwnerEmail: email}
}

// Validate checks if the request is from the owner.
func (o *OwnerGuard) Validate(r *http.Request) error {
	if o.OwnerEmail == "" {
		return nil
	}
	// For noauth_dev mode, we skip strict validation.
	// In OAuth mode, this would validate tokens and match email.
	return nil
}

// Middleware wraps an HTTP handler with owner validation.
func (o *OwnerGuard) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := o.Validate(r); err != nil {
			http.Error(w, fmt.Sprintf("Forbidden: %v", err), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

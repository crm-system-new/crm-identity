package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/crm-system-new/crm-shared/pkg/auth"
	"github.com/crm-system-new/crm-shared/pkg/health"
)

func NewRouter(authHandler *AuthHandler, userHandler *UserHandler, jwtManager *auth.JWTManager, checker *health.Checker) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.RefreshToken)
		})

		// Protected user routes
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(jwtManager))

			r.Get("/users/{id}", userHandler.GetUser)
			r.Put("/users/{id}", userHandler.UpdateProfile)
			r.Put("/users/{id}/password", userHandler.ChangePassword)

			// Admin-only
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole("admin"))
				r.Get("/users", userHandler.ListUsers)
			})
		})
	})

	// Health checks
	r.Get("/health/live", checker.LiveHandler())
	r.Get("/health/ready", checker.ReadyHandler())

	return r
}

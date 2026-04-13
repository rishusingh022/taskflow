package router

import (
	"net/http"

	"taskflow/internal/handler"
	"taskflow/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func New(
	authHandler *handler.AuthHandler,
	projectHandler *handler.ProjectHandler,
	taskHandler *handler.TaskHandler,
	authMw *middleware.AuthMiddleware,
) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Recoverer)
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api", func(r chi.Router) {
		// public routes
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMw.Authenticate)

			r.Get("/projects", projectHandler.List)
			r.Post("/projects", projectHandler.Create)
			r.Get("/projects/{id}", projectHandler.GetByID)
			r.Patch("/projects/{id}", projectHandler.Update)
			r.Delete("/projects/{id}", projectHandler.Delete)
			r.Get("/projects/{id}/stats", projectHandler.GetStats)

			r.Get("/projects/{id}/tasks", taskHandler.ListByProject)
			r.Post("/projects/{id}/tasks", taskHandler.Create)

			r.Patch("/tasks/{id}", taskHandler.Update)
			r.Delete("/tasks/{id}", taskHandler.Delete)
		})
	})

	return r
}

package api

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/Val-senseisama/payments/cmd/config"
	"github.com/Val-senseisama/payments/internal/common/redis"
	"github.com/Val-senseisama/payments/internal/domain/company"
	"github.com/Val-senseisama/payments/internal/domain/users"
	"github.com/Val-senseisama/payments/types"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	redisClient "github.com/redis/go-redis/v9"
)

type APIServer struct {
	addr       string
	config     config.Config
	db         *sql.DB
	rdb        *redisClient.Client
	auditStore types.AuditStore
}

func NewAPIServer(addr string, cfg config.Config, db *sql.DB, rdb *redisClient.Client, auditStore types.AuditStore) *APIServer {
	return &APIServer{
		addr:       addr,
		config:     cfg,
		db:         db,
		rdb:        rdb,
		auditStore: auditStore,
	}
}

func (s *APIServer) mountUserRoutes(r chi.Router) {
	userStore := users.NewStore(s.db)
	redisStore := redis.NewRedisStore(s.rdb)
	handler := users.NewHandler(userStore, redisStore, s.config, s.auditStore)

	r.Route("/users", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})
}

func (s *APIServer) mountCompanyRoutes(r chi.Router) {
	companyStore := company.NewStore(s.db)
	redisStore := redis.NewRedisStore(s.rdb)
	handler := company.NewHandle(companyStore, redisStore, s.config, s.auditStore)

	r.Route("/companies", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})
}

func (s *APIServer) Run() error {
	log.Println("Starting server on port ", s.addr)

	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/api/v1", func(r chi.Router) {
		s.mountUserRoutes(r)
		s.mountCompanyRoutes(r)
	})

	router.Get("/health", s.handleHealth)

	log.Println("Server started on port ", s.addr)
	return http.ListenAndServe(s.addr, router)
}

// health

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

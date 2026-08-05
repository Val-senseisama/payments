package api

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/Val-senseisama/payments/cmd/config"
	"github.com/Val-senseisama/payments/internal/common/auth"
	"github.com/Val-senseisama/payments/internal/common/redis"
	"github.com/Val-senseisama/payments/internal/domain/account"
	"github.com/Val-senseisama/payments/internal/domain/audit"
	"github.com/Val-senseisama/payments/internal/domain/company"
	"github.com/Val-senseisama/payments/internal/domain/ledger"
	"github.com/Val-senseisama/payments/internal/domain/payments"
	"github.com/Val-senseisama/payments/internal/domain/profiles"
	"github.com/Val-senseisama/payments/internal/domain/transactions"
	"github.com/Val-senseisama/payments/internal/domain/users"
	"github.com/Val-senseisama/payments/internal/mailer"
	"github.com/Val-senseisama/payments/types"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	redisClient "github.com/redis/go-redis/v9"
)

type APIServer struct {
	addr        string
	config      config.Config
	db          *sql.DB
	rdb         *redisClient.Client
	auditStore  types.AuditStore
	auditWorker *audit.Worker
	mailer      *mailer.Mailer
}

func NewAPIServer(addr string, cfg config.Config, db *sql.DB, rdb *redisClient.Client, auditStore types.AuditStore) *APIServer {
	return &APIServer{
		addr:        addr,
		config:      cfg,
		db:          db,
		rdb:         rdb,
		auditStore:  auditStore,
		auditWorker: audit.NewWorker(auditStore, 100),
		mailer:      mailer.New(cfg.ResendAPIKey, cfg.EmailFrom),
	}
}

func (s *APIServer) mountUserRoutes(r chi.Router) {
	userStore := users.NewStore(s.db)
	redisStore := redis.NewRedisStore(s.rdb)
	handler := users.NewHandler(userStore, redisStore, s.config, s.auditStore, s.mailer)

	r.Route("/users", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})
}

func (s *APIServer) mountCompanyRoutes(r chi.Router) {
	companyStore := company.NewStore(s.db)
	userStore := users.NewStore(s.db)
	accountStore := account.NewStore(s.db)
	redisStore := redis.NewRedisStore(s.rdb)
	handler := company.NewHandler(companyStore, userStore, redisStore, s.config, s.auditStore, s.mailer, accountStore)

	r.Route("/companies", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})
}

func (s *APIServer) mountProfileRoutes(r chi.Router) {
	profileStore := profiles.NewStore(s.db)
	handler := profiles.NewHandler(profileStore, s.config, s.auditStore)

	r.Route("/profiles", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})
}

func (s *APIServer) mountAccountRoutes(r chi.Router) {
	accountStore := account.NewStore(s.db)
	handler := account.NewHandler(accountStore, s.auditStore)

	r.Route("/accounts", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})
}

func (s *APIServer) mountTransactionRoutes(r chi.Router) {
	txnStore := transactions.NewStore(s.db)
	redisStore := redis.NewRedisStore(s.rdb)
	handler := transactions.NewHandler(txnStore, redisStore, s.config, s.auditWorker)

	r.Route("/transactions", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})
}

func (s *APIServer) mountLedgerRoutes(r chi.Router) {
	ledgerStore := ledger.NewStore(s.db)
	txnStore := transactions.NewStore(s.db)
	redisStore := redis.NewRedisStore(s.rdb)
	postWorker := ledger.NewPostWorker(ledgerStore, txnStore, redisStore, s.auditWorker)
	handler := ledger.NewHandler(ledgerStore, txnStore, postWorker)

	r.Route("/ledger", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})
}

func (s *APIServer) mountPaymentRoutes(r chi.Router) {
	paymentStore := payments.NewStore(s.db)
	txnStore := transactions.NewStore(s.db)
	redisStore := redis.NewRedisStore(s.rdb)
	mockPSP := payments.NewMockPSPAdapter("mock")

	service := payments.NewService(paymentStore, txnStore, redisStore, []types.PSPAdapter{mockPSP}, s.auditWorker)
	handler := payments.NewHandler(service, paymentStore)

	r.Route("/payments", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})
}

func (s *APIServer) Run() error {

	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/api/v1", func(r chi.Router) {
		s.mountUserRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(auth.AuthMiddleware([]byte(s.config.JWTSecret)))
			s.mountCompanyRoutes(r)
			s.mountProfileRoutes(r)
			s.mountAccountRoutes(r)
			s.mountTransactionRoutes(r)
			s.mountLedgerRoutes(r)
			s.mountPaymentRoutes(r)
		})
	})

	router.Get("/health", s.handleHealth)

	// Mount Web Dashboard Static Files
	workDir, _ := os.Getwd()
	filesDir := http.Dir(path.Join(workDir, "web"))
	fileServer(router, "/", filesDir)

	log.Println("Server started on port ", s.addr)
	return http.ListenAndServe(s.addr, router)
}

func fileServer(r chi.Router, pathStr string, root http.FileSystem) {
	if strings.HasSuffix(pathStr, "/") && pathStr != "/" {
		pathStr = pathStr[:len(pathStr)-1]
	}

	fs := http.StripPrefix(pathStr, http.FileServer(root))

	r.Get(pathStr+"*", func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}

// health

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

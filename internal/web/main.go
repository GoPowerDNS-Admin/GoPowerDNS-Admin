package web

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/activity"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/group"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/server/configuration"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/settings/pdnsserver"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/settings/zone"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/role"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/tag"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/user"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/zonetag"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/profile"
	oidchandler "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/auth/oidc"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/login"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/logout"
	zoneadd "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/zone/add"
	zoneedit "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/zone/edit"
	authmiddleware "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/middleware/auth"
	pdnsmiddleware "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/middleware/pdns"
)

// Service represents the web service.
type Service struct {
	App          *fiber.App
	cfg          *config.Config
	fastShutDown bool
	alive        atomic.Bool
	db           *gorm.DB
	authService  *auth.Service
}

// Start starts the web service on the given address.
func (s *Service) Start(addr string) error {
	var doneFiber = make(chan bool)

	go func() {
		if err := s.App.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Msgf("fiber listen error: %v", err)
		}

		doneFiber <- true
	}()

	<-doneFiber // wait for fiber to stop

	return nil
}

// WaitShutdown waits for graceful shutdown of tweety.
func (s *Service) WaitShutdown() {
	irqSig := make(chan os.Signal, 1)
	signal.Notify(irqSig, syscall.SIGINT, syscall.SIGTERM)

	// Wait interrupt or shutdown request through /shutdown
	sig := <-irqSig
	log.Info().Msgf("shutdown request (signal: %v)", sig)

	// Graceful shutdown for reverse proxies: set status to fail, so checkalive returns fail.
	if !s.fastShutDown {
		log.Info().Msgf(
			"graceful shutdown: return 503 while %d seconds to let LB to remove this pod from active targets",
			s.cfg.Webserver.ShutDownTime,
		)

		s.alive.Store(false)
		time.Sleep(time.Duration(s.cfg.Webserver.ShutDownTime) * time.Second)
	}

	// stop fiber http server
	serverShutdown := make(chan struct{})

	go func() {
		log.Info().Msg("stopping http server ...")

		err := s.App.Shutdown()
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		serverShutdown <- struct{}{}
	}()

	<-serverShutdown
	log.Info().Msg("http server was stopped ... good bye...")
}

// New creates a new web service with the given configuration.
func New(cfg *config.Config, db *gorm.DB) *Service {
	if cfg == nil {
		panic("config cannot be nil")
	}

	if db == nil {
		panic("db cannot be nil")
	}

	httpFS := http.FS(templateEmbedFS{embeddedTemplates})
	templateEngine := html.NewFileSystem(httpFS, ".gohtml")

	// in debug mode, use local filesystem for templates
	if cfg.DevMode {
		templateEngine = html.New("./internal/web/templates", ".gohtml")
		templateEngine.ShouldReload = true

		log.Warn().Msg("debug mode enabled: using local filesystem for templates")
	}

	// Add template helper functions
	templateEngine.AddFunc("iterate", func(count int) []int {
		result := make([]int, count)
		for i := range result {
			result[i] = i
		}

		return result
	})
	templateEngine.AddFunc("add", func(a, b int) int {
		return a + b
	})
	templateEngine.AddFunc("sub", func(a, b int) int {
		return a - b
	})

	// create fiber app
	app := fiber.New(
		fiber.Config{
			ReadBufferSize:    8192,
			AppName:           "GoPowerDNS-Admin",
			CaseSensitive:     true,
			Immutable:         true,
			Views:             templateEngine,
			PassLocalsToViews: true,
		},
	)

	// serve embedded static files
	staticFS, err := fs.Sub(embeddedStaticFiles, "static")
	if err != nil {
		panic("failed to create static sub-filesystem: " + err.Error())
	}

	app.Use("/static",
		static.New("", static.Config{
			FS:     staticFS,
			Browse: true,
		}),
	)

	// basic auth middleware
	app.Use(authmiddleware.Middleware)

	// Initialize auth service
	authService := auth.NewService(db)

	// Add permissions to fiber.Locals middleware (after auth)
	app.Use(auth.AddPermissionsToLocals(authService))

	// init web service
	service := &Service{
		cfg:         cfg,
		App:         app,
		db:          db,
		authService: authService,
	}

	// init handlers (they register their own routes with permission checks)
	login.Handler.Init(app, cfg, db)
	logout.Handler.Init(app, cfg, db)
	oidchandler.Handler.Init(app, cfg, db)
	dashboard.Handler.Init(app, cfg, db, authService)
	pdnsserver.Handler.Init(app, cfg, db, authService)
	zone.Handler.Init(app, cfg, db, authService)
	zoneadd.Handler.Init(app, cfg, db, authService)
	zoneedit.Handler.Init(app, cfg, db, authService)
	configuration.Handler.Init(app, cfg, db, authService)
	group.Handler.Init(app, cfg, db, authService)
	role.Handler.Init(app, cfg, db, authService)
	user.Handler.Init(app, cfg, db, authService)
	activity.Handler.Init(app, cfg, db, authService)
	profile.Handler.Init(app, cfg, db, authService)
	tag.Handler.Init(app, cfg, db, authService)
	zonetag.Handler.Init(app, cfg, db, authService)

	// Routes that require an initialized PowerDNS client.
	app.Use("/dashboard", pdnsmiddleware.RequireClient)
	app.Use("/zone", pdnsmiddleware.RequireClient)
	app.Use("/admin/server", pdnsmiddleware.RequireClient)

	// redirect root to dashboard
	app.Get("/", func(c fiber.Ctx) error {
		return c.Redirect().To("/dashboard")
	})

	return service
}

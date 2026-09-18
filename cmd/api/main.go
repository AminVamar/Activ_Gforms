package main

import (
	"context"
	"errors"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gform/internal/client/appscript"
	"gform/internal/client/testsource"
	"gform/internal/config"
	deliveryhttp "gform/internal/delivery/http"
	"gform/internal/repository/postgres"
	"gform/internal/usecase"
	"gform/migrations"
)

//	@title			Activ GForms API
//	@version		1.0
//	@description	API для тестирования стажёров Активбанка через Google Forms.
//	@contact.name	Amin Muborakkadamov
//	@contact.url	https://github.com/AminVamar
//	@BasePath		/

func main() {
	if err := run(); err != nil {
		slog.Error("сервис остановлен с ошибкой", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrations.Apply(ctx, pool); err != nil {
		return err
	}
	slog.Info("миграции применены")

	branchRepo := postgres.NewBranchRepo(pool)
	formRepo := postgres.NewFormRepo(pool)
	internRepo := postgres.NewInternRepo(pool)
	attemptRepo := postgres.NewAttemptRepo(pool)

	source := testsource.New(cfg.TestSourceURL, cfg.TestSourcePageSize, cfg.TestSourceTimeout, cfg.TestSourceCacheTTL)
	script := appscript.New(cfg.AppScriptWebAppURL, cfg.AppScriptSecret, cfg.AppScriptTimeout)

	server := deliveryhttp.NewServer(deliveryhttp.Deps{
		Catalog:       usecase.NewCatalog(source),
		Forms:         usecase.NewForms(source, script, formRepo, branchRepo, cfg.PublicWebhookURL),
		Submissions:   usecase.NewSubmissions(source, formRepo, branchRepo, attemptRepo),
		Grading:       usecase.NewGrading(attemptRepo, source),
		Interns:       usecase.NewInterns(internRepo, attemptRepo),
		Branches:      branchRepo,
		WebhookSecret: cfg.WebhookSecret,
	})

	srv := &stdhttp.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           server.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("сервис запущен", "port", cfg.HTTPPort, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("получен сигнал завершения, останавливаемся")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

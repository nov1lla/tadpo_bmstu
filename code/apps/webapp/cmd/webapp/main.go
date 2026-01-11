package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"plugin"
	"syscall"
	"time"

	"ppo/sdk/component"

	"ppo/webapp/internal/api"
	"ppo/webapp/internal/config"
)

func main() {
	cfg := config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dataProvider, cleanupData := loadDataProvider(cfg.DataPluginPath, cfg.DataSource)
	defer cleanupData()

	businessProvider, cleanupBusiness := loadBusinessProvider(cfg.BusinessPluginPath, cfg.HTTPTimeout, cfg.OpenAI, dataProvider)
	defer cleanupBusiness()

	server, err := api.NewServer(cfg.StaticDir, businessProvider)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}

	httpServer := &http.Server{Addr: cfg.Address, Handler: server}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("http shutdown: %v", err)
		}
	}()

	log.Printf("webapp listening on %s", cfg.Address)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http server: %v", err)
	}
}

func loadDataProvider(path, source string) (component.DataProvider, func()) {
	factory := lookupDataFactory(path)
	provider, err := factory(source)
	if err != nil {
		log.Fatalf("init data provider: %v", err)
	}
	closed := false
	cleanup := func() {
		if closed {
			return
		}
		if err := provider.Close(); err != nil {
			log.Printf("data provider close: %v", err)
		}
		closed = true
	}
	return provider, cleanup
}

func loadBusinessProvider(path string, timeout time.Duration, openai config.OpenAIConfig, data component.DataProvider) (component.BusinessProvider, func()) {
	factory := lookupBusinessFactory(path)
	businessCfg := component.BusinessConfig{
		OpenAIKey:     openai.APIKey,
		OpenAIModel:   openai.Model,
		OpenAIBaseURL: openai.BaseURL,
		HTTPTimeout:   timeout,
	}
	provider, err := factory(data, businessCfg)
	if err != nil {
		log.Fatalf("init business provider: %v", err)
	}
	cleanup := func() {
		if err := provider.Close(); err != nil {
			log.Printf("business provider close: %v", err)
		}
	}
	return provider, cleanup
}

func lookupDataFactory(path string) func(string) (component.DataProvider, error) {
	fullPath := mustAbs(path)
	plg, err := plugin.Open(fullPath)
	if err != nil {
		log.Fatalf("open data plugin: %v", err)
	}
	sym, err := plg.Lookup("NewDataProvider")
	if err != nil {
		log.Fatalf("lookup NewDataProvider: %v", err)
	}
	factory, ok := sym.(func(string) (component.DataProvider, error))
	if !ok {
		log.Fatalf("NewDataProvider has unexpected signature")
	}
	return factory
}

func lookupBusinessFactory(path string) func(component.DataProvider, component.BusinessConfig) (component.BusinessProvider, error) {
	fullPath := mustAbs(path)
	plg, err := plugin.Open(fullPath)
	if err != nil {
		log.Fatalf("open business plugin: %v", err)
	}
	sym, err := plg.Lookup("NewBusinessProvider")
	if err != nil {
		log.Fatalf("lookup NewBusinessProvider: %v", err)
	}
	factory, ok := sym.(func(component.DataProvider, component.BusinessConfig) (component.BusinessProvider, error))
	if !ok {
		log.Fatalf("NewBusinessProvider has unexpected signature")
	}
	return factory
}

func mustAbs(path string) string {
	fullPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("abs path: %v", err)
	}
	return fullPath
}

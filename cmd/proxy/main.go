package main

import (
	"flag"
	"fmt"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/octo-technology/tezos-link/backend/config"
	"github.com/octo-technology/tezos-link/backend/internal/proxy/infrastructure/cache"
	"github.com/octo-technology/tezos-link/backend/internal/proxy/infrastructure/database"
	httpinfra "github.com/octo-technology/tezos-link/backend/internal/proxy/infrastructure/http"
	"github.com/octo-technology/tezos-link/backend/internal/proxy/infrastructure/proxy"
	"github.com/octo-technology/tezos-link/backend/internal/proxy/usecases"
	pkgdatabase "github.com/octo-technology/tezos-link/backend/pkg/infrastructure/database"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"
)

var configPath = flag.String("conf", "", "Path to TOML config")

// Always run
func init() {
	flag.Parse()

	if *configPath == "" {
		log.Fatal("Program argument --conf is required")
	} else {
		_, err := config.ParseProxyConf(*configPath)
		if err != nil {
			log.Fatalf("Could not load config from %s. Reason: %s", *configPath, err)
		}
	}

	// We will configure the database in a next version
	database.Configure()
}

func main() {
	reverseURL, err := url.Parse("http://" + config.ProxyConfig.Tezos.Host + ":" + strconv.Itoa(config.ProxyConfig.Tezos.Port))
	if err != nil {
		log.Fatal(fmt.Sprintf("could not read blockchain node reverse url from configuration: %s", err))
	}
	logrus.Info("proxying requests to node: ", reverseURL)
	rp := httputil.NewSingleHostReverseProxy(reverseURL)

	// Repositories
	cb := cache.NewLruBlockchainRepository()
	pb := proxy.NewProxyBlockchainRepository()
	pm := pkgdatabase.NewPostgresMetricsRepository(database.Connection)

	// Use cases
	pu := usecases.NewProxyUsecase(cb, pb, pm)

	// HTTP API
	srv := http.Server{
		Addr:         ":" + strconv.Itoa(config.ProxyConfig.Server.Port),
		ReadTimeout:  time.Duration(config.ProxyConfig.Proxy.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.ProxyConfig.Proxy.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(config.ProxyConfig.Proxy.IdleTimeout) * time.Second,
	}
	httpController := httpinfra.NewHTTPController(pu, rp, &srv)
	httpController.Initialize()
	httpController.Run()
}

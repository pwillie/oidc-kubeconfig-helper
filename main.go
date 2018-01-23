package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/labstack/echo"
	echoMiddleware "github.com/labstack/echo/middleware"
	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pwillie/oidc-kubeconfig-helper/middleware"
	"github.com/pwillie/oidc-kubeconfig-helper/pkg/oidc"
)

func main() {
	provider := flag.String("provider", "", "OIDC provider url")
	clientid := flag.String("clientid", "", "OIDC Client ID")
	clientsecret := flag.String("clientsecret", "", "OIDC Client Secret")
	callbackurl := flag.String("callbackurl", "/callback", "Callback URL")
	listenAddress := flag.String("listen", ":8000", "Listen address")
	versionFlag := flag.Bool("version", false, "Version")

	flag.Parse()

	if *versionFlag {
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Version:", Version)
		if VersionPrerelease != "" {
			fmt.Println("Version PreRelease:", VersionPrerelease)
		}
		return
	}

	config := oidc.NewOidcConfig(clientid, clientsecret, callbackurl, provider)

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(echoMiddleware.LoggerWithConfig(echoMiddleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			if strings.HasPrefix(c.Request().URL.Path, "/internal/") {
				return true
			}
			return false
		},
	}))
	e.Use(echoMiddleware.Recover())
	e.Use(middleware.PrometheusMiddleware())

	e.GET("/", config.SigninHandler)
	e.GET("/callback", config.CallbackHandler)

	e.GET("/internal/healthz", healthzHandler)
	e.GET("/internal/metrics", echo.WrapHandler(promhttp.Handler()))

	// Start server
	go func() {
		if err := e.Start(*listenAddress); err != nil {
			e.Logger.Info("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

func healthzHandler(c echo.Context) error {
	return c.NoContent(200)
}

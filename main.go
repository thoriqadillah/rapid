package main

import (
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rapid-downloader/rapid/controllers/downloader"
	"github.com/rapid-downloader/rapid/controllers/logger"
	"github.com/rapid-downloader/rapid/controllers/settings"
	"github.com/rapid-downloader/rapid/env"
	downloaderSvc "github.com/rapid-downloader/rapid/services/downloader"
	loggerSvc "github.com/rapid-downloader/rapid/services/logger"
	settingsSvc "github.com/rapid-downloader/rapid/services/settings"
	"github.com/thoriqadillah/gema"
	"go.uber.org/fx"
)

func httpServer() *echo.Echo {
	e := echo.New()
	e.Use(middleware.Gzip())
	e.Use(middleware.Recover())

	return e
}

func main() {
	godotenv.Load()

	app := fx.New(
		gema.FxLogger,
		fx.Provide(httpServer),
		fx.Provide(loggerSvc.NewService),
		fx.Provide(settingsSvc.NewService),
		fx.Provide(downloaderSvc.NewService),
		gema.RegisterController(
			logger.NewController,
			settings.NewController,
			downloader.NewController,
		),
		gema.Start(env.PORT),
	)

	app.Run()
}

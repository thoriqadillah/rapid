package downloader

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rapid-downloader/rapid/api"
	"github.com/rapid-downloader/rapid/services/downloader"
	"github.com/rapid-downloader/rapid/services/logger"
	"github.com/thoriqadillah/gema"
)

type downloadController struct {
	service *downloader.DownloaderService
	logger  *logger.LoggerService
}

func NewController(service *downloader.DownloaderService, logger *logger.LoggerService) gema.Controller {
	return &downloadController{
		service: service,
		logger:  logger,
	}
}

func (ctrl *downloadController) fetch(c echo.Context) error {
	var params downloadParams
	if err := c.Bind(&params); err != nil {
		return err
	}

	item, err := ctrl.service.Fetch(c.Request().Context(), params.Url)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, item)
}

func (ctrl *downloadController) download(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}

	var params api.DownloadItem
	if err := c.Bind(&params); err != nil {
		return err
	}

	go func() {
		err := ctrl.service.Download(context.Background(), params, nil)
		if err != nil {
			ctrl.logger.Error(err)
		}
	}()

	return c.NoContent(http.StatusOK)
}

func (ctrl *downloadController) resume(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}

	var params api.DownloadItem
	if err := c.Bind(&params); err != nil {
		return err
	}

	go func() {
		err := ctrl.service.Resume(context.Background(), params, nil)
		if err != nil {
			ctrl.logger.Error(err)
		}
	}()

	return c.NoContent(http.StatusOK)
}

func (ctrl *downloadController) restart(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}

	var params api.DownloadItem
	if err := c.Bind(&params); err != nil {
		return err
	}

	go func() {
		err := ctrl.service.Restart(context.Background(), params, nil)
		if err != nil {
			ctrl.logger.Error(err)
		}
	}()

	return c.NoContent(http.StatusOK)
}

func (ctrl *downloadController) stop(c echo.Context) error {
	id := c.Param("id")
	err := ctrl.service.Stop(id)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (ctrl *downloadController) pause(c echo.Context) error {
	id := c.Param("id")
	err := ctrl.service.Pause(id)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (ctrl *downloadController) CreateRoutes(r *echo.Group) {
	r.POST("/downloader/fetch", ctrl.fetch)
	r.POST("/downloader/:id/download", ctrl.download)
	r.DELETE("/downloader/:id/stop", ctrl.stop)
	r.DELETE("/downloader/:id/pause", ctrl.pause)
	r.PUT("/downloader/:id/resume", ctrl.resume)
	r.PUT("/downloader/:id/restart", ctrl.restart)
}

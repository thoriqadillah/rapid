package logger

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rapid-downloader/rapid/services/logger"
	"github.com/thoriqadillah/gema"
)

type logController struct {
	service *logger.LoggerService
}

func NewController(service *logger.LoggerService) gema.Controller {
	return &logController{
		service,
	}
}

func (ctrl *logController) Init(ctx context.Context) error {
	return ctrl.service.Init(ctx)
}

func (ctrl *logController) Close(ctx context.Context) error {
	return ctrl.service.Close(ctx)
}

func (ctrl *logController) getLogs(c echo.Context) error {
	date := c.QueryParam("date")
	logEntries, err := ctrl.service.GetLogs(date)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, logEntries)
}

func (ctrl *logController) CreateRoutes(r *echo.Group) {
	r.GET("/logs", ctrl.getLogs)
}

package settings

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rapid-downloader/rapid/services/logger"
	"github.com/rapid-downloader/rapid/services/settings"
	"github.com/thoriqadillah/gema"
)

type settingController struct {
	service *settings.SettingService
	logger  *logger.LoggerService
}

func NewController(service *settings.SettingService, logger *logger.LoggerService) gema.Controller {
	return &settingController{
		service,
		logger,
	}
}

func (ctrl *settingController) Init(ctx context.Context) error {
	err := ctrl.service.Init(ctx)
	if err != nil {
		ctrl.logger.Error(err)
		return err
	}
	return nil
}

func (ctrl *settingController) getSettings(c echo.Context) error {
	setting, err := ctrl.service.Load()
	if err != nil {
		ctrl.logger.Error(err)
		return err
	}

	return c.JSON(http.StatusOK, Setting{
		DownloadLocation: setting.DownloadLocation,
		MaxRetry:         setting.MaxRetry,
		MinChunkSize:     setting.MinChunkSize,
		MaxChunkCount:    setting.MaxChunkCount,
	})
}

func (ctrl *settingController) CreateRoutes(r *echo.Group) {
	r.GET("/settings", ctrl.getSettings)
}

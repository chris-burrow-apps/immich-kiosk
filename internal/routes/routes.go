// Package routes provides HTTP endpoint handlers for the Kiosk application.
//
// It includes functions for rendering pages, handling API requests,
// and managing caching of page data. This package is responsible for
// defining the web routes and their corresponding handler functions.
package routes

import (
	"net/http"
	"sync"
	"time"

	"github.com/a-h/templ"
	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"

	"github.com/damongolding/immich-kiosk/internal/components/partials"
	"github.com/damongolding/immich-kiosk/internal/common"
	"github.com/damongolding/immich-kiosk/internal/config"
	"github.com/damongolding/immich-kiosk/internal/utils"
)

var (
	KioskVersion string

	ViewDataCache      *cache.Cache
	viewDataCacheMutex sync.Mutex

	drawFacesOnImages string
)

type PersonOrAlbum struct {
	Type string
	ID   string
}

func ShouldDrawFacesOnImages() bool {
	return drawFacesOnImages == "true"
}

func init() {
	// Setting up Immich api cache
	ViewDataCache = cache.New(5*time.Minute, 10*time.Minute)

}

// InitializeRequestData processes incoming request context and configuration to create RouteRequestData.
// It handles kiosk version checks, client configuration overrides, and request metadata.
//
// Parameters:
//   - c: Echo context containing the HTTP request and response data
//   - baseConfig: Base configuration to be used as template for request-specific config
//
// Returns:
//   - *common.RouteRequestData: Processed request data and configuration
//   - error: Any errors encountered during initialization
func InitializeRequestData(c echo.Context, baseConfig *config.Config) (*common.RouteRequestData, error) {

	kioskDeviceVersion := c.Request().Header.Get("kiosk-version")
	kioskDeviceID := c.Request().Header.Get("kiosk-device-id")
	requestID := utils.ColorizeRequestId(c.Response().Header().Get(echo.HeaderXRequestID))
	clientName := c.QueryParams().Get("client")
	if clientName == "" {
		clientName = c.FormValue("client")
	}

	// create a copy of the global config to use with this request
	requestConfig := *baseConfig

	// If kiosk version on client and server do not match refresh client.
	if kioskDeviceVersion != "" && KioskVersion != kioskDeviceVersion {
		c.Response().Header().Set("HX-Refresh", "true")
		return nil, c.NoContent(http.StatusNoContent)
	}

	err := requestConfig.ConfigWithOverrides(c)
	if err != nil {
		log.Error("Failed to initialise request data", "error", err, "path", c.Request().URL.Path)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to process request")
	}

	return &common.RouteRequestData{
		RequestConfig: requestConfig,
		DeviceID:      kioskDeviceID,
		RequestID:     requestID,
		ClientName:    clientName,
	}, nil
}

func RenderError(c echo.Context, err error, message string) error {
	log.Error(message, "err", err)
	return Render(c, http.StatusOK, partials.Error(partials.ErrorData{
		Title:   "Error " + message,
		Message: err.Error(),
	}))
}

// This custom Render replaces Echo's echo.Context.Render() with templ's templ.Component.Render().
func Render(ctx echo.Context, statusCode int, t templ.Component) error {

	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		log.Error("rendering view", "err", err)
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}
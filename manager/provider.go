package manager

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/db"
)

// RegisterRoutes 注册 Manager API 路由
func RegisterRoutes(e *echo.Echo, database *sql.DB) {
	m := e.Group("/manager")

	m.POST("/providers", func(c echo.Context) error {
		return addProvider(c, database)
	})
	m.GET("/providers", func(c echo.Context) error {
		return listProviders(c, database)
	})
	m.GET("/providers/:type", func(c echo.Context) error {
		return getProvider(c, database)
	})
	m.DELETE("/providers/:type", func(c echo.Context) error {
		return deleteProvider(c, database)
	})
}

type providerReq struct {
	Type    string `json:"type"`
	Config  string `json:"config"` // JSON: {"domain":"https://...","headers":{...}}
	Enabled *bool  `json:"enabled"`
}

func addProvider(c echo.Context, database *sql.DB) error {
	var req providerReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, constant.BuildHttpResponseFail("invalid request body"))
	}
	if req.Type == "" || req.Config == "" {
		return c.JSON(http.StatusBadRequest, constant.BuildHttpResponseFail("type and config are required"))
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	p := &db.Provider{
		Type:    req.Type,
		Config:  req.Config,
		Enabled: enabled,
	}
	if err := db.AddProvider(database, p); err != nil {
		return c.JSON(http.StatusInternalServerError, constant.BuildHttpResponseFail(err.Error()))
	}
	return c.JSON(http.StatusOK, constant.BuildHttpResponseSuccess(p))
}

func listProviders(c echo.Context, database *sql.DB) error {
	providers, err := db.ListProviders(database)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, constant.BuildHttpResponseFail(err.Error()))
	}
	return c.JSON(http.StatusOK, constant.BuildHttpResponseSuccess(providers))
}

func getProvider(c echo.Context, database *sql.DB) error {
	p, ok := db.GetProvider(database, c.Param("type"))
	if !ok {
		return c.JSON(http.StatusNotFound, constant.BuildHttpResponseFail("provider not found"))
	}
	return c.JSON(http.StatusOK, constant.BuildHttpResponseSuccess(p))
}

func deleteProvider(c echo.Context, database *sql.DB) error {
	if err := db.DeleteProvider(database, c.Param("type")); err != nil {
		return c.JSON(http.StatusInternalServerError, constant.BuildHttpResponseFail(err.Error()))
	}
	return c.JSON(http.StatusOK, constant.BuildHttpResponseSuccess(nil))
}

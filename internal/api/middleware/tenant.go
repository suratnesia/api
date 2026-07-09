package middleware

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

var tenantIDRegex = regexp.MustCompile("^[a-zA-Z0-9_]+$")

func TenantMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Request().Header.Get("X-Tenant-ID")
			if tenantID == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "X-Tenant-ID header is required")
			}

			if !tenantIDRegex.MatchString(tenantID) {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid tenant ID format")
			}

			schema := fmt.Sprintf("tenant_%s", tenantID)

			// Start a scoped database transaction to isolate connection state
			tx := db.Begin()
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
					panic(r) // Re-throw panic after rollback
				}
			}()

			// Run SET search_path inside transaction to scope it strictly to this connection instance
			if err := tx.Exec(fmt.Sprintf("SET search_path TO %s", schema)).Error; err != nil {
				tx.Rollback()
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to configure tenant context")
			}

			// Bind request-scoped tx to the Echo context
			c.Set("db", tx)

			// Process next handler
			if err := next(c); err != nil {
				tx.Rollback()
				return err
			}

			// Commit transaction if no error occurred
			if err := tx.Commit().Error; err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to commit transaction")
			}

			return nil
		}
	}
}

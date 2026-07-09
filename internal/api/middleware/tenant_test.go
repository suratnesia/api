package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"suratnesia/internal/repository"
)

func TestTenantMiddleware(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := repository.InitMockDB(sqlDB)
	assert.NoError(t, err)

	e := echo.New()

	t.Run("Missing Tenant ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mw := TenantMiddleware(gormDB)
		err := mw(func(c echo.Context) error {
			return c.String(http.StatusOK, "OK")
		})(c)

		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
		assert.Equal(t, "X-Tenant-ID header is required", he.Message)
	})

	t.Run("Invalid Tenant ID Format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Tenant-ID", "invalid-id; DROP TABLE users;")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mw := TenantMiddleware(gormDB)
		err := mw(func(c echo.Context) error {
			return c.String(http.StatusOK, "OK")
		})(c)

		assert.Error(t, err)
		he, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, he.Code)
		assert.Equal(t, "invalid tenant ID format", he.Message)
	})

	t.Run("Valid Tenant ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Tenant-ID", "pt_kai")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Expect search_path statement in transaction
		mock.ExpectBegin()
		mock.ExpectExec("SET search_path TO tenant_pt_kai").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		mw := TenantMiddleware(gormDB)
		err := mw(func(c echo.Context) error {
			// Retrieve DB from context and perform a dummy operation to trigger transaction commit
			tx, ok := c.Get("db").(*gorm.DB)
			assert.True(t, ok)
			assert.NotNil(t, tx)
			return c.String(http.StatusOK, "OK")
		})(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

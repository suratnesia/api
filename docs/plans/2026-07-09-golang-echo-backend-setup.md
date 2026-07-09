# Golang Echo Backend Setup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Set up the initial Go backend microservices structure using the Echo framework, GORM configuration, and a dynamic schema-per-tenant PostgreSQL middleware.

**Architecture:** A layered backend structure (Handlers -> Services -> Repositories) where a tenant routing middleware intercepts incoming API requests, extracts the tenant identifier from headers, validates it, and binds a request-scoped database transaction pre-configured with the tenant's PostgreSQL `search_path` schema to the Echo context.

**Tech Stack:** Go 1.25+, Echo (v4) Web Framework, GORM (Object Relational Mapper), PostgreSQL (pg driver), Testify (assertion library), and go-sqlmock (database mock testing library).

---

### Task 1: Project Initialization & Dependency Resolution

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `internal/sanity_test.go`

**Step 1: Write the failing test**
Create `internal/sanity_test.go` with a simple placeholder test that fails to verify the test suite run.

```go
package internal

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSanityCheck(t *testing.T) {
	assert.True(t, false, "Sanity check should fail initially")
}
```

**Step 2: Run test to verify it fails**
Run:
```bash
go test ./internal/... -v
```
Expected: FAIL with "Sanity check should fail initially"

**Step 3: Write minimal implementation**
Fix the test to pass and initialize modules.
Modify `internal/sanity_test.go`:
```go
package internal

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSanityCheck(t *testing.T) {
	assert.True(t, true, "Sanity check should pass")
}
```

**Step 4: Run test to verify it passes**
Run:
```bash
go test ./internal/... -v
```
Expected: PASS

**Step 5: Commit**
```bash
go mod init suratnesia
go get github.com/labstack/echo/v4
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get github.com/stretchr/testify
go get github.com/DATA-DOG/go-sqlmock
git add go.mod go.sum internal/sanity_test.go
git commit -m "chore: initialize go module and dependencies"
```

---

### Task 2: Config Management Layer

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write the failing test**
Create `internal/config/config_test.go`:
```go
package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "secret")
	os.Setenv("DB_NAME", "suratnesia_db")

	cfg := Load()

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "5432", cfg.DBPort)
	assert.Equal(t, "postgres", cfg.DBUser)
	assert.Equal(t, "secret", cfg.DBPassword)
	assert.Equal(t, "suratnesia_db", cfg.DBName)

	// Clean up env
	os.Unsetenv("PORT")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
}
```
Run `go test ./internal/config/... -v`
Expected: Compile error because `Load()` is not defined.

**Step 2: Run test to verify it fails**
Run:
```bash
go test ./internal/config/... -v
```
Expected: FAIL (compilation error)

**Step 3: Write minimal implementation**
Create `internal/config/config.go`:
```go
package config

import "os"

type Config struct {
	Port       string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	return &Config{
		Port:       port,
		DBHost:     dbHost,
		DBPort:     dbPort,
		DBUser:     dbUser,
		DBPassword: dbPassword,
		DBName:     dbName,
	}
}
```

**Step 4: Run test to verify it passes**
Run:
```bash
go test ./internal/config/... -v
```
Expected: PASS

**Step 5: Commit**
```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add config management layer"
```

---

### Task 3: Echo Web Server Setup & Health Check Route

**Files:**
- Create: `internal/api/handler/health.go`
- Create: `internal/api/handler/health_test.go`
- Create: `internal/api/router/router.go`
- Create: `internal/api/router/router_test.go`

**Step 1: Write the failing test**
Create `internal/api/handler/health_test.go`:
```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, HealthCheck(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
	}
}
```
Expected: Compile error because `HealthCheck` is not defined.

**Step 2: Run test to verify it fails**
Run:
```bash
go test ./internal/api/handler/... -v
```
Expected: FAIL (compilation error)

**Step 3: Write minimal implementation**
Create `internal/api/handler/health.go`:
```go
package handler

import (
	"net/http"
	"github.com/labstack/echo/v4"
)

func HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
```
Create `internal/api/router/router.go`:
```go
package router

import (
	"github.com/labstack/echo/v4"
	"suratnesia/internal/api/handler"
)

func New() *echo.Echo {
	e := echo.New()
	e.GET("/health", handler.HealthCheck)
	return e
}
```
Create `internal/api/router/router_test.go`:
```go
package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouterInitialization(t *testing.T) {
	r := New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}
```

**Step 4: Run test to verify it passes**
Run:
```bash
go test ./internal/api/handler/... -v
go test ./internal/api/router/... -v
```
Expected: PASS

**Step 5: Commit**
```bash
git add internal/api/handler/health.go internal/api/handler/health_test.go internal/api/router/router.go internal/api/router/router_test.go
git commit -m "feat: setup echo web server and health check route"
```

---

### Task 4: Database Connection Setup

**Files:**
- Create: `internal/repository/db.go`
- Create: `internal/repository/db_test.go`

**Step 1: Write the failing test**
Create `internal/repository/db_test.go`:
```go
package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"suratnesia/internal/config"
)

func TestInitializeDB(t *testing.T) {
	// Let's verify compilation of InitDB with config
	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "postgres",
		DBPassword: "password",
		DBName:     "postgres",
	}
	_, mock, err := sqlmock.NewWithDSN("sqlmock_db")
	assert.NoError(t, err)
	defer mock.ExpectClose()

	// GORM initializes connection with dialer. We'll verify that our wrapper handles GORM init
	db, err := InitMockDB()
	assert.NoError(t, err)
	assert.NotNil(t, db)
}
```
Expected: Compile error because `InitMockDB` or `InitDB` are not defined.

**Step 2: Run test to verify it fails**
Run:
```bash
go test ./internal/repository/... -v
```
Expected: FAIL (compilation error)

**Step 3: Write minimal implementation**
Create `internal/repository/db.go`:
```go
package repository

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"suratnesia/internal/config"
)

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// InitMockDB is a testing helper to create a GORM db connected to mock sql
func InitMockDB() (*gorm.DB, error) {
	return gorm.Open(postgres.New(postgres.Config{
		DSN:                  "sqlmock_db",
		DriverName:           "postgres",
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
}
```

**Step 4: Run test to verify it passes**
Run:
```bash
go test ./internal/repository/... -v
```
Expected: PASS

**Step 5: Commit**
```bash
git add internal/repository/db.go internal/repository/db_test.go
git commit -m "feat: add database connection setup wrapper"
```

---

### Task 5: Dynamic Tenant Routing Middleware

**Files:**
- Create: `internal/api/middleware/tenant.go`
- Create: `internal/api/middleware/tenant_test.go`

**Step 1: Write the failing test**
Create `internal/api/middleware/tenant_test.go`:
```go
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
	gormDB, mock, err := repository.InitMockDB()
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
```
Expected: Compile error because `TenantMiddleware` is not defined.

**Step 2: Run test to verify it fails**
Run:
```bash
go test ./internal/api/middleware/... -v
```
Expected: FAIL (compilation error)

**Step 3: Write minimal implementation**
Create `internal/api/middleware/tenant.go`:
```go
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
```

**Step 4: Run test to verify it passes**
Run:
```bash
go test ./internal/api/middleware/... -v
```
Expected: PASS

**Step 5: Commit**
```bash
git add internal/api/middleware/tenant.go internal/api/middleware/tenant_test.go
git commit -m "feat: add schema-per-tenant routing middleware"
```

---

### Task 6: Main Entrypoint Integration

**Files:**
- Create: `cmd/api/main.go`

**Step 1: Write the failing test**
Create an incomplete version of `cmd/api/main.go` to verify build failures.
```go
package main

func main() {
	// Missing setup, compiler check will build but verify structure
}
```

**Step 2: Run compile verification to verify it fails or is incomplete**
Run:
```bash
go build -o /dev/null ./cmd/api
```

**Step 3: Write main implementation**
Update `cmd/api/main.go` to bind all components:
```go
package main

import (
	"log"
	"os"

	"suratnesia/internal/api/middleware"
	"suratnesia/internal/api/router"
	"suratnesia/internal/config"
	"suratnesia/internal/repository"
)

func main() {
	cfg := config.Load()

	// Initialize real database pool
	db, err := repository.InitDB(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Setup server and router
	e := router.New()

	// Global Middlewares
	e.Use(middleware.TenantMiddleware(db))

	port := cfg.Port
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	log.Printf("Server starting on port %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatalf("server shut down unexpectedly: %v", err)
	}
}
```

**Step 4: Verify it builds correctly**
Run:
```bash
go build -o bin/api ./cmd/api
```
Expected: Succeeds and creates binary `bin/api`.

**Step 5: Commit**
```bash
git add cmd/api/main.go
git commit -m "feat: wire components together in api entrypoint"
```

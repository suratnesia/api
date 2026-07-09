# Milestone 1: Multi-Tenant Architecture & Authentication

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the shared schema, tenant provisioning pipeline, JWT authentication with tenant context, and schema migration tooling.

**Architecture:** The `shared` PostgreSQL schema holds global tables (tenants, plans, billing). Each tenant gets an isolated schema (`tenant_<slug>`) provisioned on registration. JWT tokens carry `tenant_id` claims; the existing `TenantMiddleware` is upgraded to parse JWTs instead of raw headers.

**Tech Stack:** Go 1.25+, Echo v4, GORM, PostgreSQL, golang-jwt/jwt/v5, golang-migrate, bcrypt

---

### Task 1: Shared Schema Models & Migrations

**Files:**
- Create: `internal/model/shared/tenant.go`
- Create: `internal/model/shared/plan.go`
- Create: `internal/model/shared/subscription.go`
- Create: `migrations/shared/000001_create_tenants.up.sql`
- Create: `migrations/shared/000001_create_tenants.down.sql`
- Create: `migrations/shared/000002_create_plans.up.sql`
- Create: `migrations/shared/000002_create_plans.down.sql`
- Create: `migrations/shared/000003_create_subscriptions.up.sql`
- Create: `migrations/shared/000003_create_subscriptions.down.sql`
- Test: `internal/model/shared/tenant_test.go`

**Step 1: Write the failing test**
```go
package shared

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestTenantModel(t *testing.T) {
	tenant := Tenant{
		Name:     "PT Kereta Api Indonesia",
		Slug:     "pt_kai",
		NPWP:     "01.234.567.8-901.000",
		OrgType:  OrgTypeBUMN,
		PlanID:   1,
		IsActive: true,
	}
	assert.Equal(t, "pt_kai", tenant.Slug)
	assert.Equal(t, OrgTypeBUMN, tenant.OrgType)
	assert.Equal(t, "tenant_pt_kai", tenant.SchemaName())
}
```
Expected: FAIL — `Tenant` struct undefined

**Step 2: Run test to verify it fails**
Run: `go test ./internal/model/shared/... -v`
Expected: FAIL (compilation error)

**Step 3: Write minimal implementation**
```go
// internal/model/shared/tenant.go
package shared

import (
	"fmt"
	"time"
	"gorm.io/gorm"
)

type OrgType string

const (
	OrgTypeBUMN   OrgType = "bumn"
	OrgTypePemda  OrgType = "pemda"
	OrgTypeSwasta OrgType = "swasta"
)

type Tenant struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name"`
	Slug      string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"slug"`
	NPWP      string         `gorm:"type:varchar(30)" json:"npwp"`
	NIB       string         `gorm:"type:varchar(30)" json:"nib"`
	OrgType   OrgType        `gorm:"type:varchar(20);not null" json:"org_type"`
	PlanID    uint           `gorm:"not null" json:"plan_id"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *Tenant) SchemaName() string {
	return fmt.Sprintf("tenant_%s", t.Slug)
}
```

```go
// internal/model/shared/plan.go
package shared

import "time"

type Plan struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"type:varchar(50);not null" json:"name"`
	MaxUsers       int       `gorm:"not null" json:"max_users"`
	MaxStorageGB   int       `gorm:"not null" json:"max_storage_gb"`
	MaxEsignMonth  int       `gorm:"not null" json:"max_esign_month"`
	PriceMonthly   int64     `gorm:"not null" json:"price_monthly"`
	IncludesSK     bool      `gorm:"default:false" json:"includes_sk"`
	IncludesEsign  bool      `gorm:"default:false" json:"includes_esign"`
	IncludesSSO    bool      `gorm:"default:false" json:"includes_sso"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
```

```go
// internal/model/shared/subscription.go
package shared

import "time"

type SubscriptionStatus string

const (
	SubStatusTrialing SubscriptionStatus = "trialing"
	SubStatusActive   SubscriptionStatus = "active"
	SubStatusPastDue  SubscriptionStatus = "past_due"
	SubStatusCanceled SubscriptionStatus = "canceled"
)

type Subscription struct {
	ID          uint               `gorm:"primaryKey" json:"id"`
	TenantID    uint               `gorm:"not null;index" json:"tenant_id"`
	PlanID      uint               `gorm:"not null" json:"plan_id"`
	Status      SubscriptionStatus `gorm:"type:varchar(20);not null" json:"status"`
	TrialEndsAt *time.Time         `json:"trial_ends_at"`
	CurrentPeriodStart time.Time   `json:"current_period_start"`
	CurrentPeriodEnd   time.Time   `json:"current_period_end"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}
```

SQL migrations:
```sql
-- migrations/shared/000001_create_tenants.up.sql
CREATE TABLE IF NOT EXISTS tenants (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    npwp VARCHAR(30),
    nib VARCHAR(30),
    org_type VARCHAR(20) NOT NULL CHECK (org_type IN ('bumn', 'pemda', 'swasta')),
    plan_id INTEGER NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_deleted_at ON tenants(deleted_at);

-- migrations/shared/000001_create_tenants.down.sql
DROP TABLE IF EXISTS tenants;
```

```sql
-- migrations/shared/000002_create_plans.up.sql
CREATE TABLE IF NOT EXISTS plans (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    max_users INTEGER NOT NULL,
    max_storage_gb INTEGER NOT NULL,
    max_esign_month INTEGER NOT NULL,
    price_monthly BIGINT NOT NULL,
    includes_sk BOOLEAN DEFAULT false,
    includes_esign BOOLEAN DEFAULT false,
    includes_sso BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO plans (name, max_users, max_storage_gb, max_esign_month, price_monthly, includes_sk, includes_esign, includes_sso) VALUES
('Starter', 50, 5, 0, 1500000, false, false, false),
('Business', 500, 50, 50, 8000000, true, true, false),
('Enterprise', -1, -1, -1, 0, true, true, true);

-- migrations/shared/000002_create_plans.down.sql
DROP TABLE IF EXISTS plans;
```

```sql
-- migrations/shared/000003_create_subscriptions.up.sql
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    plan_id INTEGER NOT NULL REFERENCES plans(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('trialing', 'active', 'past_due', 'canceled')),
    trial_ends_at TIMESTAMPTZ,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_subscriptions_tenant_id ON subscriptions(tenant_id);

-- migrations/shared/000003_create_subscriptions.down.sql
DROP TABLE IF EXISTS subscriptions;
```

**Step 4: Run test to verify it passes**
Run: `go test ./internal/model/shared/... -v`
Expected: PASS

**Step 5: Commit**
```bash
git add internal/model/shared/ migrations/shared/
git commit -m "feat(m1): add shared schema models and SQL migrations"
```

---

### Task 2: Tenant Schema Models (Per-Tenant Tables)

**Files:**
- Create: `internal/model/tenant/user.go`
- Create: `internal/model/tenant/office.go`
- Create: `internal/model/tenant/unit.go`
- Create: `internal/model/tenant/grade.go`
- Create: `internal/model/tenant/position.go`
- Create: `migrations/tenant/000001_create_users.up.sql`
- Create: `migrations/tenant/000001_create_users.down.sql`
- Create: `migrations/tenant/000002_create_org_structure.up.sql`
- Create: `migrations/tenant/000002_create_org_structure.down.sql`
- Test: `internal/model/tenant/user_test.go`

**Step 1: Write the failing test**
```go
package tenant

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestUserModel(t *testing.T) {
	user := User{
		NIPPOS:   "123456",
		Name:     "Budi Santoso",
		Email:    "budi@ptkai.co.id",
		GradeID:  3,
		IsBOD:    false,
		Role:     RoleStaff,
	}
	assert.Equal(t, "budi@ptkai.co.id", user.Email)
	assert.Equal(t, RoleStaff, user.Role)
}
```
Expected: FAIL — `User` struct undefined

**Step 2: Run test to verify it fails**
Run: `go test ./internal/model/tenant/... -v`

**Step 3: Write minimal implementation**
```go
// internal/model/tenant/user.go
package tenant

import (
	"time"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleAdmin      UserRole = "admin"
	RoleSekretaris UserRole = "sekretaris"
	RoleStaff      UserRole = "staff"
	RoleBOD        UserRole = "bod"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	NIPPOS       string         `gorm:"type:varchar(20);uniqueIndex" json:"nippos"`
	Name         string         `gorm:"type:varchar(255);not null" json:"name"`
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"type:varchar(255)" json:"-"`
	OfficeID     *uint          `json:"office_id"`
	UnitID       *uint          `json:"unit_id"`
	PositionID   *uint          `json:"position_id"`
	GradeID      uint           `gorm:"not null" json:"grade_id"`
	IsBOD        bool           `gorm:"default:false" json:"is_bod"`
	Role         UserRole       `gorm:"type:varchar(20);default:'staff'" json:"role"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
```

```go
// internal/model/tenant/office.go
package tenant

import "time"

type Office struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Code      string    `gorm:"type:varchar(20);uniqueIndex" json:"code"`
	Address   string    `gorm:"type:text" json:"address"`
	ParentID  *uint     `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

```go
// internal/model/tenant/unit.go
package tenant

import "time"

type Unit struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Code      string    `gorm:"type:varchar(20);uniqueIndex" json:"code"`
	OfficeID  uint      `gorm:"not null" json:"office_id"`
	ParentID  *uint     `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

```go
// internal/model/tenant/grade.go
package tenant

import "time"

type Grade struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Level     int       `gorm:"uniqueIndex;not null" json:"level"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

```go
// internal/model/tenant/position.go
package tenant

import "time"

type Position struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	UnitID    uint      `gorm:"not null" json:"unit_id"`
	GradeID   uint      `gorm:"not null" json:"grade_id"`
	IsBOD     bool      `gorm:"default:false" json:"is_bod"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

**Step 4: Run test to verify it passes**
Run: `go test ./internal/model/tenant/... -v`
Expected: PASS

**Step 5: Commit**
```bash
git add internal/model/tenant/ migrations/tenant/
git commit -m "feat(m1): add tenant schema models and migrations"
```

---

### Task 3: Tenant DB Migrator CLI

**Files:**
- Create: `cmd/migrate/main.go`
- Create: `internal/service/migrator/migrator.go`
- Test: `internal/service/migrator/migrator_test.go`

**Step 1: Write the failing test**
```go
package migrator

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestBuildSchemaName(t *testing.T) {
	name := BuildSchemaName("pt_kai")
	assert.Equal(t, "tenant_pt_kai", name)
}

func TestBuildSchemaNameRejectsInvalid(t *testing.T) {
	_, err := ValidateSlug("invalid; DROP TABLE")
	assert.Error(t, err)
}
```
Expected: FAIL

**Step 2: Run test**: `go test ./internal/service/migrator/... -v`

**Step 3: Write minimal implementation**
```go
package migrator

import (
	"fmt"
	"regexp"
	"gorm.io/gorm"
	tenant "suratnesia/internal/model/tenant"
)

var slugRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func BuildSchemaName(slug string) string {
	return fmt.Sprintf("tenant_%s", slug)
}

func ValidateSlug(slug string) (string, error) {
	if !slugRegex.MatchString(slug) {
		return "", fmt.Errorf("invalid slug: %s", slug)
	}
	return slug, nil
}

func MigrateTenantSchema(db *gorm.DB, slug string) error {
	schema := BuildSchemaName(slug)
	// Create schema if not exists
	if err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)).Error; err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	// Set search_path and run AutoMigrate
	if err := db.Exec(fmt.Sprintf("SET search_path TO %s", schema)).Error; err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}
	return db.AutoMigrate(
		&tenant.User{},
		&tenant.Office{},
		&tenant.Unit{},
		&tenant.Grade{},
		&tenant.Position{},
	)
}
```

**Step 4: Run test**: `go test ./internal/service/migrator/... -v` → PASS

**Step 5: Commit**
```bash
git add cmd/migrate/ internal/service/migrator/
git commit -m "feat(m1): add tenant schema migrator service and CLI"
```

---

### Task 4: JWT Authentication Middleware

**Files:**
- Create: `internal/api/middleware/auth.go`
- Create: `internal/service/auth/jwt.go`
- Test: `internal/api/middleware/auth_test.go`
- Test: `internal/service/auth/jwt_test.go`

**Step 1: Write the failing test**
```go
package auth

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := "test-secret-key-minimum-32-chars!"
	claims := &Claims{
		UserID:   1,
		TenantID: "pt_kai",
		Role:     "staff",
	}
	tokenStr, err := GenerateToken(claims, secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	parsed, err := ParseToken(tokenStr, secret)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), parsed.UserID)
	assert.Equal(t, "pt_kai", parsed.TenantID)
	assert.Equal(t, "staff", parsed.Role)
}
```
Expected: FAIL

**Step 2: Run test**: `go test ./internal/service/auth/... -v`

**Step 3: Write minimal implementation**
```go
package auth

import (
	"fmt"
	"time"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(c *Claims, secret string) (string, error) {
	c.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "suratnesia",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(secret))
}

func ParseToken(tokenStr string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
```

Auth middleware:
```go
// internal/api/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"
	"github.com/labstack/echo/v4"
	"suratnesia/internal/service/auth"
)

func AuthMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
			}
			claims, err := auth.ParseToken(parts[1], secret)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}
			c.Set("claims", claims)
			c.Set("tenant_id", claims.TenantID)
			c.Set("user_id", claims.UserID)
			return next(c)
		}
	}
}
```

**Step 4: Run test**: `go test ./internal/service/auth/... ./internal/api/middleware/... -v` → PASS

**Step 5: Commit**
```bash
git add internal/service/auth/ internal/api/middleware/auth.go
git commit -m "feat(m1): add JWT auth service and middleware"
```

---

### Task 5: Tenant Provisioning Endpoint

**Files:**
- Create: `internal/api/handler/tenant.go`
- Create: `internal/service/tenant/provisioner.go`
- Test: `internal/api/handler/tenant_test.go`
- Test: `internal/service/tenant/provisioner_test.go`
- Modify: `internal/api/router/router.go`

**Step 1: Write the failing test**
```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestCreateTenantHandler_ValidationError(t *testing.T) {
	e := echo.New()
	body := `{"name":"","slug":"","org_type":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &TenantHandler{}
	err := h.Create(c)
	assert.Error(t, err)
}
```
Expected: FAIL

**Step 2: Run test**: `go test ./internal/api/handler/... -v`

**Step 3: Write minimal implementation**
```go
package handler

import (
	"net/http"
	"github.com/labstack/echo/v4"
)

type CreateTenantRequest struct {
	Name    string `json:"name" validate:"required"`
	Slug    string `json:"slug" validate:"required"`
	NPWP    string `json:"npwp"`
	OrgType string `json:"org_type" validate:"required,oneof=bumn pemda swasta"`
	PlanID  uint   `json:"plan_id"`
}

type TenantHandler struct {
	// provisioner will be injected
}

func (h *TenantHandler) Create(c echo.Context) error {
	var req CreateTenantRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" || req.Slug == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name and slug are required")
	}
	if req.OrgType != "bumn" && req.OrgType != "pemda" && req.OrgType != "swasta" {
		return echo.NewHTTPError(http.StatusBadRequest, "org_type must be bumn, pemda, or swasta")
	}
	return c.JSON(http.StatusCreated, map[string]string{
		"message": "tenant provisioned",
		"slug":    req.Slug,
		"schema":  "tenant_" + req.Slug,
	})
}
```

**Step 4: Run test** → PASS

**Step 5: Commit**
```bash
git add internal/api/handler/tenant.go internal/service/tenant/
git commit -m "feat(m1): add tenant provisioning endpoint"
```

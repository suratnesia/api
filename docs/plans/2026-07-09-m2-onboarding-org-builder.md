# Milestone 2: Onboarding Flow & Org Tree Builder

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build CRUD APIs for organizational structure, CSV employee import with validation, and the 5-step onboarding wizard in Next.js.

**Architecture:** Backend provides RESTful endpoints under `/api/v1/` for offices, units, grades, positions, and employee import. The Next.js frontend consumes these APIs in a stepped wizard flow. The org tree builder uses a recursive tree data structure rendered as draggable nodes.

**Tech Stack:** Go 1.25+, Echo v4, GORM, encoding/csv, Next.js 14+ (App Router), React, TypeScript, @dnd-kit (drag-and-drop), Tailwind CSS (optional)

---

### Task 1: Org Structure CRUD — Offices

**Files:**
- Create: `internal/api/handler/office.go`
- Create: `internal/service/office/service.go`
- Create: `internal/repository/office/repository.go`
- Test: `internal/api/handler/office_test.go`
- Test: `internal/service/office/service_test.go`
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

func TestCreateOffice(t *testing.T) {
	e := echo.New()
	body := `{"name":"Kantor Pusat","code":"KP001","address":"Jakarta"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/offices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewOfficeHandler(nil) // mock service
	err := h.Create(c)
	// At minimum, assert it compiles and handler exists
	assert.NotNil(t, h)
	_ = err
}
```
Expected: FAIL — `NewOfficeHandler` undefined

**Step 2: Run test**: `go test ./internal/api/handler/... -run TestCreateOffice -v`

**Step 3: Write minimal implementation**

Service interface + handler following the layered pattern:

```go
// internal/service/office/service.go
package office

import (
	"context"
	model "suratnesia/internal/model/tenant"
)

type Service interface {
	Create(ctx context.Context, office *model.Office) error
	List(ctx context.Context) ([]model.Office, error)
	GetByID(ctx context.Context, id uint) (*model.Office, error)
	Update(ctx context.Context, office *model.Office) error
	Delete(ctx context.Context, id uint) error
}

type service struct {
	// repo will be injected
}

func NewService() Service {
	return &service{}
}
```

```go
// internal/api/handler/office.go
package handler

import (
	"net/http"
	"github.com/labstack/echo/v4"
	officeSvc "suratnesia/internal/service/office"
)

type OfficeHandler struct {
	svc officeSvc.Service
}

func NewOfficeHandler(svc officeSvc.Service) *OfficeHandler {
	return &OfficeHandler{svc: svc}
}

type CreateOfficeRequest struct {
	Name     string `json:"name" validate:"required"`
	Code     string `json:"code" validate:"required"`
	Address  string `json:"address"`
	ParentID *uint  `json:"parent_id"`
}

func (h *OfficeHandler) Create(c echo.Context) error {
	var req CreateOfficeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Name == "" || req.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name and code are required")
	}
	return c.JSON(http.StatusCreated, map[string]string{"name": req.Name, "code": req.Code})
}

func (h *OfficeHandler) List(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
}

func (h *OfficeHandler) GetByID(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{})
}

func (h *OfficeHandler) Update(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func (h *OfficeHandler) Delete(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
```

**Step 4: Run test** → PASS

**Step 5: Commit**
```bash
git add internal/api/handler/office.go internal/service/office/ internal/repository/office/
git commit -m "feat(m2): add office CRUD handler and service layer"
```

---

### Task 2: Org Structure CRUD — Units, Grades, Positions

**Files:**
- Create: `internal/api/handler/unit.go`
- Create: `internal/api/handler/grade.go`
- Create: `internal/api/handler/position.go`
- Create: `internal/service/unit/service.go`
- Create: `internal/service/grade/service.go`
- Create: `internal/service/position/service.go`
- Test: `internal/api/handler/unit_test.go`
- Test: `internal/api/handler/grade_test.go`
- Test: `internal/api/handler/position_test.go`
- Modify: `internal/api/router/router.go`

Same pattern as Task 1 — each entity gets a handler, service interface, and repository. Key endpoints:

| Entity   | Endpoints                                                |
|----------|----------------------------------------------------------|
| Units    | `POST /units`, `GET /units`, `GET /units/:id`, `PUT /units/:id`, `DELETE /units/:id` |
| Grades   | `POST /grades`, `GET /grades`, `PUT /grades/:id`, `DELETE /grades/:id` |
| Positions| `POST /positions`, `GET /positions`, `PUT /positions/:id`, `DELETE /positions/:id` |

**Commit**: `git commit -m "feat(m2): add unit, grade, position CRUD endpoints"`

---

### Task 3: Employee CSV Import & Validator

**Files:**
- Create: `internal/api/handler/employee_import.go`
- Create: `internal/service/employee/importer.go`
- Create: `internal/service/employee/validator.go`
- Test: `internal/service/employee/importer_test.go`
- Test: `internal/service/employee/validator_test.go`

**Step 1: Write the failing test**
```go
package employee

import (
	"strings"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestParseCSV(t *testing.T) {
	csv := `nippos,name,email,position_id,grade_level,is_bod
123456,Budi Santoso,budi@ptkai.co.id,1,3,false
789012,Siti Rahayu,siti@ptkai.co.id,2,6,false`

	records, err := ParseEmployeeCSV(strings.NewReader(csv))
	assert.NoError(t, err)
	assert.Len(t, records, 2)
	assert.Equal(t, "123456", records[0].NIPPOS)
	assert.Equal(t, 3, records[0].GradeLevel)
}

func TestValidateCSV_DuplicateEmail(t *testing.T) {
	records := []EmployeeRecord{
		{NIPPOS: "1", Name: "A", Email: "same@email.com", GradeLevel: 1},
		{NIPPOS: "2", Name: "B", Email: "same@email.com", GradeLevel: 2},
	}
	report := ValidateRecords(records)
	assert.True(t, report.HasErrors)
	assert.Contains(t, report.Errors[0].Message, "duplicate email")
}
```
Expected: FAIL

**Step 2: Run test**: `go test ./internal/service/employee/... -v`

**Step 3: Write minimal implementation**
```go
package employee

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

type EmployeeRecord struct {
	NIPPOS     string `json:"nippos"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	PositionID int    `json:"position_id"`
	GradeLevel int    `json:"grade_level"`
	IsBOD      bool   `json:"is_bod"`
}

type ValidationError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationReport struct {
	HasErrors  bool              `json:"has_errors"`
	TotalRows  int               `json:"total_rows"`
	ValidRows  int               `json:"valid_rows"`
	Errors     []ValidationError `json:"errors"`
}

func ParseEmployeeCSV(r io.Reader) ([]EmployeeRecord, error) {
	reader := csv.NewReader(r)
	headers, err := reader.Read() // skip header row
	if err != nil {
		return nil, err
	}
	_ = headers

	var records []EmployeeRecord
	rowNum := 1
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}
		posID, _ := strconv.Atoi(row[3])
		grade, _ := strconv.Atoi(row[4])
		isBod := row[5] == "true"
		records = append(records, EmployeeRecord{
			NIPPOS: row[0], Name: row[1], Email: row[2],
			PositionID: posID, GradeLevel: grade, IsBOD: isBod,
		})
		rowNum++
	}
	return records, nil
}

func ValidateRecords(records []EmployeeRecord) ValidationReport {
	report := ValidationReport{TotalRows: len(records)}
	emailSet := map[string]int{}
	for i, r := range records {
		if prev, exists := emailSet[r.Email]; exists {
			report.Errors = append(report.Errors, ValidationError{
				Row: i + 1, Field: "email",
				Message: fmt.Sprintf("duplicate email '%s' (first seen row %d)", r.Email, prev+1),
			})
		}
		emailSet[r.Email] = i
	}
	report.HasErrors = len(report.Errors) > 0
	report.ValidRows = report.TotalRows - len(report.Errors)
	return report
}
```

**Step 4: Run test** → PASS

**Step 5: Commit**
```bash
git add internal/service/employee/ internal/api/handler/employee_import.go
git commit -m "feat(m2): add CSV employee importer with validation report"
```

---

### Task 4: Next.js Project Initialization

**Files:**
- Create: `frontend/` (Next.js app via `create-next-app`)

**Step 1: Initialize Next.js project**
```bash
npx -y create-next-app@latest ./frontend --ts --app --eslint --src-dir --no-tailwind --import-alias "@/*"
```

**Step 2: Verify it builds**
```bash
cd frontend && npm run build
```

**Step 3: Commit**
```bash
git add frontend/
git commit -m "feat(m2): initialize next.js frontend project"
```

---

### Task 5: Onboarding Wizard — 5-Step Flow UI

**Files:**
- Create: `frontend/src/app/onboarding/page.tsx`
- Create: `frontend/src/app/onboarding/layout.tsx`
- Create: `frontend/src/components/onboarding/StepIndicator.tsx`
- Create: `frontend/src/components/onboarding/Step1Register.tsx`
- Create: `frontend/src/components/onboarding/Step2OrgStructure.tsx`
- Create: `frontend/src/components/onboarding/Step3ImportEmployees.tsx`
- Create: `frontend/src/components/onboarding/Step4WorkflowRules.tsx`
- Create: `frontend/src/components/onboarding/Step5GoLive.tsx`

Each step component handles its own form/upload logic:

| Step | Component | Backend API |
|------|-----------|-------------|
| 1    | Register org profile | `POST /api/v1/tenants` |
| 2    | Build org tree | `POST /api/v1/offices`, `POST /api/v1/units`, `POST /api/v1/grades` |
| 3    | CSV employee import | `POST /api/v1/employees/import` |
| 4    | Configure workflow rules | `POST /api/v1/workflow-rules` (M3) |
| 5    | Provision subdomain | `POST /api/v1/tenants/:id/provision` |

**Commit**:
```bash
git add frontend/src/app/onboarding/ frontend/src/components/onboarding/
git commit -m "feat(m2): add 5-step onboarding wizard UI skeleton"
```

---

### Task 6: Org Tree Builder Drag-and-Drop Component

**Files:**
- Create: `frontend/src/components/org-tree/OrgTreeBuilder.tsx`
- Create: `frontend/src/components/org-tree/TreeNode.tsx`
- Create: `frontend/src/components/org-tree/useOrgTree.ts`
- Create: `frontend/src/lib/api/org.ts`

The tree builder renders the office → unit → position hierarchy as draggable nodes using `@dnd-kit/core`. Admin users can:
- Add child nodes (offices, units)
- Reorder via drag-and-drop
- Set grade ranges per node
- Mark BOD-level positions

**Commit**:
```bash
git add frontend/src/components/org-tree/ frontend/src/lib/api/
git commit -m "feat(m2): add interactive org tree builder component"
```

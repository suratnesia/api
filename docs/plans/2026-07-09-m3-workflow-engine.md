# Milestone 3: Dynamic Rule-Based Workflow Engine

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a configurable workflow routing engine where each tenant defines their own grade-based approval chains, replacing any hardcoded routing logic.

**Architecture:** Workflow rules are stored as rows in a per-tenant `workflow_rules` table. The `RoutingResolver` service receives a sender and recipient, evaluates matching rules, and returns an ordered approval chain. The frontend provides a visual rule builder with a live chain simulator.

**Tech Stack:** Go 1.25+, Echo v4, GORM, Next.js, React Flow (for chain visualization)

---

### Task 1: Workflow Rules DB Schema & Model

**Files:**
- Create: `internal/model/tenant/workflow_rule.go`
- Create: `internal/model/tenant/workflow_step.go`
- Create: `migrations/tenant/000003_create_workflow_rules.up.sql`
- Create: `migrations/tenant/000003_create_workflow_rules.down.sql`
- Test: `internal/model/tenant/workflow_rule_test.go`

**Step 1: Write the failing test**
```go
package tenant

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowRuleModel(t *testing.T) {
	rule := WorkflowRule{
		Name:             "Staff to Director",
		SenderGradeMin:   1,
		SenderGradeMax:   3,
		RecipientGradeMin: 7,
		RecipientGradeMax: 9,
		DocumentType:     DocTypeNDE,
		IsActive:         true,
	}
	assert.Equal(t, "Staff to Director", rule.Name)
	assert.Equal(t, DocTypeNDE, rule.DocumentType)
}
```
Expected: FAIL

**Step 2: Run test**: `go test ./internal/model/tenant/... -run TestWorkflowRule -v`

**Step 3: Write minimal implementation**
```go
// internal/model/tenant/workflow_rule.go
package tenant

import "time"

type DocumentType string

const (
	DocTypeNDE DocType = "nde"
	DocTypeSK  DocType = "sk"
	DocTypeAll DocType = "all"
)

type WorkflowRule struct {
	ID                uint         `gorm:"primaryKey" json:"id"`
	Name              string       `gorm:"type:varchar(255);not null" json:"name"`
	SenderGradeMin    int          `gorm:"not null" json:"sender_grade_min"`
	SenderGradeMax    int          `gorm:"not null" json:"sender_grade_max"`
	RecipientGradeMin int          `gorm:"not null" json:"recipient_grade_min"`
	RecipientGradeMax int          `gorm:"not null" json:"recipient_grade_max"`
	DocumentType      DocumentType `gorm:"type:varchar(20);default:'all'" json:"document_type"`
	Priority          int          `gorm:"default:0" json:"priority"`
	IsActive          bool         `gorm:"default:true" json:"is_active"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
	Steps             []WorkflowStep `gorm:"foreignKey:RuleID" json:"steps"`
}

// internal/model/tenant/workflow_step.go
type WorkflowStep struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	RuleID     uint   `gorm:"not null;index" json:"rule_id"`
	StepOrder  int    `gorm:"not null" json:"step_order"`
	GradeLevel int    `gorm:"not null" json:"grade_level"`
	RoleName   string `gorm:"type:varchar(100)" json:"role_name"`
}
```

SQL migration:
```sql
-- migrations/tenant/000003_create_workflow_rules.up.sql
CREATE TABLE IF NOT EXISTS workflow_rules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    sender_grade_min INTEGER NOT NULL,
    sender_grade_max INTEGER NOT NULL,
    recipient_grade_min INTEGER NOT NULL,
    recipient_grade_max INTEGER NOT NULL,
    document_type VARCHAR(20) DEFAULT 'all',
    priority INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workflow_steps (
    id SERIAL PRIMARY KEY,
    rule_id INTEGER NOT NULL REFERENCES workflow_rules(id) ON DELETE CASCADE,
    step_order INTEGER NOT NULL,
    grade_level INTEGER NOT NULL,
    role_name VARCHAR(100)
);
CREATE INDEX idx_workflow_steps_rule_id ON workflow_steps(rule_id);
```

**Step 4: Run test** → PASS

**Step 5: Commit**
```bash
git add internal/model/tenant/workflow_rule.go internal/model/tenant/workflow_step.go migrations/tenant/
git commit -m "feat(m3): add workflow rules schema and models"
```

---

### Task 2: Workflow Rules CRUD API

**Files:**
- Create: `internal/api/handler/workflow_rule.go`
- Create: `internal/service/workflow/rule_service.go`
- Create: `internal/repository/workflow/rule_repository.go`
- Test: `internal/api/handler/workflow_rule_test.go`
- Modify: `internal/api/router/router.go`

Endpoints:
| Method | Path | Description |
|--------|------|-------------|
| `POST`   | `/api/v1/workflow-rules`      | Create a new rule with steps |
| `GET`    | `/api/v1/workflow-rules`      | List all rules for the tenant |
| `GET`    | `/api/v1/workflow-rules/:id`  | Get rule details with steps |
| `PUT`    | `/api/v1/workflow-rules/:id`  | Update rule and steps |
| `DELETE` | `/api/v1/workflow-rules/:id`  | Soft-delete a rule |

**Commit**: `git commit -m "feat(m3): add workflow rules CRUD endpoints"`

---

### Task 3: Routing Resolver Service (Core Engine)

**Files:**
- Create: `internal/service/workflow/resolver.go`
- Test: `internal/service/workflow/resolver_test.go`

**Step 1: Write the failing test**
```go
package workflow

import (
	"testing"
	"github.com/stretchr/testify/assert"
	model "suratnesia/internal/model/tenant"
)

func TestResolveApprovalChain(t *testing.T) {
	rules := []model.WorkflowRule{
		{
			SenderGradeMin: 1, SenderGradeMax: 3,
			RecipientGradeMin: 7, RecipientGradeMax: 9,
			DocumentType: "all", IsActive: true, Priority: 10,
			Steps: []model.WorkflowStep{
				{StepOrder: 1, GradeLevel: 4, RoleName: "Kasubag"},
				{StepOrder: 2, GradeLevel: 6, RoleName: "Kabag"},
				{StepOrder: 3, GradeLevel: 8, RoleName: "Direktur"},
			},
		},
	}

	resolver := NewResolver(rules)

	// Sender is grade 2 (Staf), recipient is grade 8 (Direktur)
	chain, err := resolver.GetApprovalChain(2, 8, "nde")
	assert.NoError(t, err)
	assert.Len(t, chain, 3)
	assert.Equal(t, "Kasubag", chain[0].RoleName)
	assert.Equal(t, "Kabag", chain[1].RoleName)
	assert.Equal(t, "Direktur", chain[2].RoleName)
}

func TestResolveApprovalChain_NoMatchingRule(t *testing.T) {
	resolver := NewResolver([]model.WorkflowRule{})
	chain, err := resolver.GetApprovalChain(5, 6, "nde")
	assert.NoError(t, err)
	assert.Len(t, chain, 0) // direct delivery, no intermediaries
}

func TestResolveApprovalChain_PriorityOrdering(t *testing.T) {
	rules := []model.WorkflowRule{
		{SenderGradeMin: 1, SenderGradeMax: 5, RecipientGradeMin: 6, RecipientGradeMax: 9,
			DocumentType: "all", IsActive: true, Priority: 1,
			Steps: []model.WorkflowStep{{StepOrder: 1, GradeLevel: 6, RoleName: "Generic"}}},
		{SenderGradeMin: 1, SenderGradeMax: 3, RecipientGradeMin: 7, RecipientGradeMax: 9,
			DocumentType: "nde", IsActive: true, Priority: 10,
			Steps: []model.WorkflowStep{{StepOrder: 1, GradeLevel: 4, RoleName: "Specific"}}},
	}
	resolver := NewResolver(rules)
	chain, _ := resolver.GetApprovalChain(2, 8, "nde")
	// Higher priority rule should win
	assert.Equal(t, "Specific", chain[0].RoleName)
}
```
Expected: FAIL

**Step 2: Run test**: `go test ./internal/service/workflow/... -v`

**Step 3: Write minimal implementation**
```go
package workflow

import (
	"sort"
	model "suratnesia/internal/model/tenant"
)

type ChainStep struct {
	GradeLevel int    `json:"grade_level"`
	RoleName   string `json:"role_name"`
	StepOrder  int    `json:"step_order"`
}

type Resolver struct {
	rules []model.WorkflowRule
}

func NewResolver(rules []model.WorkflowRule) *Resolver {
	return &Resolver{rules: rules}
}

func (r *Resolver) GetApprovalChain(senderGrade, recipientGrade int, docType string) ([]ChainStep, error) {
	var matched []model.WorkflowRule
	for _, rule := range r.rules {
		if !rule.IsActive {
			continue
		}
		if senderGrade < rule.SenderGradeMin || senderGrade > rule.SenderGradeMax {
			continue
		}
		if recipientGrade < rule.RecipientGradeMin || recipientGrade > rule.RecipientGradeMax {
			continue
		}
		if string(rule.DocumentType) != "all" && string(rule.DocumentType) != docType {
			continue
		}
		matched = append(matched, rule)
	}
	if len(matched) == 0 {
		return []ChainStep{}, nil
	}
	// Pick highest priority rule
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Priority > matched[j].Priority
	})
	best := matched[0]
	// Sort steps by order
	sort.Slice(best.Steps, func(i, j int) bool {
		return best.Steps[i].StepOrder < best.Steps[j].StepOrder
	})
	var chain []ChainStep
	for _, s := range best.Steps {
		chain = append(chain, ChainStep{
			GradeLevel: s.GradeLevel,
			RoleName:   s.RoleName,
			StepOrder:  s.StepOrder,
		})
	}
	return chain, nil
}
```

**Step 4: Run test** → PASS

**Step 5: Commit**
```bash
git add internal/service/workflow/resolver.go internal/service/workflow/resolver_test.go
git commit -m "feat(m3): implement grade-aware routing resolver engine"
```

---

### Task 4: Chain Preview API Endpoint

**Files:**
- Create: `internal/api/handler/workflow_preview.go`
- Test: `internal/api/handler/workflow_preview_test.go`
- Modify: `internal/api/router/router.go`

Endpoint: `POST /api/v1/workflow-rules/preview`

Request body:
```json
{
  "sender_grade": 2,
  "recipient_grade": 8,
  "document_type": "nde"
}
```

Response:
```json
{
  "chain": [
    {"step_order": 1, "grade_level": 4, "role_name": "Kasubag"},
    {"step_order": 2, "grade_level": 6, "role_name": "Kabag"},
    {"step_order": 3, "grade_level": 8, "role_name": "Direktur"}
  ],
  "rule_matched": "Staff to Director"
}
```

**Commit**: `git commit -m "feat(m3): add workflow chain preview endpoint"`

---

### Task 5: Interactive Rule Builder UI (Frontend)

**Files:**
- Create: `frontend/src/app/admin/workflow-rules/page.tsx`
- Create: `frontend/src/components/workflow/RuleBuilder.tsx`
- Create: `frontend/src/components/workflow/ChainPreview.tsx`
- Create: `frontend/src/components/workflow/StepEditor.tsx`
- Create: `frontend/src/lib/api/workflow.ts`

The rule builder allows admins to:
- Set sender/recipient grade ranges using sliders
- Add ordered verification steps (grade + role name)
- Select document type filter (NDE / SK / All)
- See a **live chain preview** that calls the preview API on each change

**Commit**: `git commit -m "feat(m3): add interactive workflow rule builder UI with live chain preview"`

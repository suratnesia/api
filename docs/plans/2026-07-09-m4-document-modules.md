# Milestone 4: Core Document Modules (Composer & Disposisi)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the document lifecycle — composing NDE/SK documents, routing them through approval chains, delegating via disposisi, and logging an immutable audit trail.

**Architecture:** Documents are created as drafts, submitted to the workflow engine for approval routing, signed, then archived. Each state transition is logged in the audit trail. Disposisi is a separate flow where a superior assigns a directive to a subordinate with a deadline.

**Tech Stack:** Go 1.25+, Echo v4, GORM, PostgreSQL, crypto/sha256 (audit hash chain), Next.js, TipTap (rich-text editor)

---

### Task 1: Document Model & State Machine

**Files:**
- Create: `internal/model/tenant/document.go`
- Create: `internal/model/tenant/attachment.go`
- Create: `internal/service/document/state_machine.go`
- Create: `migrations/tenant/000004_create_documents.up.sql`
- Create: `migrations/tenant/000004_create_documents.down.sql`
- Test: `internal/model/tenant/document_test.go`
- Test: `internal/service/document/state_machine_test.go`

**Step 1: Write the failing test**
```go
package document

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestValidTransitions(t *testing.T) {
	sm := NewStateMachine()

	assert.True(t, sm.CanTransition(StateDraft, StateSubmitted))
	assert.True(t, sm.CanTransition(StateSubmitted, StateReviewing))
	assert.True(t, sm.CanTransition(StateReviewing, StateSigned))
	assert.True(t, sm.CanTransition(StateSigned, StateArchived))

	// Invalid transitions
	assert.False(t, sm.CanTransition(StateDraft, StateSigned))
	assert.False(t, sm.CanTransition(StateArchived, StateDraft))
	assert.True(t, sm.CanTransition(StateReviewing, StateRejected))
	assert.True(t, sm.CanTransition(StateRejected, StateDraft))
}
```
Expected: FAIL

**Step 2: Run test**: `go test ./internal/service/document/... -v`

**Step 3: Write minimal implementation**
```go
// internal/model/tenant/document.go
package tenant

import (
	"time"
	"gorm.io/gorm"
)

type DocumentState string

const (
	StateDraft     DocumentState = "draft"
	StateSubmitted DocumentState = "submitted"
	StateReviewing DocumentState = "reviewing"
	StateSigned    DocumentState = "signed"
	StateRejected  DocumentState = "rejected"
	StateArchived  DocumentState = "archived"
)

type Document struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	DocumentType  DocumentType   `gorm:"type:varchar(20);not null" json:"document_type"`
	Number        string         `gorm:"type:varchar(100)" json:"number"`
	Subject       string         `gorm:"type:varchar(500);not null" json:"subject"`
	Body          string         `gorm:"type:text" json:"body"`
	State         DocumentState  `gorm:"type:varchar(20);default:'draft'" json:"state"`
	SenderID      uint           `gorm:"not null" json:"sender_id"`
	RecipientID   uint           `gorm:"not null" json:"recipient_id"`
	CurrentStepID *uint          `json:"current_step_id"`
	Priority      string         `gorm:"type:varchar(20);default:'normal'" json:"priority"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Attachments   []Attachment   `gorm:"foreignKey:DocumentID" json:"attachments"`
}

// internal/model/tenant/attachment.go
type Attachment struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DocumentID uint      `gorm:"not null;index" json:"document_id"`
	FileName   string    `gorm:"type:varchar(255);not null" json:"file_name"`
	FileURL    string    `gorm:"type:text;not null" json:"file_url"`
	FileSize   int64     `json:"file_size"`
	MimeType   string    `gorm:"type:varchar(100)" json:"mime_type"`
	CreatedAt  time.Time `json:"created_at"`
}
```

```go
// internal/service/document/state_machine.go
package document

import model "suratnesia/internal/model/tenant"

type StateMachine struct {
	transitions map[model.DocumentState][]model.DocumentState
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		transitions: map[model.DocumentState][]model.DocumentState{
			model.StateDraft:     {model.StateSubmitted},
			model.StateSubmitted: {model.StateReviewing},
			model.StateReviewing: {model.StateSigned, model.StateRejected},
			model.StateSigned:    {model.StateArchived},
			model.StateRejected:  {model.StateDraft},
		},
	}
}

func (sm *StateMachine) CanTransition(from, to model.DocumentState) bool {
	allowed, exists := sm.transitions[from]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
```

**Step 4: Run test** → PASS

**Step 5: Commit**
```bash
git add internal/model/tenant/document.go internal/model/tenant/attachment.go internal/service/document/ migrations/tenant/
git commit -m "feat(m4): add document model with state machine"
```

---

### Task 2: Document CRUD & Submission API

**Files:**
- Create: `internal/api/handler/document.go`
- Create: `internal/service/document/service.go`
- Create: `internal/repository/document/repository.go`
- Test: `internal/api/handler/document_test.go`
- Modify: `internal/api/router/router.go`

Endpoints:
| Method | Path | Description |
|--------|------|-------------|
| `POST`   | `/api/v1/documents`             | Create draft document |
| `GET`    | `/api/v1/documents`             | List documents (with filters) |
| `GET`    | `/api/v1/documents/:id`         | Get document details |
| `PUT`    | `/api/v1/documents/:id`         | Update draft document |
| `POST`   | `/api/v1/documents/:id/submit`  | Submit for approval (triggers workflow) |
| `POST`   | `/api/v1/documents/:id/approve` | Approve at current step |
| `POST`   | `/api/v1/documents/:id/reject`  | Reject with reason |

Submit flow: handler calls `StateMachine.CanTransition()` → calls `Resolver.GetApprovalChain()` → creates `DocumentApproval` records for each step → moves state to `submitted`.

**Commit**: `git commit -m "feat(m4): add document CRUD and approval submission API"`

---

### Task 3: Disposisi (Delegation) Model & API

**Files:**
- Create: `internal/model/tenant/disposisi.go`
- Create: `internal/api/handler/disposisi.go`
- Create: `internal/service/disposisi/service.go`
- Create: `migrations/tenant/000005_create_dispositions.up.sql`
- Test: `internal/api/handler/disposisi_test.go`

**Step 1: Write the failing test**
```go
package tenant

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestDisposisiModel(t *testing.T) {
	deadline := time.Now().Add(72 * time.Hour)
	d := Disposisi{
		DocumentID:  1,
		FromUserID:  10,
		ToUserID:    20,
		Directive:   DirectiveTindaklanjuti,
		Deadline:    &deadline,
		Notes:       "Segera ditindaklanjuti",
		IsRead:      false,
	}
	assert.Equal(t, DirectiveTindaklanjuti, d.Directive)
	assert.False(t, d.IsRead)
}
```

**Step 3: Write minimal implementation**
```go
// internal/model/tenant/disposisi.go
package tenant

import "time"

type DirectiveType string

const (
	DirectiveTindaklanjuti DirectiveType = "tindaklanjuti"
	DirectiveHadirMewakili DirectiveType = "hadir_mewakili"
	DirectivePelajari      DirectiveType = "pelajari"
	DirectiveKoordinasi    DirectiveType = "koordinasi"
	DirectiveArsip         DirectiveType = "arsip"
)

type Disposisi struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	DocumentID  uint          `gorm:"not null;index" json:"document_id"`
	FromUserID  uint          `gorm:"not null" json:"from_user_id"`
	ToUserID    uint          `gorm:"not null;index" json:"to_user_id"`
	Directive   DirectiveType `gorm:"type:varchar(50);not null" json:"directive"`
	Notes       string        `gorm:"type:text" json:"notes"`
	Deadline    *time.Time    `json:"deadline"`
	IsRead      bool          `gorm:"default:false" json:"is_read"`
	ReadAt      *time.Time    `json:"read_at"`
	CompletedAt *time.Time    `json:"completed_at"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}
```

Endpoints:
| Method | Path | Description |
|--------|------|-------------|
| `POST`  | `/api/v1/dispositions`          | Create a disposition |
| `GET`   | `/api/v1/dispositions/inbox`    | Get user's disposition inbox |
| `PUT`   | `/api/v1/dispositions/:id/read` | Mark as read |
| `PUT`   | `/api/v1/dispositions/:id/complete` | Mark as completed |

**Commit**: `git commit -m "feat(m4): add disposisi model and delegation API"`

---

### Task 4: Immutable Audit Trail Logger

**Files:**
- Create: `internal/model/tenant/audit_log.go`
- Create: `internal/service/audit/logger.go`
- Create: `migrations/tenant/000006_create_audit_logs.up.sql`
- Test: `internal/service/audit/logger_test.go`

**Step 1: Write the failing test**
```go
package audit

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestAuditLogHashChain(t *testing.T) {
	logger := NewLogger()

	entry1 := logger.Log("", 1, 1, "created", "Document created")
	assert.NotEmpty(t, entry1.Hash)

	entry2 := logger.Log(entry1.Hash, 1, 1, "submitted", "Document submitted for review")
	assert.NotEmpty(t, entry2.Hash)
	assert.NotEqual(t, entry1.Hash, entry2.Hash)

	// Verify chain integrity
	assert.True(t, logger.VerifyChain([]AuditEntry{entry1, entry2}))
}

func TestAuditLogTamperDetection(t *testing.T) {
	logger := NewLogger()
	entry1 := logger.Log("", 1, 1, "created", "Document created")
	entry2 := logger.Log(entry1.Hash, 1, 1, "submitted", "Document submitted")

	// Tamper with entry1
	entry1.Action = "tampered"
	assert.False(t, logger.VerifyChain([]AuditEntry{entry1, entry2}))
}
```
Expected: FAIL

**Step 3: Write minimal implementation**
```go
package audit

import (
	"crypto/sha256"
	"fmt"
	"time"
)

type AuditEntry struct {
	ID           uint      `json:"id"`
	DocumentID   uint      `json:"document_id"`
	UserID       uint      `json:"user_id"`
	Action       string    `json:"action"`
	Description  string    `json:"description"`
	PreviousHash string    `json:"previous_hash"`
	Hash         string    `json:"hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) computeHash(prevHash string, docID, userID uint, action, desc string, ts time.Time) string {
	data := fmt.Sprintf("%s|%d|%d|%s|%s|%s", prevHash, docID, userID, action, desc, ts.Format(time.RFC3339Nano))
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h)
}

func (l *Logger) Log(prevHash string, docID, userID uint, action, desc string) AuditEntry {
	ts := time.Now()
	hash := l.computeHash(prevHash, docID, userID, action, desc, ts)
	return AuditEntry{
		DocumentID: docID, UserID: userID, Action: action,
		Description: desc, PreviousHash: prevHash, Hash: hash, CreatedAt: ts,
	}
}

func (l *Logger) VerifyChain(entries []AuditEntry) bool {
	for i, e := range entries {
		expected := l.computeHash(e.PreviousHash, e.DocumentID, e.UserID, e.Action, e.Description, e.CreatedAt)
		if expected != e.Hash {
			return false
		}
		if i > 0 && e.PreviousHash != entries[i-1].Hash {
			return false
		}
	}
	return true
}
```

**Step 4: Run test** → PASS

**Step 5: Commit**
```bash
git add internal/model/tenant/audit_log.go internal/service/audit/ migrations/tenant/
git commit -m "feat(m4): add immutable hash-chained audit trail logger"
```

---

### Task 5: Document Composer UI (Frontend)

**Files:**
- Create: `frontend/src/app/documents/new/page.tsx`
- Create: `frontend/src/app/documents/[id]/page.tsx`
- Create: `frontend/src/app/documents/page.tsx`
- Create: `frontend/src/components/document/Composer.tsx`
- Create: `frontend/src/components/document/TemplateSelector.tsx`
- Create: `frontend/src/components/document/ApprovalTracker.tsx`
- Create: `frontend/src/components/disposisi/DisposisiForm.tsx`
- Create: `frontend/src/components/disposisi/DisposisiInbox.tsx`
- Create: `frontend/src/lib/api/documents.ts`

The composer uses TipTap for rich-text editing with template support (NDE, SK, Agendaris). The approval tracker shows the current position in the verification chain with status badges.

**Commit**: `git commit -m "feat(m4): add document composer and disposisi UI components"`

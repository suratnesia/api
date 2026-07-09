# Milestone 5: IPFS Archive & PrivyID E-Sign Integration

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement tenant-isolated file storage for signed documents and integrate PrivyID for legally binding digital signatures compliant with UU ITE.

**Architecture:** A storage service abstracts the upload target (S3 in production, local filesystem in dev). Files are namespaced by tenant ID. The PrivyID client wraps the PrivyID REST API behind an interface for testability. The signature canvas lets users position their signature on the document before triggering OTP verification.

**Tech Stack:** Go 1.25+, Echo v4, AWS SDK for Go v2 (S3), PrivyID REST API, Next.js, react-pdf, Fabric.js (canvas)

---

### Task 1: Storage Service Interface & S3 Implementation

**Files:**
- Create: `internal/service/storage/storage.go`
- Create: `internal/service/storage/s3.go`
- Create: `internal/service/storage/local.go`
- Test: `internal/service/storage/storage_test.go`

**Step 1: Write the failing test**
```go
package storage

import (
	"bytes"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestLocalStorage_UploadAndRetrieve(t *testing.T) {
	store := NewLocalStorage(t.TempDir())

	content := []byte("test document content")
	path, err := store.Upload("tenant_pt_kai", "doc_001.pdf", bytes.NewReader(content))
	assert.NoError(t, err)
	assert.Contains(t, path, "tenant_pt_kai/doc_001.pdf")

	reader, err := store.Download(path)
	assert.NoError(t, err)
	defer reader.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	assert.Equal(t, content, buf.Bytes())
}

func TestLocalStorage_Delete(t *testing.T) {
	store := NewLocalStorage(t.TempDir())
	content := []byte("to be deleted")
	path, _ := store.Upload("tenant_test", "delete_me.pdf", bytes.NewReader(content))

	err := store.Delete(path)
	assert.NoError(t, err)

	_, err = store.Download(path)
	assert.Error(t, err)
}
```
Expected: FAIL

**Step 2: Run test**: `go test ./internal/service/storage/... -v`

**Step 3: Write minimal implementation**
```go
// internal/service/storage/storage.go
package storage

import "io"

type StorageService interface {
	Upload(tenantID, fileName string, content io.Reader) (path string, err error)
	Download(path string) (io.ReadCloser, error)
	Delete(path string) error
	GetURL(path string) string
}

// internal/service/storage/local.go
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type localStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) StorageService {
	return &localStorage{baseDir: baseDir}
}

func (s *localStorage) Upload(tenantID, fileName string, content io.Reader) (string, error) {
	dir := filepath.Join(s.baseDir, tenantID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(tenantID, fileName)
	fullPath := filepath.Join(s.baseDir, path)
	f, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, content); err != nil {
		return "", err
	}
	return path, nil
}

func (s *localStorage) Download(path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.baseDir, path)
	return os.Open(fullPath)
}

func (s *localStorage) Delete(path string) error {
	fullPath := filepath.Join(s.baseDir, path)
	return os.Remove(fullPath)
}

func (s *localStorage) GetURL(path string) string {
	return fmt.Sprintf("/storage/%s", path)
}
```

S3 implementation (`s3.go`) wraps `aws-sdk-go-v2` with the same interface — deferred to production setup.

**Step 4: Run test** → PASS

**Step 5: Commit**
```bash
git add internal/service/storage/
git commit -m "feat(m5): add storage service with local filesystem implementation"
```

---

### Task 2: File Upload API Endpoint

**Files:**
- Create: `internal/api/handler/upload.go`
- Test: `internal/api/handler/upload_test.go`
- Modify: `internal/api/router/router.go`

**Step 1: Write the failing test**
```go
package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestUploadFileHandler(t *testing.T) {
	e := echo.New()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.pdf")
	part.Write([]byte("fake pdf content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewUploadHandler(nil) // mock storage
	assert.NotNil(t, h)
}
```

**Step 3: Write minimal implementation**
```go
package handler

import (
	"net/http"
	"github.com/labstack/echo/v4"
	"suratnesia/internal/service/storage"
)

type UploadHandler struct {
	storage storage.StorageService
}

func NewUploadHandler(s storage.StorageService) *UploadHandler {
	return &UploadHandler{storage: s}
}

func (h *UploadHandler) Upload(c echo.Context) error {
	tenantID, _ := c.Get("tenant_id").(string)
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read file")
	}
	defer src.Close()

	path, err := h.storage.Upload(tenantID, file.Filename, src)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "upload failed")
	}
	return c.JSON(http.StatusOK, map[string]string{
		"path": path,
		"url":  h.storage.GetURL(path),
	})
}
```

**Commit**: `git commit -m "feat(m5): add file upload API endpoint"`

---

### Task 3: PrivyID API Client Wrapper

**Files:**
- Create: `internal/service/esign/privyid.go`
- Create: `internal/service/esign/types.go`
- Create: `internal/service/esign/mock.go`
- Test: `internal/service/esign/privyid_test.go`

**Step 1: Write the failing test**
```go
package esign

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestMockESignClient_RequestSignature(t *testing.T) {
	client := NewMockClient()

	req := &SignatureRequest{
		DocumentID:  "doc_001",
		DocumentURL: "https://storage.example.com/tenant_pt_kai/doc_001.pdf",
		SignerEmail:  "direktur@ptkai.co.id",
		SignerName:   "Pak Direktur",
		PageNumber:   1,
		PosX:         100,
		PosY:         500,
	}

	resp, err := client.RequestSignature(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.RequestID)
	assert.Equal(t, SignStatusPending, resp.Status)
}

func TestMockESignClient_VerifyOTP(t *testing.T) {
	client := NewMockClient()
	ok, err := client.VerifyOTP("req_001", "123456")
	assert.NoError(t, err)
	assert.True(t, ok)
}
```
Expected: FAIL

**Step 3: Write minimal implementation**
```go
// internal/service/esign/types.go
package esign

type SignStatus string

const (
	SignStatusPending  SignStatus = "pending"
	SignStatusSigned   SignStatus = "signed"
	SignStatusRejected SignStatus = "rejected"
	SignStatusExpired  SignStatus = "expired"
)

type SignatureRequest struct {
	DocumentID  string `json:"document_id"`
	DocumentURL string `json:"document_url"`
	SignerEmail  string `json:"signer_email"`
	SignerName   string `json:"signer_name"`
	PageNumber   int    `json:"page_number"`
	PosX         int    `json:"pos_x"`
	PosY         int    `json:"pos_y"`
}

type SignatureResponse struct {
	RequestID string     `json:"request_id"`
	Status    SignStatus `json:"status"`
	OTPURL    string     `json:"otp_url"`
}

// internal/service/esign/privyid.go
type ESignClient interface {
	RequestSignature(req *SignatureRequest) (*SignatureResponse, error)
	VerifyOTP(requestID, otp string) (bool, error)
	GetStatus(requestID string) (*SignatureResponse, error)
}

// internal/service/esign/mock.go
type mockClient struct{}

func NewMockClient() ESignClient {
	return &mockClient{}
}

func (m *mockClient) RequestSignature(req *SignatureRequest) (*SignatureResponse, error) {
	return &SignatureResponse{
		RequestID: "req_" + req.DocumentID,
		Status:    SignStatusPending,
		OTPURL:    "https://mock.privyid.tech/otp/" + req.DocumentID,
	}, nil
}

func (m *mockClient) VerifyOTP(requestID, otp string) (bool, error) {
	return otp == "123456", nil
}

func (m *mockClient) GetStatus(requestID string) (*SignatureResponse, error) {
	return &SignatureResponse{RequestID: requestID, Status: SignStatusSigned}, nil
}
```

**Step 4: Run test** → PASS

**Step 5: Commit**
```bash
git add internal/service/esign/
git commit -m "feat(m5): add PrivyID e-sign client with mock implementation"
```

---

### Task 4: E-Sign API Endpoints

**Files:**
- Create: `internal/api/handler/esign.go`
- Test: `internal/api/handler/esign_test.go`
- Modify: `internal/api/router/router.go`

Endpoints:
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/documents/:id/sign`      | Request signature placement |
| `POST` | `/api/v1/esign/verify`             | Verify OTP and finalize signature |
| `GET`  | `/api/v1/esign/:requestId/status`  | Check signature status |

**Commit**: `git commit -m "feat(m5): add e-sign request and verification endpoints"`

---

### Task 5: Signature Placement Canvas (Frontend)

**Files:**
- Create: `frontend/src/components/esign/SignatureCanvas.tsx`
- Create: `frontend/src/components/esign/OTPVerification.tsx`
- Create: `frontend/src/components/esign/DocumentViewer.tsx`
- Create: `frontend/src/lib/api/esign.ts`

The canvas renders the PDF using `react-pdf`, overlays a draggable signature placeholder using Fabric.js or a simple absolute-positioned div, and captures the position coordinates (pageNumber, posX, posY) before sending the signature request to the backend.

**Commit**: `git commit -m "feat(m5): add signature placement canvas and OTP verification UI"`

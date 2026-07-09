# Suratnesia Master Product Roadmap & Feature Tasks

This master roadmap outlines the development tasks for the entire **Suratnesia** SaaS platform. It breaks down the application into 6 core milestones spanning the Golang backend microservices and the Next.js frontend.

---

## Milestone 1: Multi-Tenant Architecture & Authentication
**Goal**: Secure and isolate tenant data, route requests based on tenant headers, and handle schema migrations.

### Tasks:
- [ ] **M1.1: Shared Schema & Tenant Provisioning Database Setup**
  - Design the `shared` database schema: `tenants`, `subscriptions`, and `global_configs`.
  - Write SQL migrations to initialize the global database.
- [ ] **M1.2: Tenant DB Migrator**
  - Implement a Go command-line tool/utility to run migrations on a specific tenant schema (e.g., `SET search_path TO tenant_<id>;` followed by `AutoMigrate`).
  - Create a migration registry for tenant-specific tables (`users`, `documents`, `workflow_rules`, `dispositions`).
- [ ] **M1.3: JWT Auth with Tenant Context**
  - Implement JWT token generator injecting `tenant_id` and `user_role` claims.
  - Update `TenantMiddleware` to extract `tenant_id` from JWT claims and initialize the request-scoped GORM transaction.
- [ ] **M1.4: Base Tenant Provisioning Endpoint**
  - Create `POST /api/v1/tenants` (system-only) to provision a new database schema, run migrations, and seed the initial admin account.

---

## Milestone 2: Onboarding Flow & Org Tree Builder
**Goal**: Enable newly registered tenants to upload their employee list and build their organizational structure.

### Tasks:
- [ ] **M2.1: Org Structure API (Backend)**
  - Implement CRUD endpoints for Office Locations (`/offices`), Units/Divisions (`/units`), and Positions (`/positions`).
  - Implement Grade Level management (`/grades` e.g., Grade 1 to N).
- [ ] **M2.2: Employee CSV Upload & Validator**
  - Implement `POST /api/v1/employees/import` endpoint.
  - Parse CSV fields: `nippos`, `name`, `email`, `position_id`, `grade_level`, `is_bod`.
  - Validate email uniqueness, position references, and return pre-import validation reports.
- [ ] **M2.3: Next.js Org Builder UI**
  - Build a visual drag-and-drop hierarchy builder to model divisions and units.
  - Integrate list view for importing and managing employee directories.
- [ ] **M2.4: 5-Step Onboarding Wizard Frontend**
  - Step 1: Register organization profile (Nama perusahaan, NPWP/NIB, Swasta/BUMN/Pemda).
  - Step 2: Configure Office Units & Grades.
  - Step 3: Bulk import employees via CSV.
  - Step 4: Configure initial workflow rules.
  - Step 5: Provision subdomain (e.g., `pt-kai.suratnesia.id`) and trigger bulk invitations.

---

## Milestone 3: Dynamic Rule-Based Workflow Engine
**Goal**: Replace hardcoded routing with a configurable engine mapping how documents are verified based on grades and units.

### Tasks:
- [ ] **M3.1: Workflow Rules DB Schema**
  - Create `workflow_rules` table: `id`, `sender_grade_min`, `sender_grade_max`, `recipient_grade_min`, `recipient_grade_max`, `route_steps` (JSON/Array of intermediate grades or roles).
- [ ] **M3.2: Routing Resolver Service (Backend)**
  - Implement `GetApprovalChain(senderID, recipientID)` core engine logic.
  - Query tenant workflow rules, match grades, and output the sequence of intermediate verifiers (e.g., *Staf (Grade 3) -> Kabag (Grade 6) -> Direktur (Grade 8)*).
- [ ] **M3.3: Interactive Rule Builder UI**
  - Design a rule-building panel where admins define routing logic using conditional sliders or dropdowns.
  - Implement a **Live Preview Chain Simulator** showcasing a visual graph of how a test memo will route given the current rules.

---

## Milestone 4: Core Document Modules (Composer & Disposisi)
**Goal**: Allow users to write decrees/memos, trigger approval workflows, delegate tasks, and maintain audit trails.

### Tasks:
- [ ] **M4.1: NDE/SK Composer**
  - Build a rich-text document composer UI in Next.js.
  - Support templates for NDE (internal memo), SK (official decree), and Agendaris registers.
  - Create backend metadata schema to store document states (`draft`, `reviewing`, `signed`, `archived`) and attachments.
- [ ] **M4.2: Disposisi (Directive & Delegation) Flow**
  - Build Disposition dispatch form allowing superiors to delegate documents to subordinates.
  - Add deadline tracking, action directives (e.g., *"Tindaklanjuti"*, *"Hadir Mewakili"*), and read-status logs.
- [ ] **M4.3: Immutable Audit Trail Logger**
  - Create a database table tracking all document lifecycle actions (created, reviewed, edited, signed, forwarded, archived).
  - Generate a secure hash of each log entry chained to the previous one to prevent audit logs from being modified.

---

## Milestone 5: IPFS Archive & PrivyID E-Sign Integration
**Goal**: Implement legally binding digital signatures and securely archive finalized documents.

### Tasks:
- [ ] **M5.1: IPFS/S3 Storage Integration**
  - Implement file uploader backend service.
  - Upload PDF renders of signed documents to IPFS/S3 with tenant-isolated folder namespaces.
- [ ] **M5.2: PrivyID API Client Wrapper**
  - Implement a mockable client for PrivyID APIs (Registration, Upload Document, Request Signature, OTP callback verification).
- [ ] **M5.3: Signature Placement Canvas**
  - Implement a frontend component where users drag-and-drop their signature placeholder onto the document canvas before requesting OTP signing.

---

## Milestone 6: Subscription & Metered Billing
**Goal**: Integrate payment gateways and implement usage metering to track storage and signature overages.

### Tasks:
- [ ] **M6.1: Usage Metering Logger**
  - Implement middleware/hooks to count:
    - Active users (checked monthly).
    - IPFS storage volume (checked daily).
    - Completed E-Signs (checked per transaction).
- [ ] **M6.2: Midtrans / Xendit Payment Integration**
  - Set up webhooks to handle subscription payments, invoices, and card auto-billing.
  - Implement pricing tier limits enforcement (e.g., block document composer if a Starter tenant exceeds 50 users or 5GB storage).

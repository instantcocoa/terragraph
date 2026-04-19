# TerraGraph

A visual Terraform graph editor and inspector. Load any Terraform workspace, visualize resources as an interactive graph, edit HCL through a structured inspector or raw editor, run validation and plan previews, and manage infrastructure visually.

![TerraGraph](terragraph.png)

## Architecture

```
terragraph/
  backend/                  # Go API server
    cmd/server/             # Entry point
    internal/
      api/                  # HTTP handlers (CORS, routing)
      graph/                # Graph extraction + module expansion
      history/              # Undo/redo version tracking
      parser/               # HCL parsing, patching, block ops
      terraform/            # terraform-exec, schema, plan, LSP
  src/                      # SvelteKit frontend
    lib/
      api.ts                # API client
      layout.ts             # Dagre DAG layout with tier ordering
      types.ts              # TypeScript types
      stores/
        workspace.svelte.ts # Core state (Svelte 5 runes)
        theme.svelte.ts     # Theme state (dark/light/solarized)
      components/
        GraphCanvas         # Svelte Flow graph
        TerraNode           # Custom graph node
        Inspector           # Right panel: schema-driven attributes
        HCLEditor           # Monaco editor modal
        ExpressionEditor    # Structured expression editors
        ResourcePalette     # Searchable resource type browser
        BottomPanel         # Diagnostics, plan details
        Sidebar             # Explorer tree + palette tabs
        AddBlockDialog      # Add resource/data/variable/output/provider
        Toolbar             # Actions, undo/redo, theme, validate, plan
  examples/                 # Example Terraform workspaces
```

## Tech Stack

- **Frontend**: SvelteKit, Svelte 5, Svelte Flow, Monaco Editor, Tailwind CSS, dagre
- **Backend**: Go, terraform-exec, terraform-json, HCL v2, terraform-ls (LSP)
- **Build**: Bun, Vite, Go modules
- **Testing**: Vitest, Playwright, Go test

## Setup

### Prerequisites

- [Bun](https://bun.sh/)
- [Go](https://go.dev/) 1.21+
- [Terraform](https://www.terraform.io/) CLI

### Optional (for LSP features)

```bash
# Install terraform-ls for live diagnostics and hover
brew install hashicorp/tap/terraform-ls

# Or download from https://github.com/hashicorp/terraform-ls/releases
```

### Install & Run

```bash
# Install frontend dependencies
bun install

# Install Go dependencies
cd backend && go mod tidy && cd ..

# Start backend (port 3001)
cd backend && go run ./cmd/server &

# Start frontend (port 5173, proxies /api to backend)
bun run dev
```

Then open http://localhost:5173 and load a Terraform workspace (type a path or click Open).

## Features

### Graph Visualization
- All Terraform block types as color-coded nodes with dependency edges
- Deterministic DAG layout: variables at top, resources in middle, outputs at bottom
- Connected resources cluster together
- Zoom-to-node when selecting from sidebar
- Module expansion for local modules (shows internal resources as children)

### Inspector & Editing
- Schema-driven inspector showing all possible attributes per resource type
- Required, optional, and computed fields clearly separated
- Inline editing with type-aware editors (text, bool toggle, number, reference links)
- Structured expression editors for strings, references, functions, conditionals
- One-click navigation between references ("Used By" / clickable ref tags)
- Full HCL editor modal with Monaco (syntax highlighting, line numbers)
- Rename blocks with automatic cross-file reference updates
- Add/remove nested blocks, remove attributes

### Workspace Operations
- Add resources, data sources, variables, outputs, and providers
- Resource palette with searchable provider-grouped type browser
- Connected resource scaffolding (e.g. adding aws_instance suggests VPC, subnet, AMI)
- Required fields auto-populated with sensible defaults
- Terraform validate with clear pass/fail feedback
- Terraform plan with expandable per-resource diffs (create/update/delete details)
- Data source evaluation (query live provider values)
- Undo/redo version history (Cmd+Z / Cmd+Shift+Z)
- Native folder picker (Open button)

### UI
- Three themes: Tokyo Night (dark), Clean Light, Solarized Dark
- Resizable panels (drag edges of sidebar, inspector, bottom panel)
- Collapsible bottom panel with status badges
- Keyboard shortcuts (Cmd+Z undo, Cmd+Shift+Z redo, Cmd+B sidebar, Escape deselect)

### LSP Integration
- Connects to terraform-ls for live diagnostics and hover info
- Auto-starts when terraform-ls is available
- Diagnostics fed to Monaco editor markers

### Supported Terraform Constructs
- Resources, data sources, modules, providers, variables, locals, outputs
- Expression references, depends_on, nested blocks
- count, for_each, template strings, function calls
- Local module expansion

## Development

```bash
bun run check          # Type check
bun run build          # Production build
bun run test:unit      # Frontend unit tests
bun run test:e2e       # Playwright E2E tests
cd backend && go test ./...   # Backend tests
cd backend && go build ./...  # Backend build
```

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Health check |
| POST | `/api/workspace/load` | Parse workspace, return graph |
| POST | `/api/workspace/validate` | Run terraform validate |
| POST | `/api/workspace/plan` | Run terraform plan |
| POST | `/api/workspace/patch` | Patch attribute in HCL |
| POST | `/api/workspace/add-block` | Add resource/data/variable/output |
| POST | `/api/workspace/remove-block` | Remove a block |
| POST | `/api/workspace/add-provider` | Add provider + terraform block |
| POST | `/api/workspace/rename-block` | Rename with ref updates |
| POST | `/api/workspace/schema` | Get provider schemas |
| POST | `/api/workspace/scaffold` | Get resource dependencies |
| POST | `/api/workspace/eval-data` | Evaluate data source |
| POST | `/api/workspace/undo` | Undo last change |
| POST | `/api/workspace/redo` | Redo last undo |
| POST | `/api/workspace/write-file` | Write file content |
| GET | `/api/lsp/status` | Check terraform-ls availability |
| POST | `/api/lsp/diagnostics` | Get LSP diagnostics for file |
| POST | `/api/lsp/hover` | Get hover info at position |

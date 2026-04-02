@AGENTS.md

## Conventions
- Server construction uses options pattern: `NewServer(llm, opts...)`
- Dual-path persistence: if eventStore is wired, emit events; otherwise direct store writes
- `ReadStore` for queries, `ReadModelWriter` for projections — never mix read/write paths
- "Tools" means axon agent tools (tool_router.go, tools.go) — not Claude Code skills
- React + shadcn/ui + Tailwind + TanStack Query frontend in `web/`, built output embedded via `//go:embed all:static`

## Constraints
- Leaf service — no other axon-* module may import axon-chat
- Frontend is IN this repo (`web/` dir) — do not create or reference a separate frontend repo
- Do not add direct database imports — persistence is behind store interfaces
- Do not bypass the event sourcing path when eventStore is configured

## Testing
- `go test ./...` — backend tests
- Frontend: `cd web && npm install && npm run build` (builds to `static/`)
- Backend tests do not require a running database or frontend build

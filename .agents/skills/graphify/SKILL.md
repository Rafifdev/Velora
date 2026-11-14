---
name: graphify
description: any input (code, docs, papers, images) -> knowledge graph -> clustered communities -> HTML + JSON + audit report for token savings
---

# Graphify (`/graphify`) — Knowledge Graph for Token Savings

Turn any codebase or folder of files into a navigable knowledge graph with deterministic AST extraction (`tree-sitter`), community clustering, and compact GraphRAG queries (`GRAPH_REPORT.md` & `graph.json`) to save tokens instead of re-reading raw files.

## Python Runtime on This Machine

Always invoke Graphify using the installed Python 3.13 interpreter:

```powershell
$py = "C:\Program Files\PostgreSQL\18\pgAdmin 4\python\python.exe"
$env:PATH = "C:\Program Files\PostgreSQL\18\pgAdmin 4\python;C:\Users\MyBook Hype AMD\AppData\Roaming\Python\Python313\Scripts;$env:PATH"
```

## Core Commands

- **Rebuild AST + Graph (0 LLM tokens for code):**
  ```powershell
  & $py scripts\run_graphify.py
  ```
- **Query the graph with a strict token budget (BFS / DFS):**
  ```powershell
  & $py scripts\query_graphify.py "<question>" --mode bfs --budget 1500
  & $py scripts\query_graphify.py "<question>" --mode dfs --budget 1500
  ```
- **Watch folder for live AST auto-sync:**
  ```powershell
  & $py -m graphify.watch . --debounce 3
  ```
- **Start Graphify MCP stdio server:**
  ```powershell
  & $py -m graphify.serve graphify-out/graph.json
  ```

## Outputs in `graphify-out/`

- `graphify-out/GRAPH_REPORT.md` — God nodes, community map, surprising cross-module connections, and suggested questions.
- `graphify-out/graph.json` — Persistent NetworkX node-link graph for token-budgeted BFS/DFS queries.
- `graphify-out/graph.html` — Interactive browser visualization.
- `graphify-out/obsidian/` & `graphify-out/obsidian/graph.canvas` — Obsidian vault & canvas layout.
- `graphify-out/cost.json` — Token cost & benchmark tracker.

## Token-Saving Workflow Rules

1. **Read `graphify-out/GRAPH_REPORT.md` first** before opening or grepping multiple source files.
2. **Query `graphify-out/graph.json`** with `scripts/query_graphify.py` (`--budget 1500`) to find exact `source_file` and `source_location` targets before reading individual files with `StartLine`/`EndLine`.
3. **Never dump full `graph.json`** into context; always traverse only the matching subgraph.
4. **Update after edits:** Re-run `& $py scripts\run_graphify.py` after significant code changes so the graph stays accurate.

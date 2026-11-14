from __future__ import annotations

import argparse
import json
from pathlib import Path

from networkx.readwrite import json_graph


def main() -> None:
    parser = argparse.ArgumentParser(description="Token-budgeted BFS/DFS query on graphify-out/graph.json")
    parser.add_argument("question", help="Question or concept keywords to search")
    parser.add_argument("--mode", choices=["bfs", "dfs"], default="bfs")
    parser.add_argument("--budget", type=int, default=1500)
    parser.add_argument("--graph", default="graphify-out/graph.json")
    args = parser.parse_args()

    graph_path = Path(args.graph)
    if not graph_path.exists():
        raise SystemExit(f"ERROR: {graph_path} not found. Run scripts/run_graphify.py first.")

    data = json.loads(graph_path.read_text(encoding="utf-8"))
    G = json_graph.node_link_graph(data, edges="links")

    terms = [t.lower() for t in args.question.replace("?", "").split() if len(t) > 2]
    scored = []
    for nid, ndata in G.nodes(data=True):
        label = str(ndata.get("label", "")).lower()
        src = str(ndata.get("source_file", "")).lower()
        score = sum(1 for t in terms if t in label or t in src)
        if score > 0:
            scored.append((score, nid))
    scored.sort(reverse=True)
    start_nodes = [nid for _, nid in scored[:3]]

    if not start_nodes:
        print(f"No matching nodes found for terms: {terms}")
        return

    subgraph_nodes = set()
    subgraph_edges = []

    if args.mode == "dfs":
        visited = set()
        stack = [(n, 0) for n in reversed(start_nodes)]
        while stack:
            node, depth = stack.pop()
            if node in visited or depth > 5:
                continue
            visited.add(node)
            subgraph_nodes.add(node)
            for neighbor in G.neighbors(node):
                if neighbor not in visited:
                    stack.append((neighbor, depth + 1))
                    subgraph_edges.append((node, neighbor))
    else:
        frontier = set(start_nodes)
        subgraph_nodes = set(start_nodes)
        for _ in range(3):
            next_frontier = set()
            for n in frontier:
                for neighbor in G.neighbors(n):
                    if neighbor not in subgraph_nodes:
                        next_frontier.add(neighbor)
                        subgraph_edges.append((n, neighbor))
            subgraph_nodes.update(next_frontier)
            frontier = next_frontier

    def relevance(nid: str) -> int:
        label = str(G.nodes[nid].get("label", "")).lower()
        return sum(1 for t in terms if t in label)

    ranked_nodes = sorted(subgraph_nodes, key=relevance, reverse=True)
    lines = [
        f"Traversal: {args.mode.upper()} | Start: {[G.nodes[n].get('label', n) for n in start_nodes]} | {len(subgraph_nodes)} nodes"
    ]
    for nid in ranked_nodes:
        d = G.nodes[nid]
        lines.append(f"  NODE {d.get('label', nid)} [src={d.get('source_file', '')} loc={d.get('source_location', '')}]")
    for u, v in subgraph_edges:
        if u in subgraph_nodes and v in subgraph_nodes:
            d = G.edges[u, v]
            lines.append(
                f"  EDGE {G.nodes[u].get('label', u)} --{d.get('relation', '')} [{d.get('confidence', '')}]--> {G.nodes[v].get('label', v)}"
            )

    char_budget = args.budget * 4
    output = "\n".join(lines)
    if len(output) > char_budget:
        output = output[:char_budget] + f"\n... (truncated at ~{args.budget} token budget)"
    print(output)


if __name__ == "__main__":
    main()

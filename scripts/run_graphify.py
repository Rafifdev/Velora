from __future__ import annotations

import json
from datetime import datetime, timezone
from pathlib import Path

from graphify.analyze import god_nodes, suggest_questions, surprising_connections
from graphify.benchmark import print_benchmark, run_benchmark
from graphify.build import build_from_json
from graphify.cache import save_semantic_cache
from graphify.cluster import cluster, label_communities_by_hub, score_all
from graphify.detect import detect, save_manifest
from graphify.export import to_canvas, to_html, to_json, to_obsidian
from graphify.extract import collect_files, extract
from graphify.report import generate


EXCLUDE_PARTS = {"node_modules", "dist", "build", "graphify-out", ".agents", "scripts"}


def is_included(rel_path: str) -> bool:
    parts = set(Path(rel_path).parts)
    return not bool(parts & EXCLUDE_PARTS)


def build_velora_semantic_layer(root: Path) -> dict:
    """Capture cross-layer Svelte -> Wails Bridge -> Go -> SQLite relationships."""
    nodes = [
        {
            "id": "velora_architecture_plan",
            "label": "Velora Implementation Plan",
            "file_type": "document",
            "source_file": "implementation_plan.md",
            "source_location": "L1-L173",
        },
        {
            "id": "svelte_app_shell",
            "label": "App.svelte (Root Shell & Audio Engine)",
            "file_type": "code",
            "source_file": "frontend/src/App.svelte",
            "source_location": "L1-L490",
        },
        {
            "id": "svelte_titlebar",
            "label": "Titlebar.svelte (Nav, Global Search, Streak Badge)",
            "file_type": "code",
            "source_file": "frontend/src/components/Titlebar.svelte",
            "source_location": "L1-L143",
        },
        {
            "id": "svelte_sidebar",
            "label": "Sidebar.svelte (Menu & Your Library)",
            "file_type": "code",
            "source_file": "frontend/src/components/Sidebar.svelte",
            "source_location": "L1-L260",
        },
        {
            "id": "svelte_playerbar",
            "label": "PlayerBar.svelte (Spotify 1:1 Bottom Controls)",
            "file_type": "code",
            "source_file": "frontend/src/components/PlayerBar.svelte",
            "source_location": "L1-L243",
        },
        {
            "id": "svelte_tracktable",
            "label": "TrackTable.svelte (Song Rows & Playlist Menu)",
            "file_type": "code",
            "source_file": "frontend/src/components/TrackTable.svelte",
            "source_location": "L1-L157",
        },
        {
            "id": "svelte_streak_modal",
            "label": "StreakModal.svelte (Native Dialog Fire Streak)",
            "file_type": "code",
            "source_file": "frontend/src/components/StreakModal.svelte",
            "source_location": "L1-L152",
        },
        {
            "id": "svelte_create_playlist_modal",
            "label": "CreatePlaylistModal.svelte (Native Dialog Playlist)",
            "file_type": "code",
            "source_file": "frontend/src/components/CreatePlaylistModal.svelte",
            "source_location": "L1-L148",
        },
        {
            "id": "svelte_home_view",
            "label": "HomeView.svelte (Greeting, Quick Grid, All Songs)",
            "file_type": "code",
            "source_file": "frontend/src/views/HomeView.svelte",
            "source_location": "L1-L194",
        },
        {
            "id": "svelte_library_views",
            "label": "LibraryViews.svelte (Search, Liked, Albums, Artists)",
            "file_type": "code",
            "source_file": "frontend/src/views/LibraryViews.svelte",
            "source_location": "L1-L347",
        },
        {
            "id": "svelte_playlist_views",
            "label": "PlaylistViews.svelte (Playlists, Local M3U, Reorder)",
            "file_type": "code",
            "source_file": "frontend/src/views/PlaylistViews.svelte",
            "source_location": "L1-L399",
        },
        {
            "id": "svelte_sources_view",
            "label": "SourcesView.svelte (Upload Local & YTM Explorer)",
            "file_type": "code",
            "source_file": "frontend/src/views/SourcesView.svelte",
            "source_location": "L1-L185",
        },
        {
            "id": "svelte_analytics_settings_views",
            "label": "AnalyticsAndSettingsViews.svelte (History, Stats, Streak, Prefs)",
            "file_type": "code",
            "source_file": "frontend/src/views/AnalyticsAndSettingsViews.svelte",
            "source_location": "L1-L407",
        },
        {
            "id": "sqlite_schema_file",
            "label": "SQLite Schema (sources, tracks, playlists, history, daily_streaks)",
            "file_type": "code",
            "source_file": "internal/db/schema.sql",
            "source_location": "L1-L97",
        },
        {
            "id": "wails_bridge_app_go",
            "label": "Wails App Bindings (app.go)",
            "file_type": "code",
            "source_file": "app.go",
            "source_location": "L1-L333",
        },
        {
            "id": "frontend_api_bridge",
            "label": "Frontend Wails API Client (api.ts)",
            "file_type": "code",
            "source_file": "frontend/src/lib/api.ts",
            "source_location": "L1-L170",
        },
    ]

    edges = [
        ("svelte_app_shell", "svelte_titlebar", "renders"),
        ("svelte_app_shell", "svelte_sidebar", "renders"),
        ("svelte_app_shell", "svelte_playerbar", "renders"),
        ("svelte_app_shell", "svelte_streak_modal", "renders"),
        ("svelte_app_shell", "svelte_create_playlist_modal", "renders"),
        ("svelte_app_shell", "svelte_home_view", "renders"),
        ("svelte_app_shell", "svelte_library_views", "renders"),
        ("svelte_app_shell", "svelte_playlist_views", "renders"),
        ("svelte_app_shell", "svelte_sources_view", "renders"),
        ("svelte_app_shell", "svelte_analytics_settings_views", "renders"),
        ("svelte_home_view", "svelte_tracktable", "renders"),
        ("svelte_library_views", "svelte_tracktable", "renders"),
        ("svelte_app_shell", "frontend_api_bridge", "calls"),
        ("svelte_titlebar", "frontend_api_bridge", "calls"),
        ("frontend_api_bridge", "wails_bridge_app_go", "calls"),
        ("wails_bridge_app_go", "sqlite_schema_file", "persists_to"),
        ("svelte_playerbar", "svelte_app_shell", "emits_playback_and_streak_progress"),
        ("svelte_analytics_settings_views", "sqlite_schema_file", "queries_listening_stats_and_fire_streak"),
        ("velora_architecture_plan", "svelte_app_shell", "specifies"),
        ("velora_architecture_plan", "wails_bridge_app_go", "specifies"),
        ("velora_architecture_plan", "sqlite_schema_file", "specifies"),
    ]

    formatted_edges = [
        {
            "source": src,
            "target": dst,
            "relation": rel,
            "confidence": "EXTRACTED",
            "source_file": "frontend/src/App.svelte",
            "source_location": None,
            "weight": 1.0,
        }
        for src, dst, rel in edges
    ]

    # Save into graphify semantic cache so --update preserves it cleanly
    allowed_files = {n["source_file"] for n in nodes if (root / n["source_file"]).exists()}
    if allowed_files:
        save_semantic_cache(
            nodes,
            formatted_edges,
            root=root,
            allowed_source_files=sorted(allowed_files),
        )
    return {"nodes": nodes, "edges": formatted_edges, "input_tokens": 0, "output_tokens": 0}


def assign_community_labels(G, communities: dict[int, list[str]]) -> dict[int, str]:
    auto = label_communities_by_hub(G, communities)
    labels: dict[int, str] = {}
    for cid, members in communities.items():
        files = " ".join(G.nodes[m].get("source_file", "") for m in members).lower()
        if "internal/db" in files or "schema.sql" in files:
            labels[cid] = "SQLite Database & Library CRUD"
        elif "internal/stats" in files:
            labels[cid] = "Listening Stats & Fire Streak"
        elif "internal/audio" in files:
            labels[cid] = "Audio Engine & Media Server"
        elif "internal/sources" in files and "ytmusic" in files:
            labels[cid] = "YouTube Music InnerTube Source"
        elif "internal/sources" in files:
            labels[cid] = "Local Audio Scanner & M3U"
        elif "app.go" in files or "main.go" in files:
            labels[cid] = "Wails Desktop Bridge & Lifecycle"
        elif "frontend/src" in files:
            labels[cid] = "Svelte 5 Spotify UI & Components"
        else:
            labels[cid] = auto.get(cid, f"Community {cid}")
    return labels


def main() -> None:
    root = Path(".").resolve()
    out_dir = root / "graphify-out"
    out_dir.mkdir(parents=True, exist_ok=True)

    detection = detect(root)
    filtered_files: dict[str, list[str]] = {}
    for kind, items in detection.get("files", {}).items():
        filtered_files[kind] = [p for p in items if is_included(p)]
    detection["files"] = filtered_files
    detection["total_files"] = sum(len(v) for v in filtered_files.values())

    code_paths: list[Path] = []
    for p in filtered_files.get("code", []):
        fp = root / p
        if fp.is_dir():
            code_paths.extend(collect_files(fp, root=root))
        elif fp.is_file():
            code_paths.append(fp)

    ast_result = extract(code_paths, root=root) if code_paths else {"nodes": [], "edges": []}
    sem_result = build_velora_semantic_layer(root)

    seen_ids = {n["id"] for n in ast_result["nodes"]}
    merged_nodes = list(ast_result["nodes"])
    for n in sem_result["nodes"]:
        if n["id"] not in seen_ids:
            merged_nodes.append(n)
            seen_ids.add(n["id"])
    merged_edges = list(ast_result["edges"]) + list(sem_result["edges"])

    extraction = {
        "nodes": merged_nodes,
        "edges": merged_edges,
        "input_tokens": 0,
        "output_tokens": 0,
    }

    G = build_from_json(extraction, root=root)
    communities = cluster(G)
    cohesion = score_all(G, communities)
    labels = assign_community_labels(G, communities)
    gods = god_nodes(G)
    surprises = surprising_connections(G, communities)
    questions = suggest_questions(G, communities, labels)
    tokens = {"input": 0, "output": 0}

    report_md = generate(
        G,
        communities,
        cohesion,
        labels,
        gods,
        surprises,
        detection,
        tokens,
        str(root),
        suggested_questions=questions,
    )
    (out_dir / "GRAPH_REPORT.md").write_text(report_md, encoding="utf-8")
    to_json(G, communities, str(out_dir / "graph.json"), community_labels=labels, force=True)
    to_html(G, communities, str(out_dir / "graph.html"), community_labels=labels)

    obs_dir = out_dir / "obsidian"
    obs_dir.mkdir(parents=True, exist_ok=True)
    to_obsidian(G, communities, str(obs_dir), community_labels=labels, cohesion=cohesion)
    to_canvas(G, communities, str(obs_dir / "graph.canvas"), community_labels=labels)

    save_manifest(filtered_files, root=root)

    cost_path = out_dir / "cost.json"
    cost = json.loads(cost_path.read_text(encoding="utf-8")) if cost_path.exists() else {
        "runs": [],
        "total_input_tokens": 0,
        "total_output_tokens": 0,
    }
    cost["runs"].append({
        "date": datetime.now(timezone.utc).isoformat(),
        "input_tokens": 0,
        "output_tokens": 0,
        "files": detection["total_files"],
        "nodes": G.number_of_nodes(),
        "edges": G.number_of_edges(),
    })
    cost_path.write_text(json.dumps(cost, indent=2), encoding="utf-8")

    print(
        f"Graphify complete: {G.number_of_nodes()} nodes, {G.number_of_edges()} edges, "
        f"{len(communities)} communities, {detection['total_files']} files"
    )
    bench = run_benchmark(str(out_dir / "graph.json"), corpus_words=detection.get("total_words", 0))
    print_benchmark(bench)


if __name__ == "__main__":
    main()

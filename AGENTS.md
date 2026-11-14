# Velora — Project Rules & Graphify Token Optimization

## Graphify Knowledge Graph (Hemat Token)

Proyek ini terintegrasi dengan **Graphify** (`Graphify-Labs/graphify`) di `graphify-out/` untuk memetakan arsitektur kode Go, TypeScript, Svelte, SQLite schema, dan dokumen proyek.

### Environment CLI Graphify di Mesin Ini

```powershell
$py = "C:\Program Files\PostgreSQL\18\pgAdmin 4\python\python.exe"
$env:PATH = "C:\Program Files\PostgreSQL\18\pgAdmin 4\python;C:\Users\MyBook Hype AMD\AppData\Roaming\Python\Python313\Scripts;$env:PATH"
```

### Aturan Hemat Token

1. **Query Graph Sebelum Membaca Banyak File**:
   - Gunakan `graphify query "<pertanyaan>" --budget 1500` (atau `& $py scripts\query_graphify.py "<pertanyaan>" --budget 1500`) untuk menelusuri subgraph relevan.
   - Gunakan `graphify path "<A>" "<B>"` untuk melacak jalur ketergantungan antar komponen.
   - Gunakan `graphify explain "<konsep>"` untuk memahami satu node dan seluruh koneksinya.
   - Baca `graphify-out/GRAPH_REPORT.md` untuk tinjauan arsitektur global (God Nodes, komunitas modul, dan pertanyaan arsitektural).
2. **Jangan Membaca `graph.json` Secara Utuh**: Selalu traversal subgraph dengan `--budget`.
3. **Sinkronisasi Setelah Edit Kode (0 Token LLM)**:
   - Jalankan `graphify update .` atau `& $py scripts\run_graphify.py` setelah mengubah kode agar `graphify-out/` selalu mutakhir.

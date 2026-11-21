## Git Workflow & Commit Standard

Aturan setiap kali diminta melakukan commit, push, dan merge di proyek Velora:

1. **Pisahkan Commit Sesuai Kategori Perubahan**:
   - Jangan menggabungkan seluruh perubahan ke dalam 1 commit.
   - Kelompokkan file berdasarkan area/fiturnya (misalnya `audio`, `lyrics`, `dynamic-island`, `settings`, `ui`, `db`, `sync`).
2. **Format Conventional Commits (Standar Industri)**:
   - Gunakan awalan `feat(<scope>):`, `fix(<scope>):`, `refactor(<scope>):`, `style(<scope>):`, atau `chore(<scope>):`.
   - Tulis pesan commit dalam **bahasa Indonesia yang natural, jelas, dan mudah dipahami developer lain** (hindari bahasa terlalu baku/kaku).
3. **Alur Push & Merge**:
   - **Dilarang melakukan `git push` atau `git merge` ke GitHub tanpa izin dan perintah eksplisit dari pengguna.** Selalu tunggu pengguna memerintahkan push/merge terlebih dahulu.
   - Setelah diizinkan: commit dan push ke branch `develop` terlebih dahulu (`git push origin develop`), lalu checkout `main`, lakukan merge dari `develop` tanpa menyertakan file internal development (`.agents/`, `AGENTS.md`, `THEME_PALETTES.md`, `implementation_plan.md`, `.graphifyignore`, `scripts/query_graphify.py`, `scripts/run_graphify.py`), push ke `origin/main`, dan kembali ke branch `develop`.
   - Branch `develop` tetap menyimpan seluruh file lengkap (termasuk `.agents/`, `AGENTS.md`, `THEME_PALETTES.md`, dll.), sedangkan branch `main` bersih dari file internal tersebut.
   - Jangan ikutkan file referensi sementara yang tidak terkait (seperti `Auralys-1.9.7-Settings.md`).

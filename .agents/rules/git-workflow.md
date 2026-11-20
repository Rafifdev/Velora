## Git Workflow & Commit Standard

Aturan setiap kali diminta melakukan commit, push, dan merge di proyek Velora:

1. **Pisahkan Commit Sesuai Kategori Perubahan**:
   - Jangan menggabungkan seluruh perubahan ke dalam 1 commit.
   - Kelompokkan file berdasarkan area/fiturnya (misalnya `audio`, `lyrics`, `dynamic-island`, `settings`, `ui`, `db`, `sync`).
2. **Format Conventional Commits (Standar Industri)**:
   - Gunakan awalan `feat(<scope>):`, `fix(<scope>):`, `refactor(<scope>):`, `style(<scope>):`, atau `chore(<scope>):`.
   - Tulis pesan commit dalam **bahasa Indonesia yang natural, jelas, dan mudah dipahami developer lain** (hindari bahasa terlalu baku/kaku).
3. **Alur Push & Merge**:
   - Commit dan push ke branch `develop` terlebih dahulu (`git push origin develop`).
   - Checkout `main`, lakukan `git merge develop --no-ff -m "Merge branch 'develop'"`, push ke `origin/main`, lalu kembali ke branch `develop`.
   - Jangan ikutkan file referensi sementara yang tidak terkait (seperti `Auralys-1.9.7-Settings.md`).

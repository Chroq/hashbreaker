# HashBreaker — Optimisation Haute Performance en Go

Projet pédagogique pour le cours **M2 Performance for Backend** (Sup de Vinci).  
Étude pratique de l'ingénierie des performances en Go : calcul intensif, gestion fine de la mémoire, élimination de la pression sur le Garbage Collector et parallélisme multi-cœurs à travers le cassage d'empreintes **SHA-256** par force brute (_Time-Memory Tradeoff_).

---

## 1. Document d'Audit de Performance (PDF & HTML)

L'analyse technique approfondie, les profils dynamiques `pprof`, les captures de **flamegraphs** et les diagrammes micro-architecturaux sont centralisés dans le dossier d'audit dédié :

📁 **`audit/`** *(exclu de Git via `.git/info/exclude`)*
- **`audit/index.html`** : Document d'audit interactif paginé (6 pages A4 strictes) prêt pour l'exportation PDF via le navigateur (Ctrl+P / bouton d'impression).
- **`audit/rapport_audit_performance.pdf`** : Version PDF prête à l'emploi.
- **`audit/assets/`** : Flamegraphs CPU/Heap, histogrammes comparatifs vectoriels et diagrammes d'alignement mémoire SoA.

```bash
# Ouvrir le document d'audit dans le navigateur :
xdg-open audit/index.html

# Ou régénérer le PDF en ligne de commande :
chromium --headless=new --no-sandbox --print-to-pdf-no-header --print-to-pdf=audit/rapport_audit_performance.pdf file://$(pwd)/audit/index.html
```

---

## 2. Synthèse Comparative des Performances ($d=4$, 16M hashs)

| Version | Stratégie Architecturale                                 | Temps de Génération |  Empreinte RAM  | Allocations Tas | Latence de Recherche (`GetRaw`) |
| :------ | :------------------------------------------------------- | :-----------------: | :-------------: | :-------------: | :-----------------------------: |
| **V0**  | Baseline mono-thread (`map[[32]byte]string`)             |      `19.92 s`      |   `3.553 GiB`   |  `16 137 750`   |           `34.54 ns`            |
| **V1**  | Préallocation de capacité + Odomètre itératif            |      `18.52 s`      |   `1.811 GiB`   |  `16 072 515`   |           `37.77 ns`            |
| **V2**  | Sharded Maps (256 shards + Mutex + Micro-lots 64)        |      `1.80 s`       |   `1.506 GiB`   |    `66 070`     |           `18.80 ns`            |
| **V3**  | Tableaux Contigus Triés (`[256][]Entry` + Dichotomie)    |      `3.01 s`       | **`0.693 GiB`** |     **`287`**   |           `191.95 ns`           |
| **V4**  | Single Map unifiée + `sync.RWMutex` + Batched            |      `7.65 s`       |   `1.501 GiB`   |    `65 710`     |           `31.38 ns`            |
| **V5**  | **SoA Tagged Flat Table (Fast-Path Pur & Sortie Texte)** |    **`1.56 s`**     |   `1.318 GiB`   |     **`808`**   |  **`8.21 ns`** (0 alloc / 0 B)  |

### Faits Marquants de la Version V5 :
- ⚡ **Latence de recherche record :** **`8.21 ns`** par opération (0 allocation, 0 B/op).
- ⏱️ **Génération accélérée (x12.8) :** **`1.56 s`** pour 16 millions d'entrées.
- 📉 **Élimination du GC Churn :** Allocations divisées par 20 000 (808 allocs vs 16.1M en baseline) grâce aux types scalaires purs (`noscan`).
- 🧠 **Sondage L2 Cache (SoA) :** Tags 16-bit résidant à 100% dans le cache L2 CPU, évitant 99.998% des comparaisons de hashs 32 octets.

---

## 3. Guide d'Exécution & Outillage

Toutes les commandes d'ingénierie sont encapsulées dans le [Makefile](file:///home/chris/Projets/Cours/Sup%20de%20Vinci/2026-2027/M2-Performance-for-backend/Exercices/HashBreaker/Makefile) :

```bash
# 1. Exécuter les tests unitaires
make test

# 2. Exécuter les benchmarks mémoire et CPU
make bench

# 3. Analyse statistique de variance des benchmarks
make benchstat

# 4. Compiler et lancer le serveur HTTP précalculé (Terminal 1)
make run PORT=8080 DEPTH=4

# 5. Calculer l'empreinte SHA-256 d'un mot test (Terminal 2)
make hash WORD=Sh3n

# 6. Interroger l'endpoint HTTP de déchiffrement
make guess HASH=bd7d0ea8cf7ade4a446ba4efc46fd99071ec3f423770991ac51f70ec5a894dc7

# 7. Test d'injection de charge HTTP soutenu (2 000 req/s pendant 15s)
make load RATE=2000 DURATION=15s

# 8. Profiling CPU en direct sous charge (ouvre l'interface Web pprof sur :6060)
make load-and-profile PPROF_PORT=6060

# 9. Inspection de la mémoire résidente Heap
make profile-heap PPROF_PORT=6060
```

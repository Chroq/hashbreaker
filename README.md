# HashBreaker

Projet d'optimisation des performances en Go (**M2 Performance for Backend**).  
Cassage d'empreintes **SHA-256** par force brute et étude du compromis temps/mémoire (*Time-Memory Tradeoff*).

---

## 🎯 Espace Combinatoire

Charset : `[a-zA-Z0-9@]` ($N = 63$ caractères).  
Espace total pour une profondeur $D$ : $\sum_{d=1}^{D} N^d$

- **Profondeur 3 (`z3D`)** : $\approx 2,5 \times 10^5$ combinaisons
- **Profondeur 4 (`Sh3n`)** : $\approx 1,57 \times 10^7$ combinaisons
- **Profondeur 5 (`Ak@l1`)** : $\approx 9,92 \times 10^8$ combinaisons

---

## 📊 Synthèse des Résultats

| Étape | Approche | Génération ($d=4$) | Recherche ($d=4$) | RAM ($d=4$) | Conclusion |
| :--- | :--- | :---: | :---: | :---: | :--- |
| **1** | Récursif naïf | N/A | ~9.20 s | Saturée (GC) | Rejeté : allocations massives de `string` |
| **1** | **Itératif (odomètre)** | N/A | **~2.40 s** | **~0 B** | **Retenu : 3.8x plus rapide, zéro allocation** |
| **2** | Référentiel **Slice** (`[]Candidate`) | **~5.33 s** | $O(N)$ séquentiel | **~1.76 GB** | Génération 2.2x plus rapide, 2.1x moins de RAM |
| **2** | Référentiel **Map** (`map[[32]byte]string`) | ~11.93 s | **$O(1)$ instantané** | ~3.82 GB | Recherche immédiate mais surcoût mémoire élevé |

---

## 🔬 Enseignements Clés

### Étape 1 : Récursif vs Itératif
- **Récursif :** Création d'une nouvelle `string` à chaque feuille $\rightarrow$ saturation du Garbage Collector.
- **Itératif :** Réutilisation d'un buffer `[]byte` unique en mémoire $\rightarrow$ 0 allocation, cache CPU optimal.

### Étape 2 : Pré-calcul Slice vs Map
- **Génération :** La `Slice` est **2.2x plus rapide** grâce à l'allocation contiguë en mémoire.
- **Recherche :** La `Map` offre un accès **$O(1)$ instantané**, contre un parcours $O(N)$ pour la `Slice`.
- **Limite :** À $d=5$, le stockage brut en RAM dépasserait les ~100 GB. Nécessite des approches compressées (Rainbow Tables).

---

## 🛠️ Commandes (via Makefile)

| Action | Commande Makefile | Commande brute équivalente |
| :--- | :--- | :--- |
| **Démarrer le serveur** | `make run` | `go build -o bin/hashbreaker-server ./cmd/srv && ./bin/hashbreaker-server` |
| **Test de charge (Vegeta)** | `make load` | `echo "GET http://localhost:8080/guess?word=z3D" \| vegeta attack -duration=15s -rate=2000 \| vegeta report` |
| **Charge + Profiling Web CPU** | `make load-and-profile` | Lance Vegeta en arrière-plan et ouvre pprof sur `http://localhost:6060` |
| **Profil Mémoire (Heap)** | `make profile-heap` | `go tool pprof -http=:6060 http://localhost:8080/debug/pprof/heap` |
| **Profil CPU ponctuel** | `make profile-cpu` | `go tool pprof -http=:6060 http://localhost:8080/debug/pprof/profile?seconds=10` |
| **Benchmarks + Benchstat** | `make benchstat` | `go test -bench=. -benchmem -count=6 ./pkg > bench.txt && benchstat bench.txt` |
| **Tests unitaires** | `make test` | `go test -v ./...` |

> 💡 **Variables configurables :** `make run PORT=9000 DEPTH=4`, `make load WORD=Sh3n RATE=5000 DURATION=20s`.




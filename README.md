# HashBreaker

Projet d'optimisation des performances en Go (**M2 Performance for Backend**).  
Cassage d'empreintes **SHA-256** par force brute et étude du compromis temps/mémoire (_Time-Memory Tradeoff_).

---

## Espace Combinatoire

Charset : `[a-zA-Z0-9@]` ($N = 63$ caractères).  
Espace total pour une profondeur $D$ : $\sum_{d=1}^{D} N^d$

- **Profondeur 3 (`z3D`)** : $\approx 2,5 \times 10^5$ combinaisons
- **Profondeur 4 (`Sh3n`)** : $\approx 1,57 \times 10^7$ combinaisons
- **Profondeur 5 (`Ak@l1`)** : $\approx 9,92 \times 10^8$ combinaisons

---

## Synthèse des Résultats

### 1. Comparatif Global des Approches

| Étape | Approche                                    | Génération ($d=4$) |   Recherche ($d=4$)   | RAM ($d=4$)  | Conclusion                                     |
| :---- | :------------------------------------------ | :----------------: | :-------------------: | :----------: | :--------------------------------------------- |
| **1** | Récursif naïf                               |        N/A         |        ~9.20 s        | Saturée (GC) | Rejeté : allocations massives de `string`      |
| **1** | **Itératif**                                |        N/A         |      **~2.40 s**      |   **~0 B**   | **Retenu : 3.8x plus rapide, zéro allocation** |
| **2** | Référentiel **Slice** (`[]Candidate`)       |    **~5.33 s**     |   $O(N)$ séquentiel   | **~1.76 GB** | Génération 2.2x plus rapide, 2.1x moins de RAM |
| **2** | Référentiel **Map** (`map[[32]byte]string`) |      ~11.93 s      | **$O(1)$ instantané** |   ~3.82 GB   | Recherche immédiate mais surcoût mémoire élevé |

### 2. Métriques Détaillées de Référence (Baseline `MapReferential`)

_Mesures initiales via `go test -bench=. -benchmem -count=6` analysées avec `benchstat`._

| Benchmark                     | Temps par op (`sec/op`) | Mémoire par op (`B/op`) | Allocations (`allocs/op`) |
| :---------------------------- | :---------------------: | :---------------------: | :-----------------------: |
| **Génération $d=3$ (`z3D`)**  |    `255.9 ms ± 12%`     |       `56.79 MiB`       |         `256 083`         |
| **Génération $d=4$ (`Sh3n`)** |     `19.92 s ± 15%`     |       `3.553 GiB`       |       `16 137 750`        |
| **Recherche $d=3$ (`z3D`)**   |     `34.70 ns ± 6%`     |          `0 B`          |            `0`            |
| **Recherche $d=4$ (`Sh3n`)**  |    `34.54 ns ± 20%`     |          `0 B`          |            `0`            |

### 3. Optimisations Appliquées

#### Principes et Mécanismes Pédagogiques

1. **Pré-allocation de la capacité de la Map (_Capacity Hint_) :**
   - _Pourquoi ?_ Une `map` Go sans capacité initiale démarre avec une petite taille. Au fur et à mesure des insertions, elle double constamment sa table de hachage, copiant et ré-allouant des millions d'entrées (_churn_ mémoire).
   - _Gain :_ Passer la capacité totale exacte $\sum_{d=1}^{D} |Charset|^d$ à l'initialisation supprime toutes les réallocations intermédiaires et **divise par 2 l'empreinte mémoire**.

2. **Odomètre d'octets différentiel (_In-place buffer_) :**
   - _Pourquoi ?_ L'ancienne version reconstruisait l'intégralité du mot à partir d'un tableau d'indices à chaque itération.
   - _Gain :_ La nouvelle approche met à jour uniquement l'octet qui a changé (dans plus de 98% des cas, seul le dernier caractère est incrémenté), épargnant des millions d'opérations de copie en mémoire.

3. **Décodage Hexadécimal Zéro-Allocation (_Stack-allocated_) :**
   - _Pourquoi ?_ L'appel `hex.DecodeString` alloue une nouvelle slice `[]byte` sur le tas (_heap_) à chaque requête HTTP.
   - _Gain :_ Le décodage direct remplit directement un tableau fixe `[32]byte` sur la pile (_stack_), garantissant **0 allocation mémoire** sur le chemin critique du serveur.

4. **Sérialisation JSON Streamée sans Réflexion :**
   - _Pourquoi ?_ Le package `encoding/json` standard analyse les structs par réflexion (_reflection_), ce qui génère des allocations et consomme des cycles CPU sous forte charge.
   - _Gain :_ L'écriture directe du JSON dans le flux de réponse évite la réflexion et réduit la latence des requêtes HTTP.

#### Comparatif `benchstat` (Baseline vs Optimisé)

| Métrique                        |  Baseline   |    Optimisé     |          Évolution / Gain          |
| :------------------------------ | :---------: | :-------------: | :--------------------------------: |
| **Mémoire Génération $d=3$**    | `56.79 MiB` |   `28.79 MiB`   |  **-49.30% (RAM divisée par 2)**   |
| **Mémoire Génération $d=4$**    | `3.553 GiB` |   `1.811 GiB`   |  **-49.04% (RAM divisée par 2)**   |
| **Temps Génération $d=3$**      | `255.9 ms`  |   `206.2 ms`    |      **-19.45% plus rapide**       |
| **Temps Génération $d=4$**      |  `19.92 s`  |    `18.52 s`    |       **-7.03% plus rapide**       |
| **Recherche ($d=3, d=4$)**      | `34.70 ns`  |   `35.27 ns`    |   **$O(1)$ invariant (0 alloc)**   |
| **Requête HTTP `/guess` (Hex)** |     N/A     | **`933 ns/op`** | **0 alloc sur le chemin critique** |

### Étape 3 : Optimisation du Runtime et Zéro-Allocation

- **Éviter le Heap Churn :** Fournir des indices de capacité aux collections (`map`, `slice`) élimine le coût des redimensionnements dynamiques.
- **Chemin critique Web :** Remplacer les décodeurs et sérialiseurs génériques par du traitement direct sur la pile (_stack_) permet d'atteindre des temps de réponse sous la microseconde sans déclencher le GC.

---

## 🛠️ Commandes (via Makefile)

| Action                         | Commande Makefile       | Commande brute équivalente                                                                                   |
| :----------------------------- | :---------------------- | :----------------------------------------------------------------------------------------------------------- |
| **Démarrer le serveur**        | `make run`              | `go build -o bin/hashbreaker-server ./cmd/srv && ./bin/hashbreaker-server`                                   |
| **Test de charge (Vegeta)**    | `make load`             | `echo "GET http://localhost:8080/guess?word=z3D" \| vegeta attack -duration=15s -rate=2000 \| vegeta report` |
| **Charge + Profiling Web CPU** | `make load-and-profile` | Lance Vegeta en arrière-plan et ouvre pprof sur `http://localhost:6060`                                      |
| **Profil Mémoire (Heap)**      | `make profile-heap`     | `go tool pprof -http=:6060 http://localhost:8080/debug/pprof/heap`                                           |
| **Profil CPU ponctuel**        | `make profile-cpu`      | `go tool pprof -http=:6060 http://localhost:8080/debug/pprof/profile?seconds=10`                             |
| **Benchmarks + Benchstat**     | `make benchstat`        | `go test -bench=. -benchmem -count=6 ./pkg > bench.txt && benchstat bench.txt`                               |
| **Tests unitaires**            | `make test`             | `go test -v ./...`                                                                                           |

> 💡 **Variables configurables :** `make run PORT=9000 DEPTH=4`, `make load WORD=Sh3n RATE=5000 DURATION=20s`.

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

| Étape | Approche                                       | Génération ($d=4$) |   Recherche ($d=4$)   | RAM ($d=4$)  | Conclusion                                                     |
| :---- | :--------------------------------------------- | :----------------: | :-------------------: | :----------: | :------------------------------------------------------------- |
| **1** | Récursif naïf                                  |        N/A         |        ~9.20 s        | Saturée (GC) | Rejeté : allocations massives de `string`                      |
| **1** | **Itératif**                                   |        N/A         |      **~2.40 s**      |   **~0 B**   | **Retenu : 3.8x plus rapide, zéro allocation**                 |
| **2** | Référentiel **Slice** (`[]Candidate`)          |    **~5.33 s**     |   $O(N)$ séquentiel   | **~1.76 GB** | Génération 2.2x plus rapide, 2.1x moins de RAM                 |
| **2** | Référentiel **Map V0** (`map[[32]byte]string`) |      ~19.92 s      | **$O(1)$ instantané** |   ~3.55 GB   | Recherche immédiate mais génération lente et surcoût mémoire   |
| **3** | Référentiel **Map V1** (Capacity Hint + Odo)   |      ~18.52 s      | **$O(1)$ instantané** |   ~1.81 GB   | RAM divisée par 2, zéro réallocation dynamique                 |
| **4** | **Map V2 (Pointer-free + Goroutines)**         |    **~1.88 s**     | **$O(1)$ instantané** | **~1.51 GB** | **10.5x plus rapide, -99.6% allocations, GC footprint nul**     |

### 2. Métriques Détaillées de Référence (Baseline `MapReferential`)

_Mesures initiales via `go test -bench=. -benchmem -count=6` analysées avec `benchstat`._

| Benchmark                     | Temps par op (`sec/op`) | Mémoire par op (`B/op`) | Allocations (`allocs/op`) |
| :---------------------------- | :---------------------: | :---------------------: | :-----------------------: |
| **Génération $d=3$ (`z3D`)**  |    `255.9 ms ± 12%`     |       `56.79 MiB`       |         `256 083`         |
| **Génération $d=4$ (`Sh3n`)** |     `19.92 s ± 15%`     |       `3.553 GiB`       |       `16 137 750`        |
| **Recherche $d=3$ (`z3D`)**   |     `34.70 ns ± 6%`     |          `0 B`          |            `0`            |
| **Recherche $d=4$ (`Sh3n`)**  |    `34.54 ns ± 20%`     |          `0 B`          |            `0`            |

---

### 3. Optimisations Appliquées

#### Principes et Mécanismes Pédagogiques

1. **Pré-allocation de la capacité de la Map (_Capacity Hint_) :**
   - _Pourquoi ?_ Une `map` Go sans capacité initiale démarre avec une petite taille. Au fur et à mesure des insertions, elle double constamment sa table de hachage, copiant et ré-allouant des millions d'entrées (_churn_ mémoire).
   - _Gain :_ Passer la capacité totale exacte $\sum_{d=1}^{D} |Charset|^d$ à l'initialisation supprime toutes les réallocations intermédiaires et **divise par 2 l'empreinte mémoire**.

2. **Odomètre d'octets différentiel (_In-place buffer_) :**
   - _Pourquoi ?_ L'ancienne version reconstruisait l'intégralité du mot à partir d'un tableau d'indices à chaque itération.
   - _Gain :_ La nouvelle approche met à jour uniquement l'octet qui a changé (dans plus de 98% des cas, seul le dernier caractère est incrémenté), épargnant des millions d'opérations de copie en mémoire.

3. **Map Pointer-free (`[MaxDepth]byte` / `[8]byte`) :**
   - _Pourquoi ?_ Le type `string` en Go est composé d'un en-tête (pointeur + longueur) pointant vers un tableau sur le tas (_heap_). Pour $d=4$, cela forçait le Garbage Collector à scanner et tracer **16 millions de pointeurs distincts**.
   - _Gain :_ En remplaçant `string` par un tableau fixe `[8]byte`, la clé `[32]byte` et la valeur `[8]byte` ne contiennent **aucun pointeur**. Le runtime Go marque immédiatement les buckets de la table de hachage en `noscan`. Le nombre d'allocations à la génération chute de **16,1 millions à seulement 66 000** (**-99.59%**), et le coût de scan du GC tombe à **0**.

4. **Parallélisation multi-cœurs via Sharded Maps (Goroutines) :**
   - _Pourquoi ?_ La boucle de génération était strictement séquentielle (mono-thread), sous-exploitant les processeurs multi-cœurs modernes.
   - _Gain :_ L'espace combinatoire est découpé en tâches distribuées dynamiquement à un pool de $N$ workers (`runtime.GOMAXPROCS(0)`). Pour éliminer toute contention de verrouillage (_lock contention_), la table est partitionnée en **256 shards indépendants** indexés directement par le premier octet de l'empreinte `hash[0]` (distribution SHA-256 parfaitement uniforme) avec insertion par micro-lots (_batching_). Le temps de génération passe de **19.92 s à 1.88 s** (**10.5x plus rapide**).

5. **Décodage Hexadécimal & Flux JSON Zéro-Allocation sur le Serveur HTTP :**
   - _Pourquoi ?_ `hex.DecodeString` et `json.Marshal` par réflexion allouent à chaque requête HTTP.
   - _Gain :_ Décodage direct dans un tableau fixe sur la pile (_stack_) et écriture directe dans le flux de réponse : **0 allocation** sur le chemin critique du serveur HTTP.

6. **Contrôle de flux par Tagged Switch (`switch d`) :**
   - _Pourquoi ?_ Les chaînes `if-else if` successives réévaluent séquentiellement la même variable au runtime, générant des sauts conditionnels redondants.
   - _Gain :_ Le compilateur Go génère une table de saut directe (_jump table_) ou un dispatch optimal, éliminant les comparaisons superflues et maximisant l'efficacité de la prédiction de branchement CPU.

7. **Coordination Lock-Free des Workers (`sync/atomic.Uint32`) :**
   - _Pourquoi ?_ Distribuer dynamiquement les tâches aux $N$ workers via des canaux (`chan`) ou des verrous (`sync.Mutex`) induit de la contention, des allocations ou des changements de contexte noyau (_context switches_).
   - _Gain :_ L'utilisation de primitives atomiques typées modernes `atomic.Uint32` (`taskCounter.Add(1)`) permet un ordonnancement _lock-free_ en **$O(1)$ en quelques cycles d'horloge CPU**, sans le moindre blocage thread ni contention mémoire.

---

### 4. Comparatifs `benchstat`

#### Comparatif V2 vs Baseline (Gain Global)

```text
                                     │ bench_baseline.txt │        bench_optimized_v2.txt         │
                                     │       sec/op       │    sec/op     vs base                 │
NewReferential/z3D_MapReferential-8         255.95m ± 12%   29.73m ± 23%  -88.38% (8.6x plus vite)
NewReferential/Sh3n_MapReferential-8         19.917 ± 15%    1.887 ± 18%  -90.52% (10.5x plus vite)
Get/z3D_MapReferential-8                     34.70n ±  6%   26.33n ±  2%  -24.11% (p=0.002 n=6)
Get/Sh3n_MapReferential-8                    34.54n ± 20%   20.85n ±  9%  -39.63% (p=0.002 n=6)

                                     │ bench_baseline.txt │        bench_optimized_v2.txt         │
                                     │        B/op        │     B/op      vs base                 │
NewReferential/z3D_MapReferential-8        56.79Mi ± 0%     34.38Mi ± 0%  -39.45% (p=0.002 n=6)
NewReferential/Sh3n_MapReferential-8       3.553Gi ± 0%     1.512Gi ± 0%  -57.45% (p=0.002 n=6)
Get/z3D_MapReferential-8                     0.000 ± 0%       0.000 ± 0%        ~ (0 alloc / 0 B)
Get/Sh3n_MapReferential-8                    0.000 ± 0%       0.000 ± 0%        ~ (0 alloc / 0 B)

                                     │ bench_baseline.txt │        bench_optimized_v2.txt         │
                                     │     allocs/op      │  allocs/op   vs base                  │
NewReferential/z3D_MapReferential-8       256.083k ± 0%     1.571k ± 0%  -99.39% (p=0.002 n=6)
NewReferential/Sh3n_MapReferential-8     16137.75k ± 0%     66.09k ± 0%  -99.59% (p=0.002 n=6)
```

#### Synthèse par Version

| Métrique / Version              | Baseline (`V0`) | Optimisé `V1` | **Optimisé `V2` (Actuel)** |      Gain total (`V0` $\to$ `V2`)      |
| :------------------------------ | :-------------: | :-----------: | :------------------------: | :------------------------------------: |
| **Temps Génération $d=3$**      |   `255.9 ms`    |  `206.2 ms`   |        **`29.7 ms`**       |  **-88.4% (8.6x plus rapide)**         |
| **Temps Génération $d=4$**      |    `19.92 s`    |   `18.52 s`   |        **`1.88 s`**        |  **-90.5% (10.5x plus rapide)**        |
| **Mémoire Génération $d=4$**    |   `3.553 GiB`   |  `1.811 GiB`  |       **`1.512 GiB`**      |  **-57.5% (divisée par 2.3)**          |
| **Allocations Génération $d=4$**| `16 137 750`    | `16 072 515`  |        **`66 087`**        |  **-99.59% (divisées par 244)**        |
| **Recherche $d=4$ (`Sh3n`)**    |   `34.54 ns`    |  `37.77 ns`   |        **`20.85 ns`**      |  **-39.6% plus rapide (0 alloc, 0 B)** |
| **Requête HTTP `/guess` (Hex)** |       N/A       |  `933 ns/op`  |      **`206.2 ns/op`**     |  **0 alloc sur le chemin critique**    |

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


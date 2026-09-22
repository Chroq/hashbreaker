# HashBreaker - Optimisation des Performances en Go

Projet réalisé dans le cadre du cours **M2 - Performance for Backend**.  
L'objectif est d'explorer et de mesurer l'impact de différentes techniques d'optimisation (algorithmiques, gestion mémoire, concurrence, etc.) sur un cas concret et intensif en calculs : le cassage d'empreintes cryptographiques **SHA-256** par force brute.

---

## 🎯 Problématique & Espace Combinatoire

Pour une empreinte SHA-256 (`[32]byte`), on cherche le mot en clair d'origine composé à partir d'un jeu de caractères (`charset = [a-zA-Z0-9@]`, soit $N = 63$ caractères) jusqu'à une profondeur maximale $D$.

L'espace de recherche total pour une profondeur $D$ correspond à :
$$\sum_{d=1}^{D} N^d$$

- **Profondeur 3 (`z3D`)** : $\approx 250\ 047$ combinaisons
- **Profondeur 4 (`Sh3n`)** : $\approx 15\ 752\ 961$ combinaisons
- **Profondeur 5 (`Ak@l1`)** : $\approx 992\ 436\ 543$ combinaisons

---

## 📊 Synthèse des Étapes d'Optimisation

| Étape                 | Approche / Technique                    | Phase Génération ($d=4$) | Phase Recherche ($d=4$) |   Mémoire Allouée ($d=4$)   |
| :-------------------- | :-------------------------------------- | :----------------------: | :---------------------: | :-------------------------: |
| **Étape 1 (Base)**    | Récursif naïf                           | N/A (calcul à la volée)  |         ~9.20 s         |   Très élevé (GC saturé)    |
| **Étape 1 (Retenue)** | Itératif / Compteur                     | N/A (calcul à la volée)  |       **~2.40 s**       |        **Quasi nul**        |
| **Étape 2 (Array)**   | Référentiel Tableau (`[]Candidate`)     |       **~5.33 s**        |    $O(N)$ séquentiel    | **~1.76 GB** (16.0M allocs) |
| **Étape 2 (Map)**     | Référentiel Map (`map[[32]byte]string`) |         ~11.93 s         |  **$O(1)$ instantané**  |   ~3.82 GB (16.1M allocs)   |
| _Étape 3 (À venir)_   | _..._                                   |          _..._           |          _..._          |            _..._            |

---

## 🔬 Détail des Étapes

### Étape 1 : Choix du Paradigme de Base (Itératif vs Récursif)

#### 1. Description des Approches Naïves

- **Récursive (`BruteForceRecursive`)** : Parcours en profondeur (DFS). À chaque niveau de récursion, une nouvelle chaîne est créée par concaténation (`current + string(charset[i])`), puis convertie en `[]byte` lors du hash.
- **Itérative (`BruteForceIterative`)** : Compteur en base $N$ ("odomètre / compteur kilométrique"). Un unique buffer `buf := make([]byte, d)` est alloué par longueur et modifié directement en place (`buf[i] = charset[idx]`).

#### 2. Mesures Comparatives

| Métrique                               |   Version Récursive    | Version Itérative | Constat / Différence                                  |
| :------------------------------------- | :--------------------: | :---------------: | :---------------------------------------------------- |
| **Temps d'exécution (`z3D`)**          |        ~150 ms         |    **~40 ms**     | **~3.7x plus rapide**                                 |
| **Temps d'exécution (`Sh3n`)**         |        ~9.20 s         |    **~2.40 s**    | **~3.8x plus rapide**                                 |
| **Allocations (`B/op` & `allocs/op`)** |       Très élevé       |   **Quasi nul**   | Élimination des millions d'allocations temporaires    |
| **Pression Garbage Collector (GC)**    |       Très forte       |     **Nulle**     | Aucun ramasse-miettes déclenché pendant la boucle     |
| **Stack overhead**                     | Oui (appels imbriqués) |      **Non**      | Boucle plate hautement optimisable par le compilateur |

#### 3. Analyse & Décision

- L'immutabilité des strings en Go provoquait des millions d'allocations éphémères en récursif.
- L'approche **itérative** élimine les allocations dans la boucle interne et sert de base de référence.

---

### Étape 2 : Pré-calcul & Référentiels (Array vs Map)

Dans cette étape, on explore la stratégie du compromis temps/mémoire (_Time-Memory Tradeoff_) : pré-calculer et stocker l'ensemble des paires `(hash, mot)` dans une structure en mémoire pour accélérer les recherches ultérieures.

Deux structures de stockage ont été testées :

1. **`ArrayReferential`** : Stockage linéaire dans un slice de structures `[]Candidate` avec recherche linéaire $O(N)$.
2. **`MapReferential`** : Table de hachage Go `map[[32]byte]string` avec recherche en temps constant $O(1)$.

#### 1. Mesures : Phase de Génération (`New*Referential`)

Résultats observés sur la profondeur $d = 4$ (`Sh3n`, ~15.7M combinaisons) :

| Structure              | Temps de Génération | Mémoire Allouée (`B/op`) | Nombre d'Allocations |
| :--------------------- | :-----------------: | :----------------------: | :------------------: |
| **`ArrayReferential`** |     **~5.33 s**     |       **~1.76 GB**       |   **16.0M allocs**   |
| **`MapReferential`**   |      ~11.93 s       |         ~3.82 GB         |     16.1M allocs     |

> **Constat génération :** La construction de la Map est **~2.2x plus lente** et consomme **~2.1x plus de RAM** que le tableau à cause de la structure interne des buckets de la table de hachage Go, du surcoût d'en-tête et des réallocations dynamiques.

#### 2. Mesures : Phase de Recherche (`Get`)

| Structure              | Complexité Algorithmique |    Temps de recherche (`Get`)    | Comportement avec la taille                               |
| :--------------------- | :----------------------: | :------------------------------: | :-------------------------------------------------------- |
| **`ArrayReferential`** |          $O(N)$          | Variable (dépend de la position) | Se dégrade linéairement avec la taille du référentiel     |
| **`MapReferential`**   |        **$O(1)$**        |       **Quasi instantané**       | Temps d'accès constant quel que soit le volume de données |

#### 3. Conclusions de l'Étape 2

1. **Génération : Avantage au Tableau (`ArrayReferential`)**
   - Le tableau alloue un bloc contigu (`[]Candidate`), offrant une meilleure localité spatiale du cache CPU et moins d'overhead mémoire brut.
2. **Recherche : Avantage écrasant à la Map (`MapReferential`)**
   - L'accès $O(1)$ permet de retrouver n'importe quel hash immédiatement une fois la map en mémoire.
3. **Limite majeure (Goulet d'étranglement mémoire) :**
   - À $d=4$, la Map consomme déjà **~3.8 GB** de mémoire vive.
   - À $d=5$ (~1 milliard de combinaisons), cette approche nécessiterait **plusieurs dizaines à centaines de gigaoctets de RAM**, rendant le stockage en mémoire vive brut non viable sans techniques de compactage (Rainbow tables, filtres de Bloom, compression ou stockage disque).

---

## 🛠️ Commandes Utiles

- **Exécuter le programme :**

  ```bash
  go run main.go z3D
  ```

- **Lancer la suite de tests :**

  ```bash
  go test -v ./...
  ```

- **Lancer les benchmarks mémoire et temps :**

  ```bash
  go test -bench=. -benchmem ./...
  ```

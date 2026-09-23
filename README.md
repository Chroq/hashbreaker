# HashBreaker — Optimisation Haute Performance en Go

Projet pédagogique pour le cours **M2 Performance for Backend**.  
Étude pratique de l'ingénierie des performances en Go : calcul intensif, gestion fine de la mémoire, élimination de la pression sur le Garbage Collector et parallélisme multi-cœurs à travers le cassage d'empreintes **SHA-256** par force brute (_Time-Memory Tradeoff_).

---

## 1. Problématique & Espace Combinatoire

On cherche à retrouver le mot d'origine correspondant à une empreinte SHA-256 issue d'un alphabet donné.

- **Alphabet ($N = 63$) :** `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@`
- **Nombre total de combinaisons pour une profondeur $D$ :**
  $$\sum_{d=1}^{D} N^d$$

| Profondeur $D$ | Exemple cible | Nombre de combinaisons | Taille brute SHA-256 (32B) |
| :------------- | :------------ | :--------------------- | :------------------------- |
| **1**          | `@`           | $63$                   | $2 \text{ KB}$             |
| **2**          | `a1`          | $4\ 032$               | $129 \text{ KB}$           |
| **3**          | `z3D`         | $254\ 079$             | $8.1 \text{ MB}$           |
| **4**          | `Sh3n`        | $16\ 007\ 040$         | $512.2 \text{ MB}$         |
| **5**          | `Ak@l1`       | $1\ 008\ 443\ 583$     | $32.2 \text{ GB}$          |

---

## 2. Paliers d'Optimisation Pédagogiques

### Étape 1 : Algorithme de parcours (Récursivité vs Odomètre)

- **Approche naïve :** Génération récursive créant une nouvelle chaîne `string` à chaque combinaison.
- **Problème :** Des centaines de millions d'allocations éphémères sur le tas (_heap churn_), saturation du Garbage Collector.
- **Solution :** Odomètre itératif sur un buffer fixe `[MaxDepth]byte`. Seul l'octet modifié est incrémenté sur place, sans aucune allocation.

### Étape 2 : Élimination des pointeurs (_Pointer-Free Types & GC Tagging_)

- **Approche naïve :** `map[[32]byte]string`
- **Problème :** L'en-tête de `string` contient un pointeur vers le tas. À $d=4$, le GC de Go doit tracer et scanner **16 millions de pointeurs distincts**, provoquant d'importantes pauses de marquage.
- **Solution :** Remplacement par `map[[32]byte][8]byte`. Clés et valeurs étant des tableaux de scalaires purs (sans pointeurs), le runtime Go marque les buckets de la table en `noscan`. Le Garbage Collector ignore totalement cette structure en mémoire.

### Étape 3 : Dimensionnement & Sharding Anti-Contention

- **Approche :** Découpage en **256 shards indépendants** indexés par le premier octet `hash[0]`.
- **Dimensionnement :** Pré-allocation exacte par shard (`(totalComb + 255) / 256`).
- **Micro-lots :** Tampon local par thread (`batchSize = 64`) divisant la fréquence des verrous par 64.

### Étape 4 : Parallélisme multi-cœurs & Distribution Atomique

- **Dimensionnement des workers :** `runtime.NumCPU()` goroutines pour saturer 100% des cœurs physiques.
- **Distribution des tâches :** Ordonnancement dynamique par compteur atomique `sync/atomic.Uint32` (`taskIdx.Add(1)`).

### Étape 5 : Table de Hachage Plate SoA Tagguée & Sortie Texte (V5)

- **Fast Path SoA Sub-15ns :** Remplacement de la map Go standard par une table de hachage plate à adressage ouvert (_Structure-of-Arrays_). L'empreinte SHA-256 étant cryptographiquement uniforme, aucun hachage n'est recalculé à l'exécution :
  - `hash[0]` indexe l'un des 256 shards indépendants.
  - `uint16(hash[1:3])` forme un tag de métadonnées compact (2 octets).
  - `uint32(hash[3:7]) & mask` donne le slot de départ.
  - **Sondage L2-Cache :** Le tableau des tags (`[]uint16`, 262 Ko par shard) réside intégralement dans le cache L2 du processeur. La comparaison de la clé 32 octets n'a lieu que si le tag correspond ($P \approx 1/65536$).
- **Sortie Texte Native Zéro-Allocation :** Envoi HTTP direct sous forme de texte décodé (ex: `Sh3n`) sans allocation sur le tas, en remplacement du tableau brut d'octets.
- **Pureté Architecturale & Zéro-Verrou :** Table 100% immuable en lecture sans verrous concurrents, ni mécanismes de calcul dynamique à la volée.

---

## 3. Synthèse des Performances

### Évolution par version pour $d=4$ (`Sh3n`, 16 millions d'entrées)

| Version | Stratégie                                                | Temps de Génération |  Empreinte RAM  | Allocations  | Temps de Recherche (`GetRaw`) |
| :------ | :------------------------------------------------------- | :-----------------: | :-------------: | :----------: | :---------------------------: |
| **V0**  | Baseline mono-thread (`map[[32]byte]string`)             |      `19.92 s`      |   `3.553 GiB`   | `16 137 750` |          `34.54 ns`           |
| **V1**  | Préallocation de capacité + Odomètre                     |      `18.52 s`      |   `1.811 GiB`   | `16 072 515` |          `37.77 ns`           |
| **V2**  | Sharded Maps (256 shards + Mutex + Micro-lots)           |      `1.80 s`       |   `1.506 GiB`   |   `66 070`   |          `18.80 ns`           |
| **V3**  | Tableaux Contigus Triés (`[256][]Entry` + Dichotomie)    |      `3.01 s`       | **`0.693 GiB`** |  **`287`**   |          `191.95 ns`          |
| **V4**  | Single Map + `sync.RWMutex` + Batched                    |      `7.65 s`       |   `1.501 GiB`   |   `65 710`   |          `31.38 ns`           |
| **V5**  | **SoA Tagged Flat Table (Fast-Path Pur & Sortie Texte)** |    **`1.56 s`**     |   `1.318 GiB`   |    `808`     | **`8.21 ns`** (0 alloc / 0 B) |

---

## 4. Analyse & Conclusions sur la V5

### 1. Dépassement des Performances de la V2 en Lecture (`8.21 ns`)

- **Élimination de l'overhead de la map standard Go :** La map standard (`map[[32]byte][8]byte`) invoquait `runtime.mapaccess1`, recalculant un hash AES-NI sur 32 octets et parcourant des chaînes de buckets indirectes (~39 ns).
- **Architecture Structure-of-Arrays (SoA) :** En séparant les tags (`[]uint16`), les hashs (`[][32]byte`) et les mots (`[][8]byte`), le sondage linéaire ne lit que 2 octets consécutifs par slot dans une tranche contiguë résidant en cache L2.
- **Gain statistique net mesuré face à la V2 :**
  - Temps de lecture (`GetRaw`) réduit de **-62.27%** (de `21.77 ns` en V2 à **`8.21 ns`** en V5).
  - Temps de génération réduit de **-25.95%** (`1.56 s` contre `2.10 s` en V2).
  - Nombre d'allocations divisé par 80 (**-98.78%**, passant de 66 072 à seulement **808** allocs).
  - Empreinte RAM réduite de **-12.49%** (`1.318 GiB` contre `1.506 GiB` en V2).
  - Lecture strictement garantie à **`0 B/op`** et **`0 allocs/op`**.

### 2. Inlining, Zéro-Verrou & Simplicité

- **Inlining garanti du Fast-Path :** La fonction `GetRaw` est inlinée par le compilateur Go directement dans le point d'appel. La résolution s'exécute en quelques cycles processeur sans allocation, ni saut de fonction, ni création de frame sur la pile.
- **Code épuré et déterministe :** En retirant le calcul à la volée, le code redevient totalement direct, prédictible et résistant aux pics de latence en production.
- **Sortie texte claire :** Les requêtes HTTP renvoient directement le mot déchiffré en clair sous forme de chaîne (`text/plain`), évitant toute manipulation de tableau d'octets côté client.

### 3. Bilan Global

La V5 cumule le meilleur de tous les mondes :

- **Vitesse de lecture record :** **`8.21 ns`**, la version la plus rapide du projet.
- **Génération la plus rapide :** **`1.56 s`** avec seulement 808 allocations sur le tas pour 16 millions d'entrées.
- **Code concis, lisible et robuste**.

---

## 5. Guide d'Exécution & Outillage

Toutes les commandes d'ingénierie sont encapsulées dans le Makefile:

```bash
# 1. Tests unitaires
make test

# 2. Exécution des benchmarks CPU et allocations
make bench

# 3. Analyse statistique des benchmarks
make benchstat

# 4. Compilation et lancement du serveur HTTP (Terminal 1)
make run PORT=8080 DEPTH=4

# 5. Calcul d'empreinte SHA-256 (Terminal 2)
make hash WORD=Sh3n

# 6. Requête de déchiffrement HTTP
make guess HASH=bd7d0ea8cf7ade4a446ba4efc46fd99071ec3f423770991ac51f70ec5a894dc7

# 7. Injection de charge HTTP avec Vegeta (2 000 req/s pendant 15s)
make load RATE=2000 DURATION=15s

# 8. Profiling CPU en direct sous charge (ouvre l'interface Web pprof sur :6060)
make load-and-profile PPROF_PORT=6060

# 9. Inspection de la Heap (RAM)
make profile-heap PPROF_PORT=6060
```

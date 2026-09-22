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

| Étape | Approche / Technique | Temps (`z3D` - $d=3$) | Temps (`Sh3n` - $d=4$) | Gain vs Naïf Base |
| :--- | :--- | :---: | :---: | :---: |
| **Étape 1 (Base)** | Récursif naïf (`BruteForceRecursive`) | ~150 ms | ~9.20 s | 1.0x (Référence) |
| **Étape 1 (Retenue)** | **Itératif / Compteur (`BruteForceIterative`)** | **~40 ms** | **~2.40 s** | **~3.8x** |
| *Étape 2 (À venir)* | *...* | *...* | *...* | *...* |

---

## 🔬 Détail des Étapes

### Étape 1 : Choix du Paradigme de Base (Itératif vs Récursif)

#### 1. Description des Approches Naïves
- **Récursive (`BruteForceRecursive`)** : Parcours en profondeur (DFS). À chaque niveau de récursion, une nouvelle chaîne est créée par concaténation (`current + string(charset[i])`), puis convertie en `[]byte` lors du hash.
- **Itérative (`BruteForceIterative`)** : Compteur en base $N$ ("odomètre / compteur kilométrique"). Un unique buffer `buf := make([]byte, d)` est alloué par longueur et modifié directement en place (`buf[i] = charset[idx]`).

#### 2. Mesures Comparatives

| Métrique | Version Récursive | Version Itérative | Constat / Différence |
| :--- | :---: | :---: | :--- |
| **Temps d'exécution (`z3D`)** | ~150 ms | **~40 ms** | **~3.7x plus rapide** |
| **Temps d'exécution (`Sh3n`)** | ~9.20 s | **~2.40 s** | **~3.8x plus rapide** |
| **Allocations (`B/op` & `allocs/op`)** | Très élevé | **Quasi nul** | Élimination des millions d'allocations temporaires |
| **Pression Garbage Collector (GC)** | Très forte | **Nulle** | Aucun ramasse-miettes déclenché pendant la boucle |
| **Stack overhead** | Oui (appels imbriqués) | **Non** | Boucle plate hautement optimisable par le compilateur |

#### 3. Analyse Technique
1. **Coût de l'immutabilité des strings :** En Go, chaque concaténation `string + string` alloue une nouvelle mémoire sur le Heap. Sur 15 millions de combinaisons (`Sh3n`), la récursion alloue et abandonne des dizaines de millions d'objets, saturant le Garbage Collector.
2. **Mutation in-place sur `[]byte` :** L'approche itérative travaille sur une tranche d'octets pré-allouée transmise directement à `sha256.Sum256(buf)` sans conversion intermédiaire.
3. **Suppression de la pile d'appels :** Le passage d'une arborescence d'appels de fonctions à une simple boucle avec arithmétique d'indices permet un meilleur inlining et évite le coût de création des frames de pile.

#### 4. Conclusion & Décision
> **Décision :** L'approche **itérative** est retenue comme socle de référence pour la suite du projet.  
> Elle offre un gain immédiat de **~3.8x** uniquement grâce à la suppression des allocations éphémères et de l'overhead de pile, sans même encore exploiter la concurrence ou les optimisations CPU.

---

<!-- Les étapes suivantes (Goroutines, Workers, Chunking, etc.) viendront s'ajouter ici -->

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

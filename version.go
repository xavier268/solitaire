package solitaire

// Historique des versions
// ========================
//
// Les mesures de performance sont dans bench/bench-v<VERSION>.txt ; la comparaison entre versions
// est dans bench/bench-stats.md (task bench, task bench:stat). Mesures : i7-10700, Go 1.25 à 1.27.
//
// v0.1.0 : première version, recherche brute force.
//   - Position (bitmap uint64 sur une grille 8x8), Point (ligne, colonne), BitIndex, Bit, Render, Print.
//   - Masques MASK_8x8, MASK_5x5, MASK_ENGLISHCROSS ; positions de départ START_5x5 et START_ENGLISHCROSS.
//   - Move, DoMove (applique un coup en O(1) avec des masques) et AllMoves (liste des coups d'un masque).
//   - NewGame, Solve et dfs : recherche en profondeur sans élagage, sans allocation pendant la recherche.
//   - Mesures : résolution de la croix anglaise ≈ 474 ms ; NewGame ≈ 1,60 µs, 6376 B, 9 allocs.
//
// v0.1.1 : optimisation de NewGame.
//   - AllMoves préalloue sa slice (borne haute : 4 coups par case) : 1 allocation au lieu de 8.
//   - NewGame : ≈ 0,89 µs, 3456 B, 2 allocs (−44 % en temps, −46 % en mémoire). Solve inchangée.
//
// v0.1.2 : plateau européen et cible « une seule bille ».
//   - MASK_EUROPEAN (37 cases), CENTER_EUROPEAN et START_EUROPEAN.
//   - SolveAnyEnd / dfsAnyEnd : avec une cible 0, on cherche à ne laisser qu'une seule bille, n'importe où.
//     NewGame accepte la cible 0 (capacité de solution = nombre de billes de départ − 1).
//   - SortMoves : tri des coups par distance au barycentre de la cible, sans allocation.
//     NON UTILISÉ : sans élagage, il rend la résolution de la croix anglaise bien plus longue
//     (plus de 100 s au lieu de ~0,5 s, que la distance soit mesurée sur from ou sur to,
//     en ordre croissant ou décroissant). Appel commenté dans NewGame.
//   - Benchmarks du plateau européen ; outils cmd/sysinfo (en-tête machine), cmd/version et
//     cmd/benchtable (tableaux ASCII issus de benchstat).
//   - Mesures : Solve croix anglaise ≈ 469 ms, NewGame croix ≈ 0,85 µs, NewGame européen ≈ 1,15 µs.
//
// v0.1.3 : détection des configurations insolubles et trou de départ européen corrigé.
//   - VerifyRuleOfThree : test de parité modulo 3 (deux coloriages diagonaux). Il prouve en O(1)
//     qu'une cible est inatteignable ; une réponse sans erreur ne garantit pas qu'une solution existe.
//     Une cible 0 est testée pour chaque case possible de la dernière bille.
//   - NewGame appelle VerifyRuleOfThree et renvoie une erreur si la configuration est insoluble :
//     la recherche ne part plus dans une exploration exhaustive vouée à l'échec.
//   - Plateau européen : le départ avec le trou au centre est INSOLUBLE (Bergholt), la recherche de
//     la v0.1.2 ne pouvait donc jamais finir. START_EUROPEAN a maintenant le trou en d3 (HOLE_EUROPEAN),
//     avec une solution connue en 35 coups (dernière bille en g5). CENTER_EUROPEAN est conservé.
//   - Coût de la règle de trois dans NewGame : négligeable (bench/bench-v0.1.3.txt) : ≈ 0,96 µs (croix)
//     et ≈ 1,26 µs (européen), comme en v0.1.2. Solve est inchangée (≈ 474 ms).
//   - La résolution du plateau européen par le DFS brute force est toujours trop longue (plus de 90 s) :
//     son benchmark est désactivé par défaut (BENCH_EUROPEAN_SOLVE=1 pour l'activer).
//     Piste suivante (réalisée en v0.1.4) : mémoriser les positions déjà perdantes.

// v0.1.4 : mémorisation des impasses.
//   - Game.dead : map des positions dont toutes les branches ont échoué, utilisée par dfs et dfsAnyEnd
//     pour ne jamais explorer deux fois la même position (les coups commutent : la même position
//     est atteignable par un très grand nombre de chemins).
//   - La map est allouée une seule fois (à la première recherche), puis vidée par clear() à chaque
//     recherche suivante en conservant sa mémoire : 0 allocation après la première résolution.
//     Sa taille est bornée par maxDeadPositions (4 194 304 positions, ≈ 100 Mo) ; au-delà, les nouvelles
//     impasses ne sont plus mémorisées (la recherche reste correcte, mais plus lente).
//   - Résolution de la croix anglaise : ≈ 474 ms -> ≈ 1,36 ms (−99,7 %, environ 350 fois plus rapide),
//     toujours 0 allocation par résolution. NewGame inchangée (≈ 1,0 µs croix, ≈ 1,3 µs européen).
//   - Le benchmark de la croix fait une première résolution hors mesure (création de la map).
//   - Test TestDeadMapAllocatedOnce : la map n'est créée qu'une fois et réutilisée.
//   - Plateau européen (trou en d3) : malgré la mémorisation, la recherche avec l'ordre de coups fixe
//     ne termine pas en 100 s (une recherche à ordre de coups aléatoire en trouve une en 0,1 s) ;
//     son benchmark reste désactivé par défaut. Pistes : ordre de coups aléatoire avec redémarrages
//     et budget de nœuds, ou meilleure heuristique d'ordre.

// v0.1.5 : élagage par fonction de pagode (pagoda.go).
//   - Game.pag : poids entiers par case (Fibonacci de la distance de Manhattan au centre) tels que pour
//     tout coup A saute B vers C, w(A)+w(B) >= w(C) : la somme des poids des pions ne fait que décroître.
//   - Centre : case la plus proche du barycentre de la cible, ou du mask si la cible est 0.
//     Seuil : poids de la cible, ou plus petit poids du mask si la cible est 0.
//   - initPagoda (privée) est appelée par NewGame, qui vérifie aussi la pagode du départ ;
//     elle contrôle la condition de pagode sur tous les coups du jeu.
//   - dfs et dfsAnyEnd rejettent immédiatement toute position sous le seuil (fail early).
//   - L'élagage seul ne suffit pas pour l'européen (cible 0 ou cibles précises d2, a5, d5, g5 : plus de 75 s).
//   - Recherche à redémarrages (Game.search, utilisée par Solve et SolveAnyEnd) : l'ordre des coups est mélangé
//     à chaque tentative, avec un budget de nœuds (4096, +3 % par redémarrage pour garantir la terminaison).
//     Une tentative interrompue ne marque aucune impasse ; les impasses déjà prouvées restent valables.
//     La graine aléatoire est tirée à chaque NewGame (Game.seed). Résultat non déterministe.
//   - Map des impasses pleine (maxDeadPositions) : elle est vidée au lieu de cesser de mémoriser.
//   - Plateau européen, cible 0 : 10 résolutions de 0,16 s à 4,2 s (typiquement ~1 s), contre plus de 100 s avant.
//   - BenchmarkSolveEuropean est réactivé (plus de variable BENCH_EUROPEAN_SOLVE) ; temps très variable d'une exécution à l'autre.
//   - Contrepartie : avec l'ordre aléatoire, la croix anglaise est bien plus lente qu'avec l'ordre fixe de la
//     v0.1.4 (≈ 250 à 340 ms au lieu de ≈ 1,4 ms, mesure rapide sur 3 passages) et Solve n'est plus sans
//     allocation (≈ 2000 à 3000 allocs, 17 à 27 Mo : la map grossit). NewGame passe à ≈ 2,3 µs, 4 allocs
//     (pagode, générateur aléatoire). Les mesures de référence sont à refaire avec task bench.

const VERSION = "0.1.5"

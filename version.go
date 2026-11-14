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
//   - NewGame est un peu plus lent (calcul des parités) : ≈ 1,0 µs (croix) et ≈ 1,4 µs (européen),
//     mesures ponctuelles à confirmer par task bench. Solve est inchangée.
//   - La résolution du plateau européen par le DFS brute force est toujours trop longue (plus de 90 s) :
//     son benchmark est désactivé par défaut (BENCH_EUROPEAN_SOLVE=1 pour l'activer).
//     Piste suivante : mémoriser les positions déjà perdantes (une recherche avec mémoire trouve la
//     solution européenne en 0,1 s).

const VERSION = "0.1.3"

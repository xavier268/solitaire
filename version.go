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

// v0.1.6 : pagode calculée par deltas, et première tentative à ordre fixe.
//   - Move.dPag : variation de pagode du coup, w(to) - w(from) - w(saut), calculée une fois pour toutes par
//     initPagoda (la condition de pagode équivaut à dPag <= 0). Le champ est placé dans le remplissage
//     de la structure : Move reste à 24 octets (placé en dernier, il la faisait passer à 32 octets et
//     augmentait de 33 % la mémoire de NewGame).
//   - dfs et dfsAnyEnd reçoivent la valeur de pagode de la position et la mettent à jour par une addition
//     (pag + m.dPag) au lieu de resommer les poids de tous les pions à chaque nœud ; la pagode du départ
//     n'est calculée qu'une fois par recherche.
//   - search : la 1re tentative utilise l'ordre fixe des coups (Game.validMoves, jamais modifié) avec un
//     budget de 2^17 = 131 072 nœuds (la croix en utilise 30 753, soit une marge ≈ 4 ; la tentative coûte
//     au plus ≈ 12 ms sur l'européen, contre ≈ 94 ms avec 2^20). Ce n'est qu'en cas d'échec qu'on passe aux
//     redémarrages à ordre aléatoire (copie mélangée Game.shuffled, allouée à la demande, budget 4096 puis +3 %).
//     Si la pagode ou l'ordre des coups change, revérifier que la croix tient dans firstBudget.
//   - Mesures (bench/bench-v0.1.6.txt, 10 passages) :
//     croix anglaise ≈ 1,58 ms, 0 octet, 0 allocation (v0.1.5 : ≈ 255 ms +/- 42 %, ≈ 8,6 Mo, 956 allocs ; la v0.1.4
//     faisait ≈ 1,36 ms) ; NewGame ≈ 1,63 µs (croix) et ≈ 2,02 µs (européen), 4 allocs, 3,4 et 4,3 KiB
//     (le champ dPag n'agrandit pas Move) ; Solve européen ≈ 1,9 s +/- 95 % (v0.1.5 : ≈ 2,4 s +/- 52 %) :
//     la dispersion est telle que les deux versions ne sont pas distinguables, ce temps dépend surtout de la graine.

// v0.1.7 : table de fins de partie (endgame.go). Les pagodas multiples sont abandonnées.
//   - Essai abandonné (code retiré) : jusqu'à 16 pagodas empaquetées en voies de 16 bits (état [4]uint64, deltas par
//     coup dans une table indexée par Move.id). Résultat : aucun gain. Sur l'européen cible 0, 12 graines fixes :
//     40,2 s (0 pagoda), 40,6 s (1), 40,1 s (5), 41,3 s (16). Les pagodas n'élaguent presque rien (la somme des poids
//     reste très au-dessus du seuil tant qu'il reste beaucoup de billes), et NewGame passait de ≈ 2 µs à 16-19 µs.
//     Le code revient donc exactement à la v0.1.6 (une seule pagode, Move.dPag).
//   - Table de fins de partie : chaque coup retire un pion, donc toute partie passe par une position à K pions.
//     On construit l'ensemble EXACT des positions à K pions qui atteignent la cible, par recherche arrière
//     (Position.UndoMove, inverse de DoMove) depuis les positions finales (cible précise, ou cases que la règle de
//     trois ne rejette pas si cible 0), couche par couche. Dans dfs et dfsAnyEnd, une position à K pions absente de
//     la table est perdante et coupée en O(1). La table est exacte : aucune solution n'est perdue (tests).
//   - Construite à la demande (Game.endgame, posSet = table de hachage à adressage ouvert), seulement quand la
//     tentative à ordre fixe échoue : la croix anglaise (≈ 1,6 ms, 0 allocation) et NewGame sont inchangés.
//   - K est fixé par maxEndgame = 2^18 positions par couche : K = 8 sur l'européen (77 008 positions, ≈ 4 Mo, quelques
//     ms). Couches de l'européen : K = 9 : 318 437 ; K = 10 : 1 196 082 ; K = 11 : 4 025 725.
//   - Mesures sur l'européen cible 0, 12 graines fixes : 37,6 s au total sans table (max 24,9 s), 17,0 s avec K = 8
//     (max 3,9 s), 17,2 s avec K = 9, 21,3 s avec K = 10, 63 s avec K = 11 (la construction coûte plus que le gain).
//     Sur 15 résolutions à graine aléatoire : moyenne ≈ 2,5 s, de 0,07 s à 10,5 s. La queue lourde est réduite
//     (25 s -> 10 s au pire) mais reste présente : la partie coûteuse est le milieu de la partie, pas la fin.
//   - Mesures (bench/bench-v0.1.7.txt, 10 passages) : croix anglaise ≈ 1,57 ms, 0 allocation ; NewGame ≈ 1,59 µs (croix)
//     et ≈ 1,98 µs (européen), 4 allocs, inchangés ; Solve européen ≈ 1,27 s +/- 170 % (v0.1.6 : ≈ 1,89 s +/- 95 %) :
//     dispersion trop grande pour distinguer les deux versions sur 10 passages. Le benchmark réutilise le même Game,
//     donc la table n'est construite qu'une fois (première itération) ; ≈ 150 Mo et ≈ 15 800 allocs par résolution
//     (map des impasses, table, copie mélangée des coups).
//   - Tests : endgame_test.go (UndoMove, posSet, exactitude de la table contre une recherche exhaustive, positions
//     d'une vraie solution toujours présentes).
//   - Pistes : symétries du plateau (clés canoniques des impasses), ordre des coups guidé.

const VERSION = "0.1.7"

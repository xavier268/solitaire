package solitaire

import (
	"fmt"
	"math/bits"
	"math/rand/v2"
	"slices"
)

type Game struct {
	startPos   Position // Start position
	maskPos    Position // Board limit
	endPos     Position // Target end position
	validMoves []Move   // List of valid moves (ordre fixe, jamais modifié après NewGame)
	solution   []Position

	// Positions déjà explorées sans succès (impasses), pour ne jamais les explorer deux fois.
	// La map est allouée une seule fois (à la première recherche), puis simplement vidée
	// par clear() à chaque nouvelle recherche : sa mémoire est conservée et réutilisée.
	dead map[Position]struct{}

	// Fonction de pagode (voir pagoda.go) : élagage des positions qui ne peuvent plus atteindre la cible.
	pag pagoda

	// Recherche à redémarrages : l'ordre des coups est mélangé à chaque tentative, avec un budget de nœuds
	// qui croît lentement à chaque échec. seed est tirée au hasard à chaque NewGame (et conservée pour reproduire
	// un comportement) ; rng est le générateur qui en dérive.
	seed     [2]uint64
	rng      *rand.Rand
	moves    []Move // coups utilisés par la tentative en cours : validMoves (1re tentative) ou shuffled
	shuffled []Move // copie de validMoves mélangée, allouée à la première tentative aléatoire seulement
	nodes    int    // nœuds explorés pendant la tentative en cours
	budget   int    // nombre maximal de nœuds de la tentative en cours
	aborted  bool   // vrai si la tentative en cours a épuisé son budget
	guided   bool   // la tentative en cours trie les coups par évaluation (voir guide.go)
	scratch  []cand // zone de travail du tri, une tranche par profondeur (allouée à la demande)

	// Table de fins de partie (voir endgame.go) : positions à endgameK pions qui atteignent la cible.
	// Construite à la demande, la première fois que la tentative à ordre fixe échoue.
	endgame      posSet
	endgameK     int  // nombre de pions des positions de la table ; 0 si pas de table
	endgameBuilt bool // la construction a déjà été tentée
	endgameOff   bool // désactive la table (tests et mesures)
}

// firstBudget est le budget de nœuds de la première tentative, faite avec l'ordre fixe des coups (ordre de AllMoves) :
// c'est très rapide sur la croix anglaise (30 753 nœuds en ordre fixe, soit une marge d'environ 4). Si elle échoue, on passe aux tentatives à ordre aléatoire.
const firstBudget = 1 << 17

// initialBudget est le budget de la première tentative aléatoire ; il croît de 3 % à chaque redémarrage
// (ce qui garantit la terminaison).
const initialBudget = 1 << 12

// endgameAfter : nombre de tentatives échouées après lequel la table de fins de partie est construite.
const endgameAfter = 16

// maxDeadPositions borne la mémoire de la map des impasses (environ 100 Mo au maximum).
// Quand elle est pleine, elle est vidée (la recherche reste correcte, mais perd ses impasses).
const maxDeadPositions = 1 << 22

// NewGame crée un jeu à partir du mask du board, de la position de départ et de la position cible.
// Renvoie une erreur si une des positions sort du mask, ou si la cible a plus de pions que le départ
// (chaque move retire exactement un pion), ou si la règle de trois (VerifyRuleOfThree) prouve que la cible
// est inatteignable. Une cible 0 signifie « une seule bille restante, n'importe où » (voir SolveAnyEnd) :
// la règle de trois est alors testée pour chaque case possible de la dernière bille.
func NewGame(maskPos Position, startPos Position, endPos Position) (Game, error) {

	if startPos&^maskPos != 0 {
		return Game{}, fmt.Errorf("la position de départ sort du masque (cases hors masque : %#x)", uint64(startPos&^maskPos))
	}
	if endPos&^maskPos != 0 {
		return Game{}, fmt.Errorf("la position d'arrivée sort du masque (cases hors masque : %#x)", uint64(endPos&^maskPos))
	}

	// Chaque move retire un pion : le nombre de coups entre start et end est fixé.
	// Cible vide (endPos == 0) : voir SolveAnyEnd, on cherche à ne laisser qu'un seul pion (n'importe où).
	endPegs := bits.OnesCount64(uint64(endPos))
	if endPos == 0 {
		endPegs = 1
	}
	maxMoves := bits.OnesCount64(uint64(startPos)) - endPegs
	if maxMoves < 0 {
		return Game{}, fmt.Errorf("la position d'arrivée a plus de pions (%d) que le départ (%d)",
			bits.OnesCount64(uint64(endPos)), bits.OnesCount64(uint64(startPos)))
	}

	moves := maskPos.AllMoves()
	// Tri désactivé pour l'instant : sans élagage, il rend la résolution de la croix anglaise
	// beaucoup plus longue (> 100 s au lieu de ~0,5 s).
	// SortMoves(moves, endPos) // les moves partant le plus loin du centre de la cible d'abord

	g := Game{
		startPos:   startPos,
		maskPos:    maskPos,
		endPos:     endPos,
		validMoves: moves,
		solution:   make([]Position, 0, maxMoves),           // vide, capacité = nombre max de coups
		seed:       [2]uint64{rand.Uint64(), rand.Uint64()}, // graine renouvelée à chaque NewGame
	}
	g.rng = rand.New(rand.NewPCG(g.seed[0], g.seed[1]))

	// Test de parité en O(1) : évite de lancer une recherche exhaustive vouée à l'échec.
	if err := g.VerifyRuleOfThree(); err != nil {
		return Game{}, err
	}

	// Pagode centrée sur le barycentre de la cible (ou du mask si cible 0) : le départ doit déjà passer le seuil.
	if err := g.initPagoda(); err != nil {
		return Game{}, err
	}
	if !g.pag.ok(startPos) {
		return Game{}, fmt.Errorf("pagode : cible inatteignable depuis le départ (poids %d < seuil %d)",
			g.pag.value(startPos), g.pag.threshold)
	}
	return g, nil
}

// resetDead prépare la map des impasses pour une nouvelle recherche.
// Allocation unique : la map n'est créée qu'à la première recherche ; ensuite clear() la vide
// en conservant sa mémoire, donc les recherches suivantes ne réallouent rien.
func (g *Game) resetDead() {
	if g.dead == nil {
		g.dead = make(map[Position]struct{})
		return
	}
	clear(g.dead)
}

// markDead mémorise une impasse (position dont toutes les branches ont échoué), dans la limite de maxDeadPositions.
func (g *Game) markDead(p Position) {
	if len(g.dead) >= maxDeadPositions {
		clear(g.dead) // map pleine : on la vide pour repartir sur de nouvelles impasses (mémoire inchangée)
	}
	g.dead[p] = struct{}{}
}

// search lance le DFS donné (dfs ou dfsAnyEnd) avec redémarrages et budget de nœuds.
// 1re tentative : ordre fixe des coups, budget firstBudget (rapide quand l'ordre fixe convient, comme la croix anglaise).
// Ensuite : à chaque tentative, l'ordre des coups est mélangé (graine du Game) ; le budget croît lentement (+3 %) :
// beaucoup de redémarrages courts valent mieux qu'un budget qui double (queue lourde des temps de recherche).
// Les impasses mémorisées restent valables d'une tentative à l'autre (elles ne sont marquées qu'après une
// exploration complète). Une tentative qui se termine sans épuiser son budget et sans succès prouve
// qu'il n'y a pas de solution.
func (g *Game) search(run func(Position, int32) bool, anyEnd bool) error {
	g.solution = g.solution[:0]
	g.resetDead()
	if g.rng == nil { // Game construit sans NewGame
		g.rng = rand.New(rand.NewPCG(g.seed[0], g.seed[1]))
	}
	g.moves = g.validMoves
	g.guided = false
	budget := firstBudget
	attempts := 0
	for {
		g.nodes, g.budget, g.aborted = 0, budget, false
		if run(g.startPos, g.pag.value(g.startPos)) { // la pagode du départ est calculée une seule fois
			return nil
		}
		g.solution = g.solution[:0]
		if !g.aborted {
			return fmt.Errorf("aucune solution entre la position de départ et la position cible")
		}

		// La recherche est dure (beaucoup de tentatives échouées) : on construit la table de fins de partie.
		// Avec l'ordre guidé, les résolutions courantes n'en ont pas besoin, et sa construction coûte plus que le gain.
		attempts++
		if attempts >= endgameAfter && !g.endgameBuilt && !g.endgameOff {
			g.endgameBuilt = true
			g.buildEndgame(anyEnd)
		}

		// Tentative suivante : ordre des coups mélangé.
		if g.shuffled == nil {
			g.shuffled = slices.Clone(g.validMoves)
		}
		if budget == firstBudget {
			budget = initialBudget
		} else {
			budget += budget / 32
		}
		g.rng.Shuffle(len(g.shuffled), func(i, j int) {
			g.shuffled[i], g.shuffled[j] = g.shuffled[j], g.shuffled[i]
		})
		g.moves = g.shuffled
		g.guided = true // guidage des tentatives aléatoires
	}
}

// Solve recherche une solution pour aller de la position de départ à la position cible
// (depth-first avec mémorisation des impasses, élagage par pagode, ordre des coups aléatoire
// et redémarrages : voir search). Erreur si impossible.
// En cas de succès, g.solution contient les positions successives après chaque coup
// (la position de départ n'est pas incluse, la dernière est endPos).
func (g *Game) Solve() error {
	return g.search(g.dfs, false)
}

// SolveAnyEnd est une variante de Solve : si la position cible est 0, on cherche simplement
// à ne laisser qu'un seul pion, quelle que soit sa position. Sinon, même comportement que Solve
// (la cible exacte doit être atteinte). Erreur si impossible.
// En cas de succès, g.solution contient les positions successives après chaque coup.
func (g *Game) SolveAnyEnd() error {
	return g.search(g.dfsAnyEnd, true)
}

// dfsAnyEnd : comme dfs (pag est la valeur de pagode de curr, mise à jour avec Move.dPag), mais avec endPos == 0 le but est d'atteindre un seul pion.
func (g *Game) dfsAnyEnd(curr Position, pag int32) bool {

	if curr == g.endPos || (g.endPos == 0 && bits.OnesCount64(uint64(curr)) == 1) {
		return true
	}

	// Budget de nœuds épuisé : on abandonne cette tentative (rien n'est marqué comme impasse).
	g.nodes++
	if g.nodes > g.budget {
		g.aborted = true
		return false
	}

	// Table de fins de partie : une position à endgameK pions absente de la table est perdante (table exacte).
	if g.endgameK != 0 && bits.OnesCount64(uint64(curr)) == g.endgameK && !g.endgame.has(curr) {
		return false
	}

	// Pagode : la position ne peut plus atteindre la cible, inutile d'aller plus loin.
	if pag < g.pag.threshold {
		return false
	}

	// Impasse déjà rencontrée par un autre chemin : inutile de la réexplorer.
	if _, lost := g.dead[curr]; lost {
		return false
	}

	if g.guided {
		buf := g.scratchFor(len(g.solution))
		if n := g.sortedChildren(curr, buf); n >= 0 {
			for i := range n {
				c := &buf[i]
				g.solution = append(g.solution, c.next)
				if g.dfsAnyEnd(c.next, pag+c.dPag) {
					return true
				}
				g.solution = g.solution[:len(g.solution)-1] // backtrack
				if g.aborted {
					return false
				}
			}
			g.markDead(curr)
			return false
		}
	}

	for _, m := range g.moves {
		next := curr.DoMove(m)
		if next == 0 { // move illégal depuis cette position
			continue
		}
		g.solution = append(g.solution, next)
		if g.dfsAnyEnd(next, pag+m.dPag) {
			return true
		}
		g.solution = g.solution[:len(g.solution)-1] // backtrack
		if g.aborted {
			return false // budget épuisé plus bas : exploration incomplète, ne rien mémoriser
		}
	}

	g.markDead(curr) // toutes les branches ont échoué : on mémorise l'impasse
	return false
}

// Recursive depth-first exploration, with backtracking.
// Renvoie true dès que la cible est atteinte ; g.solution contient alors le chemin parcouru.
// Pas d'allocation : solution a été créée avec la capacité maximale (un pion retiré par coup).
func (g *Game) dfs(curr Position, pag int32) bool {

	if curr == g.endPos {
		return true
	}

	// Budget de nœuds épuisé : on abandonne cette tentative (rien n'est marqué comme impasse).
	g.nodes++
	if g.nodes > g.budget {
		g.aborted = true
		return false
	}

	// Table de fins de partie : une position à endgameK pions absente de la table est perdante (table exacte).
	if g.endgameK != 0 && bits.OnesCount64(uint64(curr)) == g.endgameK && !g.endgame.has(curr) {
		return false
	}

	// Pagode : la position ne peut plus atteindre la cible, inutile d'aller plus loin.
	if pag < g.pag.threshold {
		return false
	}

	// Impasse déjà rencontrée par un autre chemin : inutile de la réexplorer.
	if _, lost := g.dead[curr]; lost {
		return false
	}

	if g.guided {
		buf := g.scratchFor(len(g.solution))
		if n := g.sortedChildren(curr, buf); n >= 0 {
			for i := range n {
				c := &buf[i]
				g.solution = append(g.solution, c.next)
				if g.dfs(c.next, pag+c.dPag) {
					return true
				}
				g.solution = g.solution[:len(g.solution)-1] // backtrack
				if g.aborted {
					return false
				}
			}
			g.markDead(curr)
			return false
		}
	}

	for _, m := range g.moves {
		next := curr.DoMove(m)
		if next == 0 { // move illégal depuis cette position
			continue
		}
		g.solution = append(g.solution, next)
		if g.dfs(next, pag+m.dPag) {
			return true
		}
		g.solution = g.solution[:len(g.solution)-1] // backtrack
		if g.aborted {
			return false // budget épuisé plus bas : exploration incomplète, ne rien mémoriser
		}
	}

	g.markDead(curr) // toutes les branches ont échoué : on mémorise l'impasse
	return false
}

// VerifyRuleOfThree detecte avec des considérations de parité si la cible est certainement hors d'atteinte.
// Une réponse sans erreur ne garanti pas qu'une solution existe, mais une erreur garanti qu'elle n'existe pas.
/* Le test de classe modulo 3 (souvent appelé Rule of Three ou Coloring Invariant) est un invariant algébrique extrêmement puissant.
Là où les fonctions de pagode vérifient une inégalité (perte de potentiel), le test modulo 3 vérifie une égalité stricte de parité :
il permet de prouver en $O(1)$ qu'un état cible (ou un état intermédiaire) est mathématiquement inatteignable,
quel que soit le plateau (anglais, européen, rectangulaire, irrégulier).

1. Le principe mathématique : Les deux coloriages diagonauxOn attribue à chaque case $(x, y)$ de la grille
deux étiquettes dans $\{0, 1, 2\}$ selon les deux directions diagonales :Coloriage montant ($\nearrow$) : $c_1(x, y) = (x + y) \bmod 3$Coloriage descendant ($\searrow$) : $c_2(x, y) = (x - y) \bmod 3$ (en arithmétique positive : $((x - y) \bmod 3 + 3) \bmod 3$)PlaintextColoriage c1 = (x + y) mod 3 :
        2   0   1
    2   0   1   2   0
2   0   1   2   0   1   2
0   1   2   0   1   2   0
1   2   0   1   2   0   1
    0   1   2   0   1
        2   0   1

Pourquoi ce coloriage est invariant ?
Prenons n'importe quel saut orthogonal (horizontal ou vertical) de 3 cases consécutives $(A, B \rightarrow C)$.
Sur une ligne ou une colonne, la coordonnée qui change varie de $+1$ à chaque case.
Par conséquent, les 3 cases $(A, B, C)$ portent toujours les trois couleurs distinctes $\{0, 1, 2\}$ dans un ordre quelconque (ex: 0, 1, 2 ou 2, 1, 0).
Lorsqu'on joue le coup $(A, B \rightarrow C)$ :On retire une bille de la couleur de $A$ ($-1$).
On retire une bille de la couleur de $B$ ($-1$).On ajoute une bille de la couleur de $C$ ($+1$).
Si l'on note $N_0, N_1, N_2$ le nombre de billes présentes sur les cases de couleur 0, 1 et 2, à chaque coup :
Deux des compteurs diminuent de 1 (leur parité s'inverse).Le troisième compteur augmente de 1 (sa parité s'inverse également).
Propriété fondamentale : À chaque coup joué, la parité (pair/impair) des trois compteurs $(N_0 \bmod 2, N_1 \bmod 2, N_2 \bmod 2)$ s'inverse simultanément.

2. La signature invariante d'un plateau
Puisque les trois parités s'inversent ensemble à chaque coup, la différence de parité entre les couleurs ne change jamais au cours de toute la partie.
Pour un état du plateau $S$, on calcule le triplet de parités :$$P(S) = \big(N_0(S) \bmod 2,\; N_1(S) \bmod 2,\; N_2(S) \bmod 2\big)$$
Ce triplet ne peut prendre que deux valeurs possibles au cours d'une même partie :
Sa valeur initiale $P(S_{\text{init}})$ (après un nombre pair de coups).
Son complémentaire binaire $\overline{P(S_{\text{init}})} = (1 - p_0, 1 - p_1, 1 - p_2)$ (après un nombre impair de coups).
Condition nécessaire d'atteignabilité (Test de vraisemblance)Un état cible $S_{\text{cible}}$ ne peut être atteint depuis $S_{\text{init}}$ que
si les deux conditions suivantes sont simultanément vérifiées (pour $c_1$ et pour $c_2$) :
Cohérence de parité des classes :
Si le nombre de coups nécessaires $\Delta K = \text{Billes}(S_{\text{init}}) - \text{Billes}(S_{\text{cible}})$ est pair,
alors :$$P(S_{\text{cible}}) = P(S_{\text{init}})$$
Si $\Delta K$ est impair,
alors :$$P(S_{\text{cible}}) = \overline{P(S_{\text{init}})}$$
Double vérification ($c_1$ et $c_2$) :
Le test doit être validé indépendamment pour le coloriage $c_1 = (x+y) \bmod 3$ ET pour le coloriage $c_2 = (x-y) \bmod 3$.3.
*/
func (g Game) VerifyRuleOfThree() error {

	// Nombre de coups nécessaires pour aller du départ à la cible : chaque coup retire exactement un pion.
	startPegs := bits.OnesCount64(uint64(g.startPos))

	// Signature de parité du départ, calculée une seule fois pour toutes les cibles testées ci-dessous.
	sigStart := colorParities(g.startPos)

	// Cible vide (endPos == 0) : convention de SolveAnyEnd, on veut un seul pion, n'importe où.
	// La cible n'est alors pas une position précise : on teste chaque case du masque comme
	// emplacement possible du dernier pion. Si aucune ne passe le test, la partie est impossible.
	if g.endPos == 0 {
		if startPegs < 1 {
			return fmt.Errorf("règle de trois : aucune bille au départ, impossible d'en garder une seule")
		}
		deltaK := startPegs - 1
		for p := g.maskPos; p != 0; p &= p - 1 {
			lastPeg := Position(1) << bits.TrailingZeros64(uint64(p)) // une seule bille sur cette case
			if _, ok := ruleOfThreeMismatch(sigStart, lastPeg, deltaK); ok {
				return nil // au moins une case finale est compatible : pas de contradiction
			}
		}
		return fmt.Errorf("règle de trois : aucune case du plateau ne permet de finir avec une seule bille")
	}

	// Cible précise : le nombre de coups est fixé par la différence du nombre de billes.
	deltaK := startPegs - bits.OnesCount64(uint64(g.endPos))
	if deltaK < 0 {
		return fmt.Errorf("règle de trois : la cible a plus de billes (%d) que le départ (%d)",
			bits.OnesCount64(uint64(g.endPos)), startPegs)
	}

	// Compare les signatures de parité du départ et de la cible, pour les deux coloriages.
	if name, ok := ruleOfThreeMismatch(sigStart, g.endPos, deltaK); !ok {
		return fmt.Errorf("règle de trois : cible inatteignable (parités incompatibles pour le coloriage %s, après %d coups)",
			name, deltaK)
	}
	return nil
}

// colorParities calcule, pour une position, la signature de parité P(S) = (N0 mod 2, N1 mod 2, N2 mod 2)
// pour chacun des deux coloriages diagonaux :
//   - sig[0] pour c1 = (x+y) mod 3
//   - sig[1] pour c2 = (x-y) mod 3
//
// Ni N0, N1 ni N2 ne sont stockés : on ne garde que leur parité, obtenue par un XOR à chaque bille rencontrée.
func colorParities(p Position) (sig [2][3]uint8) {
	// On parcourt uniquement les bits à 1 (une itération par bille) :
	// p &= p-1 efface le bit de poids faible, TrailingZeros64 donne son index.
	for ; p != 0; p &= p - 1 {
		i := bits.TrailingZeros64(uint64(p))
		l, c := i/8, i%8 // ligne et colonne de la bille

		c1 := (l + c) % 3       // coloriage montant : toujours >= 0, un simple modulo suffit
		c2 := ((l-c)%3 + 3) % 3 // coloriage descendant : l-c peut être négatif, on ramène dans {0,1,2}

		sig[0][c1] ^= 1 // une bille de plus sur cette couleur : sa parité s'inverse
		sig[1][c2] ^= 1
	}
	return sig
}

// ruleOfThreeMismatch teste la condition nécessaire d'atteignabilité entre deux positions séparées
// de deltaK coups. Renvoie ok == true si les parités sont cohérentes pour les deux coloriages ;
// sinon ok == false et name désigne le premier coloriage en défaut.
// La signature de départ est fournie déjà calculée : elle est réutilisée pour tester plusieurs cibles.
func ruleOfThreeMismatch(sigStart [2][3]uint8, target Position, deltaK int) (name string, ok bool) {
	sigTarget := colorParities(target)

	// Chaque coup inverse simultanément les trois parités (deux couleurs perdent une bille,
	// la troisième en gagne une). Après deltaK coups, la signature de départ est donc :
	//   - inchangée si deltaK est pair,
	//   - complémentée (chaque parité inversée) si deltaK est impair.
	flip := uint8(deltaK & 1) // 0 si pair, 1 si impair : XOR avec flip applique cette règle

	names := [2]string{"c1 = (x+y) mod 3", "c2 = (x-y) mod 3"}
	for k := range sigStart { // k = 0 pour c1, k = 1 pour c2
		for color := range sigStart[k] { // les trois couleurs 0, 1, 2
			if sigStart[k][color]^flip != sigTarget[k][color] {
				return names[k], false // parité attendue différente de celle de la cible : inatteignable
			}
		}
	}
	return "", true // les deux coloriages sont cohérents : la cible reste possible
}

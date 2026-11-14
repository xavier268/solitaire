package solitaire

import (
	"fmt"
	"math/bits"
)

type Game struct {
	startPos   Position // Start position
	maskPos    Position // Board limit
	endPos     Position // Target end position
	validMoves []Move   // List of valid moves (sorted)
	solution   []Position

	// Positions déjà explorées sans succès (impasses), pour ne jamais les explorer deux fois.
	// La map est allouée une seule fois (à la première recherche), puis simplement vidée
	// par clear() à chaque nouvelle recherche : sa mémoire est conservée et réutilisée.
	dead map[Position]struct{}
}

// maxDeadPositions borne la mémoire de la map des impasses (environ 100 Mo au maximum).
// Au-delà, les nouvelles impasses ne sont plus mémorisées : la recherche reste correcte, mais plus lente.
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
		solution:   make([]Position, 0, maxMoves), // vide, capacité = nombre max de coups
	}

	// Test de parité en O(1) : évite de lancer une recherche exhaustive vouée à l'échec.
	if err := g.VerifyRuleOfThree(); err != nil {
		return Game{}, err
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
	if len(g.dead) < maxDeadPositions {
		g.dead[p] = struct{}{}
	}
}

// Solve recherche une solution pour aller de la position de départ à la position cible
// (depth-first, avec mémorisation des impasses, sans autre élagage). Erreur si impossible.
// En cas de succès, g.solution contient les positions successives après chaque coup
// (la position de départ n'est pas incluse, la dernière est endPos).
func (g *Game) Solve() error {
	g.solution = g.solution[:0]
	g.resetDead()
	if !g.dfs(g.startPos) {
		return fmt.Errorf("aucune solution entre la position de départ et la position cible")
	}
	return nil
}

// SolveAnyEnd est une variante de Solve : si la position cible est 0, on cherche simplement
// à ne laisser qu'un seul pion, quelle que soit sa position. Sinon, même comportement que Solve
// (la cible exacte doit être atteinte). Erreur si impossible.
// En cas de succès, g.solution contient les positions successives après chaque coup.
func (g *Game) SolveAnyEnd() error {
	g.solution = g.solution[:0]
	g.resetDead()
	if !g.dfsAnyEnd(g.startPos) {
		return fmt.Errorf("aucune solution entre la position de départ et la position cible")
	}
	return nil
}

// dfsAnyEnd : comme dfs, mais avec endPos == 0 le but est d'atteindre un seul pion.
func (g *Game) dfsAnyEnd(curr Position) bool {

	if curr == g.endPos || (g.endPos == 0 && bits.OnesCount64(uint64(curr)) == 1) {
		return true
	}

	// Impasse déjà rencontrée par un autre chemin : inutile de la réexplorer.
	if _, lost := g.dead[curr]; lost {
		return false
	}

	for _, m := range g.validMoves {
		next := curr.DoMove(m)
		if next == 0 { // move illégal depuis cette position
			continue
		}
		g.solution = append(g.solution, next)
		if g.dfsAnyEnd(next) {
			return true
		}
		g.solution = g.solution[:len(g.solution)-1] // backtrack
	}

	g.markDead(curr) // toutes les branches ont échoué : on mémorise l'impasse
	return false
}

// Recursive depth-first exploration, with backtracking.
// Renvoie true dès que la cible est atteinte ; g.solution contient alors le chemin parcouru.
// Pas d'allocation : solution a été créée avec la capacité maximale (un pion retiré par coup).
func (g *Game) dfs(curr Position) bool {

	if curr == g.endPos {
		return true
	}

	// Impasse déjà rencontrée par un autre chemin : inutile de la réexplorer.
	if _, lost := g.dead[curr]; lost {
		return false
	}

	for _, m := range g.validMoves {
		next := curr.DoMove(m)
		if next == 0 { // move illégal depuis cette position
			continue
		}
		g.solution = append(g.solution, next)
		if g.dfs(next) {
			return true
		}
		g.solution = g.solution[:len(g.solution)-1] // backtrack
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

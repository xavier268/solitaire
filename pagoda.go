package solitaire

import (
	"fmt"
	"math/bits"
)

// pagoda est une fonction de pagode entière : un poids par case du board, tel que pour tout coup
// (A saute B vers C) on ait w(A) + w(B) >= w(C). Un coup retire les pions A et B et ajoute C :
// la somme des poids des pions ne peut donc que diminuer (ou rester égale) au fil de la partie.
// Si la somme d'une position est inférieure au seuil, la cible est certainement inatteignable.
type pagoda struct {
	weight    [64]int32 // poids de chaque case (0 hors du mask), indexé comme BitIndex
	threshold int32     // somme minimale à avoir pour que la cible reste atteignable
}

// value renvoie la somme des poids des pions de la position p (une itération par pion).
func (pg *pagoda) value(p Position) (sum int32) {
	for ; p != 0; p &= p - 1 {
		sum += pg.weight[bits.TrailingZeros64(uint64(p))]
	}
	return sum
}

// ok renvoie false si la position p ne peut certainement pas atteindre la cible (somme sous le seuil).
func (pg *pagoda) ok(p Position) bool {
	return pg.value(p) >= pg.threshold
}

// barycenter renvoie la case (ligne, colonne) la plus proche du barycentre des cases à 1 de p.
// p doit être non nul. Calcul en entiers : arrondi de sum/n par (2*sum+n)/(2*n).
func barycenter(p Position) (l, c int) {
	n := bits.OnesCount64(uint64(p))
	var sumL, sumC int
	for q := p; q != 0; q &= q - 1 {
		i := bits.TrailingZeros64(uint64(q))
		sumL += i / 8
		sumC += i % 8
	}
	return (2*sumL + n) / (2 * n), (2*sumC + n) / (2 * n)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// initPagoda construit la fonction de pagode du jeu et son seuil. À appeler une fois, depuis NewGame,
// quand maskPos, endPos et validMoves sont renseignés.
//
// Poids : Fibonacci de la distance de Manhattan au centre, w = Fib(D - d), avec d = |l-cl| + |c-cc|
// et D = (distance maximale d'une case du mask) + 1, de sorte que tous les poids soient >= 1.
// Le long d'un alignement de 3 cases, la distance varie de 1 en 1 (ou fait un aller-retour en traversant
// l'axe du centre), et Fib(n) + Fib(n+1) = Fib(n+2) garantit la condition de pagode.
//
// Le centre est la case la plus proche du barycentre de endPos, ou du mask si endPos == 0.
// Seuil : somme des poids de endPos ; si endPos == 0 (une seule bille, n'importe où),
// le plus petit poids d'une case du mask.
//
// Par sécurité, la condition de pagode est vérifiée sur tous les coups du jeu (et leur dPag est renseigné) : une erreur est renvoyée
// si elle est violée (formule de poids fausse), plutôt que d'élaguer à tort.
func (g *Game) initPagoda() error {
	ref := g.endPos
	if ref == 0 {
		ref = g.maskPos
	}
	if ref == 0 {
		return fmt.Errorf("pagode : mask vide")
	}
	cl, cc := barycenter(ref)

	dist := func(i int) int { return abs(i/8-cl) + abs(i%8-cc) }

	maxD := 0
	for p := g.maskPos; p != 0; p &= p - 1 {
		maxD = max(maxD, dist(bits.TrailingZeros64(uint64(p))))
	}
	D := maxD + 1

	// Table de Fibonacci : fib[0] = 0, fib[1] = 1, ... jusqu'à D (la distance 0 donne Fib(D)).
	fib := make([]int32, D+1)
	if D >= 1 {
		fib[1] = 1
	}
	for n := 2; n <= D; n++ {
		fib[n] = fib[n-1] + fib[n-2]
	}

	g.pag = pagoda{}
	minW := int32(0)
	for p := g.maskPos; p != 0; p &= p - 1 {
		i := bits.TrailingZeros64(uint64(p))
		w := fib[D-dist(i)]
		g.pag.weight[i] = w
		if minW == 0 || w < minW {
			minW = w
		}
	}

	if g.endPos == 0 {
		g.pag.threshold = minW
	} else {
		g.pag.threshold = g.pag.value(g.endPos)
	}

	// Variation de pagode de chaque coup, calculée une fois pour toutes et stockée dans le Move :
	// dPag = w(to) - w(from) - w(saut). La condition de pagode w(from) + w(saut) >= w(to) équivaut à dPag <= 0 ;
	// elle est contrôlée sur tous les coups. La recherche met alors la pagode à jour en O(1) par coup.
	for i := range g.validMoves {
		m := &g.validMoves[i]
		from, to := int(BitIndex(m.from)), int(BitIndex(m.to))
		mid := (from + to) / 2 // la case sautée est au milieu (même ligne ou même colonne)
		m.dPag = g.pag.weight[to] - g.pag.weight[from] - g.pag.weight[mid]
		if m.dPag > 0 {
			return fmt.Errorf("pagode : condition violée pour le coup %s", *m)
		}
	}
	return nil
}

package solitaire

import (
	"math/bits"
	"math/rand/v2"
	"testing"
)

// Références lentes, case par case.
func refIsolated(p Position) (n int) {
	has := func(l, c int) bool { return l >= 0 && l < 8 && c >= 0 && c < 8 && p&Bit(uint8(l), uint8(c)) != 0 }
	for i := 0; i < 64; i++ {
		l, c := i/8, i%8
		if has(l, c) && !has(l-1, c) && !has(l+1, c) && !has(l, c-1) && !has(l, c+1) {
			n++
		}
	}
	return n
}

func refMobility(p, mask Position) (n int) {
	for _, m := range mask.AllMoves() {
		if p.DoMove(m) != 0 {
			n++
		}
	}
	return n
}

// isolated et mobility (opérations sur les bits) égalent leurs références, y compris sur une grille 8x8 complète
// (les colonnes 0 et 7 ne doivent pas se contaminer d'une ligne à l'autre).
func TestGuideFeatures(t *testing.T) {
	rng := rand.New(rand.NewPCG(21, 22))
	for name, mask := range map[string]Position{"8x8": MASK_8x8, "européen": MASK_EUROPEAN, "croix": MASK_ENGLISHCROSS, "5x5": MASK_5x5} {
		for i := 0; i < 2000; i++ {
			p := Position(rng.Uint64()&rng.Uint64()) & mask // positions plus ou moins clairsemées
			if got, want := isolated(p), refIsolated(p); got != want {
				t.Fatalf("%s : isolated(%#x) = %d, référence %d", name, uint64(p), got, want)
			}
			if got, want := mobility(p, mask), refMobility(p, mask); got != want {
				t.Fatalf("%s : mobility(%#x) = %d, référence %d", name, uint64(p), got, want)
			}
		}
	}
}

// Le tri des enfants est décroissant, stable, et contient exactement les coups légaux.
func TestSortedChildren(t *testing.T) {
	g, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
	if err != nil {
		t.Fatal(err)
	}
	g.moves = g.validMoves
	rng := rand.New(rand.NewPCG(23, 24))
	p := g.startPos
	buf := make([]cand, maxGuided)
	for step := 0; step < 30; step++ {
		n := g.sortedChildren(p, buf)
		legal := 0
		for _, m := range g.validMoves {
			if p.DoMove(m) != 0 {
				legal++
			}
		}
		if n != legal {
			t.Fatalf("%d enfants triés, %d coups légaux", n, legal)
		}
		for i := 1; i < n; i++ {
			if buf[i-1].score < buf[i].score {
				t.Fatal("tri non décroissant")
			}
		}
		if n == 0 {
			break
		}
		p = buf[rng.IntN(n)].next
	}
}

// Une recherche guidée (tentatives aléatoires) donne une suite de coups légaux jusqu'à une seule bille.
func TestSolveEuropeanGuided(t *testing.T) {
	for seed := uint64(0); seed < 5; seed++ {
		g, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
		if err != nil {
			t.Fatal(err)
		}
		g.rng = rand.New(rand.NewPCG(seed, 1))
		if err := g.SolveAnyEnd(); err != nil {
			t.Fatal(err)
		}
		prev := g.startPos
		for i, pos := range g.solution {
			ok := false
			for _, m := range g.validMoves {
				if prev.DoMove(m) == pos {
					ok = true
					break
				}
			}
			if !ok {
				t.Fatalf("graine %d, étape %d : coup illégal", seed, i)
			}
			prev = pos
		}
		if bits.OnesCount64(uint64(prev)) != 1 || len(g.solution) != 35 {
			t.Fatalf("graine %d : %d billes, %d coups", seed, bits.OnesCount64(uint64(prev)), len(g.solution))
		}
	}
}

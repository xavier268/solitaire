package solitaire

import (
	"math/bits"
	"testing"
)

func TestDoMoveLegal(t *testing.T) {
	// Saut horizontal : pion en (0,0), pion sauté en (0,1), arrivée libre en (0,2).
	m := Move{
		from:  Point{0, 0},
		to:    Point{0, 2},
		mask3: Bit(0, 0) | Bit(0, 1) | Bit(0, 2),
		mask2: Bit(0, 0) | Bit(0, 1),
	}
	p := Bit(0, 0) | Bit(0, 1) | Bit(4, 4) // un pion étranger doit rester intact

	got := p.DoMove(m)
	want := Bit(0, 2) | Bit(4, 4)
	t.Logf("avant :\n%s\naprès :\n%s", p.Render(MASK_5x5), got.Render(MASK_5x5))
	if got != want {
		t.Errorf("DoMove = %#x, attendu %#x", uint64(got), uint64(want))
	}
}

func TestDoMoveIllegal(t *testing.T) {
	m := Move{
		from:  Point{2, 2},
		to:    Point{2, 4},
		mask3: Bit(2, 2) | Bit(2, 3) | Bit(2, 4),
		mask2: Bit(2, 2) | Bit(2, 3),
	}
	tests := []struct {
		name string
		p    Position
	}{
		{"position vide", 0},
		{"départ vide", Bit(2, 3)},
		{"case sautée vide", Bit(2, 2)},
		{"arrivée occupée", Bit(2, 2) | Bit(2, 3) | Bit(2, 4)},
		{"seule l'arrivée occupée", Bit(2, 4)},
		{"mauvaise paire occupée", Bit(2, 3) | Bit(2, 4)},
	}
	for _, tc := range tests {
		if got := tc.p.DoMove(m); got != 0 {
			t.Errorf("%s : DoMove = %#x, attendu 0 (illégal)", tc.name, uint64(got))
		}
	}
}

func TestAllMovesCounts(t *testing.T) {
	tests := []struct {
		name string
		mask Position
		want int
	}{
		{"5x5", MASK_5x5, 60},                     // 30 triplets alignés x 2 sens
		{"croix anglaise", MASK_ENGLISHCROSS, 76}, // 38 triplets alignés x 2 sens
		{"8x8", MASK_8x8, 192},                    // 96 triplets alignés x 2 sens
		{"vide", 0, 0},
		{"une seule case", Bit(3, 3), 0},
		{"deux cases", Bit(3, 3) | Bit(3, 4), 0},
		{"trois alignées", Bit(3, 3) | Bit(3, 4) | Bit(3, 5), 2},
		{"trois en L", Bit(3, 3) | Bit(3, 4) | Bit(4, 4), 0},
	}
	for _, tc := range tests {
		if got := len(tc.mask.AllMoves()); got != tc.want {
			t.Errorf("%s : %d moves, attendu %d", tc.name, got, tc.want)
		}
	}
}

func TestAllMovesWellFormed(t *testing.T) {
	for _, mask := range []Position{MASK_5x5, MASK_ENGLISHCROSS, MASK_8x8} {
		seen := map[Move]bool{}
		for _, m := range mask.AllMoves() {
			if seen[m] {
				t.Errorf("move en double : %v -> %v", m.from, m.to)
			}
			seen[m] = true

			if m.mask3&^mask != 0 {
				t.Errorf("move %v -> %v sort du mask", m.from, m.to)
			}
			if bits.OnesCount64(uint64(m.mask3)) != 3 {
				t.Errorf("move %v -> %v : mask3 doit avoir 3 bits", m.from, m.to)
			}
			if bits.OnesCount64(uint64(m.mask2)) != 2 || m.mask2&^m.mask3 != 0 {
				t.Errorf("move %v -> %v : mask2 doit avoir 2 bits dans mask3", m.from, m.to)
			}
			if m.mask2&m.to.Bit() != 0 {
				t.Errorf("move %v -> %v : l'arrivée ne doit pas être dans mask2", m.from, m.to)
			}
			if m.mask3 != m.mask2|m.to.Bit() || m.mask2&m.from.Bit() == 0 {
				t.Errorf("move %v -> %v : masks incohérents avec from/to", m.from, m.to)
			}

			// from et to alignés, à distance 2.
			dl := int(m.to.l) - int(m.from.l)
			dc := int(m.to.c) - int(m.from.c)
			if !((dl == 0 && (dc == 2 || dc == -2)) || (dc == 0 && (dl == 2 || dl == -2))) {
				t.Errorf("move %v -> %v : pas un saut de 2 cases aligné", m.from, m.to)
			}
		}
	}
}

// Le sens inverse d'un move légal doit exister dans AllMoves.
func TestAllMovesSymmetric(t *testing.T) {
	moves := MASK_ENGLISHCROSS.AllMoves()
	set := map[[2]Point]bool{}
	for _, m := range moves {
		set[[2]Point{m.from, m.to}] = true
	}
	for _, m := range moves {
		if !set[[2]Point{m.to, m.from}] {
			t.Errorf("le move inverse de %v -> %v est absent", m.from, m.to)
		}
	}
}

// Joue une partie complète (premier coup légal à chaque fois) et vérifie les invariants :
// un pion en moins à chaque coup, jamais de pion hors du mask.
func TestPlayGame(t *testing.T) {
	mask := MASK_ENGLISHCROSS
	moves := mask.AllMoves()
	p := mask &^ Bit(3, 3)
	count := bits.OnesCount64(uint64(p))

	for {
		played := false
		for _, m := range moves {
			pp := p.DoMove(m)
			if pp == 0 {
				continue
			}
			if n := bits.OnesCount64(uint64(pp)); n != count-1 {
				t.Fatalf("après %v -> %v : %d pions, attendu %d", m.from, m.to, n, count-1)
			}
			if pp&^mask != 0 {
				t.Fatalf("après %v -> %v : pion hors du mask", m.from, m.to)
			}
			p, count, played = pp, count-1, true
			break
		}
		if !played {
			break
		}
	}
	t.Logf("Partie terminée avec %d pions restants :\n%s", count, p.Render(mask))
	if count < 1 {
		t.Error("il doit rester au moins un pion")
	}
}

func TestMoveString(t *testing.T) {
	m := Move{from: Point{1, 3}, to: Point{3, 3}}
	if got, want := m.String(), "d2->d4"; got != want {
		t.Errorf("Move.String() = %q, attendu %q", got, want)
	}
}

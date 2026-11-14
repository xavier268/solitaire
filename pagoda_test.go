package solitaire

import "testing"

// La condition de pagode est contrôlée par initPagoda sur tous les coups : NewGame réussit donc
// uniquement si elle est vérifiée. On la revérifie ici explicitement, indépendamment d'initPagoda.
func TestPagodaCondition(t *testing.T) {
	cases := []struct {
		name  string
		mask  Position
		start Position
		end   Position
	}{
		{"5x5", MASK_5x5, START_5x5, 0},
		{"croix anglaise cible centre", MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS},
		{"croix anglaise cible 0", MASK_ENGLISHCROSS, START_ENGLISHCROSS, 0},
		{"européen cible 0", MASK_EUROPEAN, START_EUROPEAN, 0},
	}
	for _, tc := range cases {
		g, err := NewGame(tc.mask, tc.start, tc.end)
		if err != nil {
			t.Logf("%s : NewGame refuse la configuration (%v), contrôle de la pagode sur un jeu minimal", tc.name, err)
			g = Game{maskPos: tc.mask, endPos: tc.end, startPos: tc.start, validMoves: tc.mask.AllMoves()}
		}
		if err := g.initPagoda(); err != nil {
			t.Errorf("%s : %v", tc.name, err)
			continue
		}
		for _, m := range g.validMoves {
			from, to := int(BitIndex(m.from)), int(BitIndex(m.to))
			mid := (from + to) / 2
			if g.pag.weight[from]+g.pag.weight[mid] < g.pag.weight[to] {
				t.Errorf("%s : condition violée pour %s", tc.name, m)
			}
		}
	}
}

func TestPagodaCenter(t *testing.T) {
	// Cible 0 : barycentre du mask. Croix anglaise et européen : case d4 (ligne 3, colonne 3).
	for _, mask := range []Position{MASK_ENGLISHCROSS, MASK_EUROPEAN} {
		if l, c := barycenter(mask); l != 3 || c != 3 {
			t.Errorf("barycentre du mask %#x = (%d,%d), attendu (3,3)", uint64(mask), l, c)
		}
	}
	// Cible d'une seule bille : le centre est cette case.
	target := Bit(4, 2)
	if l, c := barycenter(target); l != 4 || c != 2 {
		t.Errorf("barycentre d'une bille en (4,2) = (%d,%d)", l, c)
	}
	// Le poids est maximal au centre de la cible.
	g, err := NewGame(MASK_ENGLISHCROSS, START_ENGLISHCROSS, target)
	if err != nil {
		t.Skipf("configuration refusée : %v", err)
	}
	centre := g.pag.weight[BitIndex(Point{4, 2})]
	for i, w := range g.pag.weight {
		if w > centre {
			t.Errorf("poids de la case %d (%d) supérieur à celui du centre (%d)", i, w, centre)
		}
	}
}

func TestPagodaRejectsStart(t *testing.T) {
	// Départ : une seule bille dans un coin du 5x5 ; cible : deux billes proches du centre.
	// Moins de billes au départ qu'à l'arrivée est déjà refusé ; on teste donc un départ de 2 billes
	// éloignées du centre, avec une cible d'une bille au centre.
	start := Bit(0, 0) | Bit(0, 1)
	if _, err := NewGame(MASK_5x5, start, Bit(2, 2)); err == nil {
		t.Error("attendu une erreur : le départ pèse moins que la cible")
	}
}

func TestPagodaAcceptsKnownGames(t *testing.T) {
	if _, err := NewGame(MASK_ENGLISHCROSS, START_ENGLISHCROSS, CENTER_ENGLISHCROSS); err != nil {
		t.Errorf("croix anglaise : %v", err)
	}
	if _, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0); err != nil {
		t.Errorf("européen : %v", err)
	}
}

// La valeur de pagode mise à jour par Move.dPag doit égaler la valeur recalculée depuis la position.
func TestPagodaDelta(t *testing.T) {
	g, err := NewGame(MASK_EUROPEAN, START_EUROPEAN, 0)
	if err != nil {
		t.Fatal(err)
	}
	curr, val := g.startPos, g.pag.value(g.startPos)
	for step := 0; step < 35; step++ {
		moved := false
		for _, m := range g.validMoves {
			if next := curr.DoMove(m); next != 0 {
				curr, val, moved = next, val+m.dPag, true
				break
			}
		}
		if !moved {
			break
		}
		if want := g.pag.value(curr); val != want {
			t.Fatalf("étape %d : pagode incrémentale %d, recalculée %d", step, val, want)
		}
	}
}

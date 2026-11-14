package solitaire

import (
	"math/bits"
	"testing"
)

func TestBitIndex(t *testing.T) {
	tests := []struct{ ligne, colonne, want uint8 }{
		{0, 0, 0}, {0, 7, 7}, {1, 0, 8}, {2, 3, 19}, {7, 7, 63},
	}
	for _, tc := range tests {
		if got := BitIndex(Point{tc.ligne, tc.colonne}); got != tc.want {
			t.Errorf("BitIndex(%d,%d) = %d, attendu %d", tc.ligne, tc.colonne, got, tc.want)
		}
	}
}

func TestMask8x8(t *testing.T) {
	t.Logf("MASK_8x8 :\n%s", MASK_8x8.Render(MASK_8x8))
	if n := bits.OnesCount64(uint64(MASK_8x8)); n != 64 {
		t.Errorf("nombre de bits = %d, attendu 64", n)
	}
}

func TestMask5x5(t *testing.T) {
	m := MASK_5x5
	t.Logf("MASK_5x5 :\n%s", m.Render(m))

	if n := bits.OnesCount64(uint64(m)); n != 25 {
		t.Errorf("nombre de bits = %d, attendu 25", n)
	}
	for l := uint8(0); l < 8; l++ {
		for c := uint8(0); c < 8; c++ {
			want := l < 5 && c < 5
			if got := m&(1<<BitIndex(Point{l, c})) != 0; got != want {
				t.Errorf("bit (%d,%d) = %v, attendu %v", l, c, got, want)
			}
		}
	}
}

func TestMaskEnglishCross(t *testing.T) {
	m := MASK_ENGLISHCROSS
	t.Logf("MASK_ENGLISHCROSS :\n%s", m.Render(m))

	if n := bits.OnesCount64(uint64(m)); n != 33 {
		t.Errorf("nombre de bits = %d, attendu 33", n)
	}

	want := []string{
		"..OOO...",
		"..OOO...",
		"OOOOOOO.",
		"OOOOOOO.",
		"OOOOOOO.",
		"..OOO...",
		"..OOO...",
		"........",
	}
	for l, row := range want {
		for c, ch := range row {
			if got := m&(1<<BitIndex(Point{uint8(l), uint8(c)})) != 0; got != (ch == 'O') {
				t.Errorf("bit (%d,%d) = %v, attendu %v", l, c, got, ch == 'O')
			}
		}
	}
}

// TestRender vérifie que l'affichage tient compte du mask :
// hors mask = espace, dans le mask = "." (vide) ou "O" (occupé).
// Visible avec : go test -v -run TestRender
func TestRender(t *testing.T) {
	// Croix anglaise complète sauf le centre (3,3) : position initiale classique.
	start := MASK_ENGLISHCROSS &^ (1 << BitIndex(Point{3, 3}))
	t.Logf("Position initiale du solitaire anglais :\n%s", start.Render(MASK_ENGLISHCROSS))
	start.Print(MASK_ENGLISHCROSS)

	// Un bit hors du mask doit être ignoré.
	outside := Position(1) << BitIndex(Point{7, 7})
	if outside.Render(MASK_ENGLISHCROSS) != Position(0).Render(MASK_ENGLISHCROSS) {
		t.Error("un bit hors du mask ne doit pas être affiché")
	}
}

func TestPointString(t *testing.T) {
	tests := []struct {
		pt   Point
		want string
	}{
		{Point{0, 0}, "a1"}, {Point{3, 3}, "d4"}, {Point{2, 7}, "h3"}, {Point{7, 0}, "a8"},
	}
	for _, tc := range tests {
		if got := tc.pt.String(); got != tc.want {
			t.Errorf("Point%v.String() = %q, attendu %q", [2]uint8{tc.pt.l, tc.pt.c}, got, tc.want)
		}
	}
}

func TestRenderFormat(t *testing.T) {
	got := (Position(1) << BitIndex(Point{1, 2})).Render(MASK_5x5)
	want := "  a b c d e f g h\n" +
		"1 . . . . .       \n" +
		"2 . . O . .       \n" +
		"3 . . . . .       \n" +
		"4 . . . . .       \n" +
		"5 . . . . .       \n" +
		"6                 \n" +
		"7                 \n" +
		"8                 \n"
	if got != want {
		t.Errorf("Render =\n%s\nattendu\n%s", got, want)
	}
}

func TestStartPositions(t *testing.T) {
	tests := []struct {
		name  string
		start Position
		mask  Position
		hole  Point
		pegs  int
	}{
		{"5x5", START_5x5, MASK_5x5, Point{2, 2}, 24},
		{"croix anglaise", START_ENGLISHCROSS, MASK_ENGLISHCROSS, Point{3, 3}, 32},
	}
	for _, tc := range tests {
		t.Logf("START %s :\n%s", tc.name, tc.start.Render(tc.mask))
		if tc.start&^tc.mask != 0 {
			t.Errorf("%s : pion hors du mask", tc.name)
		}
		if n := bits.OnesCount64(uint64(tc.start)); n != tc.pegs {
			t.Errorf("%s : %d pions, attendu %d", tc.name, n, tc.pegs)
		}
		if tc.start&tc.hole.Bit() != 0 {
			t.Errorf("%s : le trou %v doit être vide", tc.name, tc.hole)
		}
		if tc.start|tc.hole.Bit() != tc.mask {
			t.Errorf("%s : doit être le mask moins le trou %v", tc.name, tc.hole)
		}
	}
}

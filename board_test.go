package solitaire

import (
	"math/bits"
	"testing"
)

func TestBitIndex(t *testing.T) {
	tests := []struct{ ligne, colonne, want int }{
		{0, 0, 0}, {0, 7, 7}, {1, 0, 8}, {2, 3, 19}, {7, 7, 63},
	}
	for _, tc := range tests {
		if got := BitIndex(tc.ligne, tc.colonne); got != tc.want {
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
	for l := 0; l < 8; l++ {
		for c := 0; c < 8; c++ {
			want := l < 5 && c < 5
			if got := m&(1<<BitIndex(l, c)) != 0; got != want {
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
			if got := m&(1<<BitIndex(l, c)) != 0; got != (ch == 'O') {
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
	start := MASK_ENGLISHCROSS &^ (1 << BitIndex(3, 3))
	t.Logf("Position initiale du solitaire anglais :\n%s", start.Render(MASK_ENGLISHCROSS))
	start.Print(MASK_ENGLISHCROSS)

	// Un bit hors du mask doit être ignoré.
	outside := Position(1) << BitIndex(7, 7)
	if outside.Render(MASK_ENGLISHCROSS) != Position(0).Render(MASK_ENGLISHCROSS) {
		t.Error("un bit hors du mask ne doit pas être affiché")
	}
}

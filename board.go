package solitaire

import (
	"fmt"
	"math/bits"
)

// Une position sur le board (8x8 max) est reprensté par un bitmap sur 64bits (uint64)
type Position uint64

// BitIndex renvoie l'index du bit (0 à 63) correspondant aux coordonnées (x, y).
// x est la colonne (0..7), y est la ligne (0..7).
func BitIndex(x, y int) int {
	return y*8 + x
}

// Square5x5Mask renvoie un masque avec un carré de 5x5 bits à 1.
// On le place par défaut en haut à gauche (x: 0..4, y: 0..4).
func Square5x5Mask() Position {
	var mask Position
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			mask |= 1 << BitIndex(x, y)
		}
	}
	return mask
}

// EnglishCrossMask renvoie le masque de la croix du solitaire anglais (33 bits).
// Le plateau fait 7x7, centré dans la grille 8x8 (coordonnées 0..6, la colonne 7 et la ligne 7 restent à 0).
func EnglishCrossMask() Position {
	var mask Position
	for y := 0; y < 7; y++ {
		for x := 0; x < 7; x++ {
			// On exclut les 4 coins 2x2 du carré 7x7 :
			// Haut-gauche (x<2 && y<2), Haut-droite (x>4 && y<2),
			// Bas-gauche (x<2 && y>4), Bas-droite (x>4 && y>4)
			inHorizontalBar := y >= 2 && y <= 4
			inVerticalBar := x >= 2 && x <= 4

			if inHorizontalBar || inVerticalBar {
				mask |= 1 << BitIndex(x, y)
			}
		}
	}
	return mask
}

func (p Position) Print() {
	fmt.Printf("Bits actifs : %d\n", bits.OnesCount64(uint64(p)))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if (p & (1 << BitIndex(x, y))) != 0 {
				fmt.Print("1 ")
			} else {
				fmt.Print(". ")
			}
		}
		fmt.Println()
	}
	fmt.Println()
}

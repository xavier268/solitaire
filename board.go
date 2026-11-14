package solitaire

import (
	"fmt"
	"math/bits"
	"strings"
)

// Une position sur le board (8x8 max) est reprensté par un bitmap sur 64bits (uint64)
type Position uint64

// BitIndex renvoie l'index du bit (0 à 63) correspondant à (ligne, colonne).
// ligne et colonne vont de 0 à 7.
func BitIndex(ligne, colonne int) int {
	return ligne*8 + colonne
}

// Masques définissant les limites des boards (1 octet par ligne, bit 0 = colonne 0).
const (
	// MASK_8x8 : toute la grille 8x8 (64 bits).
	MASK_8x8 Position = 0xFFFFFFFFFFFFFFFF

	// MASK_5x5 : carré de 5x5 en haut à gauche (x: 0..4, y: 0..4), 25 bits.
	MASK_5x5 Position = 0x0000001F1F1F1F1F

	// MASK_ENGLISHCROSS : croix du solitaire anglais, 33 bits.
	// Carré 7x7 (x,y: 0..6) privé de ses 4 coins 2x2.
	MASK_ENGLISHCROSS Position = 0x001C1C7F7F7F1C1C
)

// Affiche une position. Utilise le mask fourni pour délimiter la surface jouable.
func (p Position) Print(mask Position) {
	p &= mask
	fmt.Printf("Bits actifs : %d\n", bits.OnesCount64(uint64(p)))
	fmt.Println(p.Render(mask))
}

// Render dessine une Position en ASCII ("O" = bit actif, "." = bit inactif, " " = hors mask),
// avec les indices de colonnes et de lignes.
func (p Position) Render(mask Position) string {
	p &= mask
	var sb strings.Builder
	sb.WriteString("  0 1 2 3 4 5 6 7\n")
	for l := 0; l < 8; l++ {
		sb.WriteByte(byte('0' + l))
		sb.WriteByte(' ')
		for c := 0; c < 8; c++ {
			bit := Position(1) << BitIndex(l, c)
			switch {
			case mask&bit == 0:
				sb.WriteString("  ")
			case p&bit != 0:
				sb.WriteString("O ")
			default:
				sb.WriteString(". ")
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

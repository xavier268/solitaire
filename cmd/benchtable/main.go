// Compare les fichiers bench/bench-v*.txt avec benchstat et produit un tableau ASCII pur
// (tableaux Markdown : lisibles en texte brut et rendus correctement par une prévisualisation).
// Nécessite benchstat dans le PATH. Usage : go run ./cmd/benchtable [fichiers...]
package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	files := os.Args[1:]
	if len(files) == 0 {
		files, _ = filepath.Glob("bench/bench-v*.txt")
	}
	if len(files) < 1 {
		fmt.Fprintln(os.Stderr, "aucun fichier bench/bench-v*.txt")
		os.Exit(1)
	}

	cmd := exec.Command("benchstat", append([]string{"-format", "csv"}, files...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "benchstat : %v\n%s", err, stderr.String())
		os.Exit(1)
	}

	fmt.Println("# Comparaison des benchmarks (benchstat)")
	fmt.Println()

	for i, chunk := range strings.Split(strings.ReplaceAll(stdout.String(), "\r\n", "\n"), "\n\n") {
		chunk = strings.Trim(chunk, "\n")
		if chunk == "" {
			continue
		}
		var table []string
		for _, line := range strings.Split(chunk, "\n") {
			if strings.HasPrefix(line, ",") || len(table) > 0 {
				table = append(table, line)
			} else if i == 0 { // ligne d'en-tête goos:, goarch:, pkg:, cpu:
				fmt.Printf("- %s\n", line)
			}
		}
		if i == 0 {
			fmt.Println()
		}
		printTable(strings.Join(table, "\n"))
	}

	// Avertissements de benchstat (stderr), dédoublonnés et sans la référence de cellule.
	cell := regexp.MustCompile(`^[A-Z]+\d+: `)
	seen := map[string]bool{}
	var notes []string
	for _, line := range strings.Split(stderr.String(), "\n") {
		if msg := cell.ReplaceAllString(strings.TrimSpace(line), ""); msg != "" && !seen[msg] {
			seen[msg] = true
			notes = append(notes, msg)
		}
	}
	if len(notes) > 0 {
		fmt.Println("Notes :")
		fmt.Println()
		for _, n := range notes {
			fmt.Printf("- %s\n", n)
		}
	}
}

// printTable met en forme un bloc CSV de benchstat en tableau Markdown aligné.
func printTable(block string) {
	r := csv.NewReader(strings.NewReader(block))
	// Les lignes n'ont pas toutes le même nombre de champs : benchstat omet les colonnes
	// vides en fin de ligne (benchmark absent des premières versions, par exemple).
	r.FieldsPerRecord = -1
	recs, err := r.ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchtable : CSV illisible : %v\n", err)
		os.Exit(1)
	}
	if len(recs) < 3 {
		return
	}
	h1, h2, rows := recs[0], recs[1], recs[2:]
	unit := h2[1]

	// Colonnes : (valeur, CI) fusionnés, (vs base, P) fusionnés.
	type col struct {
		title string
		idx   int
		kind  string // "value", "delta", ""
	}
	var cols []col
	for i := 1; i < len(h2); i++ {
		switch {
		case i+1 < len(h2) && h2[i+1] == "CI":
			cols = append(cols, col{path.Base(strings.ReplaceAll(h1[i], `\`, "/")), i, "value"})
			i++
		case h2[i] == "vs base" && i+1 < len(h2) && h2[i+1] == "P":
			cols = append(cols, col{"vs base", i, "delta"})
			i++
		}
	}

	table := [][]string{{unit}}
	for _, c := range cols {
		table[0] = append(table[0], c.title)
	}
	for _, r := range rows {
		line := []string{r[0]}
		for _, c := range cols {
			line = append(line, formatCell(r, c.idx, c.kind, unit))
		}
		table = append(table, line)
	}

	width := make([]int, len(table[0]))
	for _, line := range table {
		for j, s := range line {
			width[j] = max(width[j], len(s))
		}
	}
	for n, line := range table {
		fmt.Print("|")
		for j, s := range line {
			fmt.Printf(" %-*s |", width[j], s)
		}
		fmt.Println()
		if n == 0 {
			fmt.Print("|")
			for j := range line {
				fmt.Printf("-%s-|", strings.Repeat("-", width[j]))
			}
			fmt.Println()
		}
	}
	fmt.Println()
}

func formatCell(r []string, i int, kind, unit string) string {
	get := func(k int) string {
		if k < len(r) {
			return r[k]
		}
		return ""
	}
	switch kind {
	case "value":
		s := formatValue(get(i), unit)
		if ci := get(i + 1); ci != "" {
			s += " +/- " + strings.ReplaceAll(ci, "∞", "inf")
		}
		return s
	case "delta":
		s := get(i)
		if p := get(i + 1); p != "" {
			s += " (" + p + ")"
		}
		return s
	}
	return get(i)
}

func formatValue(s, unit string) string {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	switch unit {
	case "sec/op":
		switch {
		case v >= 1:
			return fmt.Sprintf("%.3f s", v)
		case v >= 1e-3:
			return fmt.Sprintf("%.2f ms", v*1e3)
		case v >= 1e-6:
			return fmt.Sprintf("%.2f us", v*1e6)
		default:
			return fmt.Sprintf("%.1f ns", v*1e9)
		}
	case "B/op":
		if v >= 1024 {
			return fmt.Sprintf("%.3f KiB", v/1024)
		}
		return fmt.Sprintf("%.0f B", v)
	case "allocs/op":
		return fmt.Sprintf("%.0f", v)
	}
	return strconv.FormatFloat(v, 'g', 6, 64)
}

# solitaire

A peg solitaire solver in Go, written as a **learning project**.

The goal is not to ship the fastest solver for one particular board, but to explore, one step at a time and with
measurements, the optimizations that make a **general-purpose** solver for peg-solitaire-like games practical: which
ideas pay off, which ones look good on paper and do not, and by how much. Every step is a numbered version, and almost every
version has benchmark results, so the effect of each idea can be read directly from the repository history.

> This package is educational. The API is minimal and unstable, and the code favours clarity and measurability over
> completeness. Most comments and the version history are written in French.

## What it solves

Boards are described by a 64-bit **mask** (at most 8x8 cells). Any mask works; a few are predefined:

| Mask                | Cells | Predefined start                                  |
|---------------------|-------|---------------------------------------------------|
| `MASK_5x5`          | 25    | `START_5x5` (hole at the centre)                  |
| `MASK_ENGLISHCROSS` | 33    | `START_ENGLISHCROSS` (hole at the centre)         |
| `MASK_EUROPEAN`     | 37    | `START_EUROPEAN` (hole at d3, see note below)     |
| `MASK_8x8`          | 64    | none                                              |

A game is a mask, a start position and an end position. An end position of `0` means "any single remaining peg"
(`SolveAnyEnd`).

```go
g, err := solitaire.NewGame(solitaire.MASK_ENGLISHCROSS, solitaire.START_ENGLISHCROSS, solitaire.CENTER_ENGLISHCROSS)
if err != nil {
	// the configuration was proven unsolvable before any search (see "Rule of three" below)
}
if err := g.Solve(); err != nil {
	// no solution
}

positions := g.Solution()  // the position after each move (the start position is not included)
moves, err := g.Moves()    // the moves between positions, rebuilt on demand
for _, m := range moves {
	fmt.Println(m)         // "d2->d4", "b3->d3", ...
}

e, _ := solitaire.NewGame(solitaire.MASK_EUROPEAN, solitaire.START_EUROPEAN, 0)
err = e.SolveAnyEnd() // finish with a single peg, anywhere
```

Moves are written as `from->to`, with a column letter (`a`-`h`) and a row number (`1`-`8`). The search does not store
them: they are recovered from two consecutive positions, only when `Moves` is called. `Position.Render(mask)` draws a
board; `TestPrintSolutions` prints complete solutions, board by board
(`go test -run TestPrintSolutions -v`).

Note on the European board: with the hole at the centre, the game is **provably unsolvable** (Bergholt). `START_EUROPEAN`
therefore uses the hole at d3, for which a 35-move solution exists.

## How it works

- A position is a `uint64` bitmap. A move is three cells in a line (jumping peg, jumped peg, landing cell), stored as
  two masks, so checking and playing a move is one AND and one XOR (`DoMove`).
- The search is a recursive depth-first search with backtracking. Every move removes exactly one peg, so the length of
  a solution is known in advance, and the solution buffer is allocated once with its final capacity.
- Everything below is built on top of that, one version at a time (`version.go` holds the full, detailed history).

## Optimizations that were kept

1. **Bitboards and O(1) moves** (v0.1.0). Positions and moves are bit masks; the search itself allocates nothing.
   Pre-sizing the move list in `NewGame` (v0.1.1) cut its allocations from 8 to 2 and its time by 44 %.
2. **Remembering dead ends** (v0.1.4). Moves commute, so the same position is reached through a huge number of paths.
   Storing positions whose every branch failed turned the English cross from about 474 ms into about 1.4 ms (350x).
3. **Rule of three** (v0.1.3). Two diagonal colourings of the board give parity invariants; if the target is
   inconsistent with the start, the game is rejected in O(1), before any search. This is what proves that the centred
   European board is unsolvable, instead of letting an exhaustive search run forever.
4. **Pagoda function** (v0.1.5, deltas in v0.1.6). Integer weights per cell such that no move can increase the total
   weight of the pegs; a position whose weight falls under the target's is pruned. The per-move variation is
   precomputed once and stored in the move, so the check costs one addition per node. It is kept, but it is a weak
   pruner (see below).
5. **Restarts with randomized move order and node budgets** (v0.1.5). A depth-first search has a heavy-tailed running
   time: some move orders are lucky, most are not. Giving up after a budget and restarting with a new order avoids the
   unlucky tail. The budget grows by only 3 % per restart, which guarantees termination; dead ends proved so far stay
   valid across restarts. The first attempt uses the fixed move order (v0.1.6), which is very fast on the English cross.
6. **Guided move ordering** (v0.1.9). At each node the children are scored with a cheap bitwise evaluation (number of
   legal moves afterwards, minus a penalty for isolated pegs) and tried best first; ties are broken by a random rank
   per move, so restarts still differ. On the European board, from about 2.5 s on average to about 14 ms, and a far
   shorter tail.
7. **Bit-parallel move generation and a dedicated hash set** (v0.1.10). Legal moves are computed for the four
   directions at once with shifts and masks instead of testing every move of the board, and the dead-end store is an
   open-addressing hash set instead of a Go map. English cross: 1.6 ms to 0.67 ms; European: 9.4 ms to 4.1 ms.
8. **Exact endgame table** (v0.1.7). Built by retrograde search (undoing moves from the final position), it contains
   every position with K pegs that can reach the target; any other position with K pegs is dead. It is exact, so it can
   never lose a solution. Since guided ordering made it unnecessary for ordinary searches, it is built lazily, only
   after many failed attempts.

Smaller points worth knowing: lazy initialization (nothing is built before it is needed), `NewGame` kept under
about 2 microseconds, and zero allocations per solve on the English cross.

## Ideas that were explored and not kept

Negative results are part of the point of this project; all of them are documented with their numbers in `version.go`.

- **Sorting moves by distance to the target's centre** (v0.1.2). A natural heuristic, but without pruning it made the
  English cross take more than 100 s instead of about 0.5 s. The code (`SortMoves`) is left in place, unused.
- **Several pagodas at once, up to 16** (v0.1.7). Implemented with 16-bit lanes packed in `[4]uint64` so that all
  pagodas were updated with 4 additions and checked with one test. It worked as designed but gave **no measurable
  gain** (12 fixed seeds on the European board: 40.2 s with no pagoda, 40.6 s with 1, 41.3 s with 16) while `NewGame`
  went from about 2 to about 18 microseconds. The pagoda sum stays far above its threshold until few pegs remain, when
  the search is already cheap. Reverted.
- **Pagodas aimed at a precise final cell** (v0.1.5). With the four possible last pegs of the European board, the search
  still exceeded 75 s per target: pruning alone does not fix a bad move order.
- **Board symmetries** (v0.1.8). Dead ends and the endgame table stored under a canonical form (minimum over up to
  8 symmetries that preserve the mask). Sound and tested, but slower: 20.6 s against 16.5 s on 12 seeds, because
  computing the canonical form (about 20 ns) at every node costs more than sharing dead ends saves.
- **Fewest-moves-first ordering** (Warnsdorff style). Opposite of the retained heuristic: it does not terminate on the
  European board.
- **Guiding the very first attempt**. The English cross went from about 1.6 ms to about 280 ms.
- **Doubling the restart budget**. Too few restarts for a heavy-tailed search (blocked for more than 100 s on some
  runs); a slow 3 % growth works much better.
- **Larger endgame tables** (K = 11 and more). Building the table costs more than it saves.
- **Another move enumeration order** (v0.1.10). Enumerating legal moves by direction then by cell instead of in the
  board's move order made the English cross miss its first-attempt budget: it is solved in 30,753 nodes in one order
  and needs much more in another. A reminder that some "easy" results are partly luck.
- **Using the rule of three inside the search**. Reasoned out rather than coded: the parities are invariants of the
  whole game, so they say nothing new about intermediate positions.

## Lessons

- Measure before and after every idea, and keep the numbers: several plausible ideas (multiple pagodas, symmetries)
  cost more than they gained.
- Move ordering and restarts mattered much more than any pruning rule on the European board.
- Heavy-tailed search times need distributions (many seeds), not a single run; benchmarks of randomized searches have
  large variance and should be read with that in mind.
- Moving work out of the hot loop (precomputed deltas, bit-parallel move generation, zero allocation) is a constant
  factor, but a visible one once the algorithm is right.

## Layout

| File                          | Content                                                              |
|-------------------------------|----------------------------------------------------------------------|
| `board.go`                    | positions, masks, predefined boards and rendering                    |
| `moves.go`                    | `Move`, `DoMove` / `UndoMove`, move generation, `SortMoves` (unused)  |
| `game.go`                     | `NewGame`, `Solve`, `SolveAnyEnd`, `Solution`, `Moves`, the search, the rule of three |
| `pagoda.go`                   | the pagoda function                                                  |
| `guide.go`                    | bit-parallel legal moves and guided ordering                         |
| `endgame.go`                  | hash set and endgame table                                           |
| `version.go`                  | full version history, with measurements and negative results         |
| `bench/`                      | benchmark results per version and the `benchstat` comparison         |
| `cmd/`                        | small tools used by the `Taskfile.yml` (version, benchmark table)    |

## Running

```
task test     # go test -count=1 ./...
task bench    # benchmarks into bench/bench-v<VERSION>.txt, then compares all versions
```

Latest measurements (Intel i7-10700, Go 1.27, 10 runs each): English cross about 0.67 ms with no allocation,
European board (single peg anywhere) about 4 ms, `NewGame` about 1.6 to 2.1 microseconds. Randomized searches vary a lot
from one seed to another: over 100 independent seeds the European board averages about 7 ms, with a worst case of about
47 ms.

## License

MIT, see `LICENSE`.

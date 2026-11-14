# Comparaison des benchmarks (benchstat)

```text
goos: windows
goarch: amd64
pkg: github.com/xavier268/solitaire
cpu: Intel(R) Core(TM) i7-10700 CPU @ 2.90GHz
                       │ bench\bench-v0.1.0.txt │       bench\bench-v0.1.1.txt        │
                       │         sec/op         │    sec/op     vs base               │
SolveEnglishCross-16               475.0m ± ∞ ¹   470.5m ± ∞ ¹        ~ (p=0.095 n=5)
NewGameEnglishCross-16            1586.0n ± ∞ ¹   876.0n ± ∞ ¹  -44.77% (p=0.008 n=5)
geomean                            868.0µ         642.0µ        -26.03%
¹ need >= 6 samples for confidence interval at level 0.95

                       │ bench\bench-v0.1.0.txt │         bench\bench-v0.1.1.txt         │
                       │          B/op          │     B/op       vs base                 │
SolveEnglishCross-16                0.000 ± ∞ ¹     0.000 ± ∞ ¹        ~ (p=1.000 n=5)
NewGameEnglishCross-16            6.227Ki ± ∞ ¹   3.375Ki ± ∞ ¹  -45.80% (p=0.008 n=5)
geomean                                       ²                  -26.38%               ²
¹ need >= 6 samples for confidence interval at level 0.95
² summaries must be >0 to compute geomean

                       │ bench\bench-v0.1.0.txt │        bench\bench-v0.1.1.txt        │
                       │       allocs/op        │  allocs/op   vs base                 │
SolveEnglishCross-16                0.000 ± ∞ ¹   0.000 ± ∞ ¹        ~ (p=1.000 n=5) ²
NewGameEnglishCross-16              9.000 ± ∞ ¹   2.000 ± ∞ ¹  -77.78% (p=0.008 n=5)
geomean                                       ³                -52.86%               ³
¹ need >= 6 samples for confidence interval at level 0.95
² all samples are equal
³ summaries must be >0 to compute geomean
```

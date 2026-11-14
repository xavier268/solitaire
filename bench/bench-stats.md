# Comparaison des benchmarks (benchstat)

- goos: windows
- goarch: amd64
- pkg: github.com/xavier268/solitaire
- cpu: Intel(R) Core(TM) i7-10700 CPU @ 2.90GHz

| sec/op                 | bench-v0.1.0.txt  | bench-v0.1.1.txt | vs base                  |
|------------------------|-------------------|------------------|--------------------------|
| SolveEnglishCross-16   | 475.00 ms +/- inf | 474.55 ms +/- 2% | ~ (p=0.440 n=5+10)       |
| NewGameEnglishCross-16 | 1.59 us +/- inf   | 891.4 ns +/- 16% | -43.80% (p=0.001 n=5+10) |
| geomean                | 867.96 us         | 650.38 us        | -25.07%                  |

| B/op                   | bench-v0.1.0.txt  | bench-v0.1.1.txt | vs base            |
|------------------------|-------------------|------------------|--------------------|
| SolveEnglishCross-16   | 0 B +/- inf       | 0 B +/- ?        | ~ (p=0.857 n=5+10) |
| NewGameEnglishCross-16 | 6.227 KiB +/- inf | 3.375 KiB +/- 0% | -45.80% (n=5+10)   |
| geomean                |                   |                  | -26.38%            |

| allocs/op              | bench-v0.1.0.txt | bench-v0.1.1.txt | vs base            |
|------------------------|------------------|------------------|--------------------|
| SolveEnglishCross-16   | 0 +/- inf        | 0 +/- 0%         | ~ (p=1.000 n=5+10) |
| NewGameEnglishCross-16 | 9 +/- inf        | 2 +/- 0%         | -77.78% (n=5+10)   |
| geomean                |                  |                  | -52.86%            |

Notes :

- need >= 6 samples for confidence interval at level 0.95
- summaries must be >0 to compute geomean
- all samples are equal

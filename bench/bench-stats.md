# Comparaison des benchmarks (benchstat)

- goos: windows
- goarch: amd64
- pkg: github.com/xavier268/solitaire
- cpu: Intel(R) Core(TM) i7-10700 CPU @ 2.90GHz

| sec/op                 | bench-v0.1.0.txt | bench-v0.1.1.txt | vs base                | bench-v0.1.2.txt | vs base                |
|------------------------|------------------|------------------|------------------------|------------------|------------------------|
| SolveEnglishCross-16   | 473.99 ms +/- 3% | 474.55 ms +/- 2% | ~ (p=0.684 n=10)       | 469.36 ms +/- 1% | -0.98% (p=0.004 n=10)  |
| NewGameEnglishCross-16 | 1.60 us +/- 3%   | 891.4 ns +/- 16% | -44.31% (p=0.000 n=10) | 846.4 ns +/- 15% | -47.12% (p=0.000 n=10) |
| NewGameEuropean-16     |                  |                  |                        | 1.15 us +/- 10%  |                        |
| geomean                | 870.98 us        | 650.38 us        | -25.33%                | 76.92 us         | -27.63%                |

| B/op                   | bench-v0.1.0.txt | bench-v0.1.1.txt | vs base                | bench-v0.1.2.txt | vs base                |
|------------------------|------------------|------------------|------------------------|------------------|------------------------|
| SolveEnglishCross-16   | 0 B +/- ?        | 0 B +/- ?        | ~ (p=1.000 n=10)       | 0 B +/- ?        | ~ (p=1.000 n=10)       |
| NewGameEnglishCross-16 | 6.227 KiB +/- 0% | 3.375 KiB +/- 0% | -45.80% (p=0.000 n=10) | 3.375 KiB +/- 0% | -45.80% (p=0.000 n=10) |
| NewGameEuropean-16     |                  |                  |                        | 4.281 KiB +/- 0% |                        |
| geomean                |                  |                  | -26.38%                |                  | -26.38%                |

| allocs/op              | bench-v0.1.0.txt | bench-v0.1.1.txt | vs base                | bench-v0.1.2.txt | vs base                |
|------------------------|------------------|------------------|------------------------|------------------|------------------------|
| SolveEnglishCross-16   | 0 +/- 0%         | 0 +/- 0%         | ~ (p=1.000 n=10)       | 0 +/- 0%         | ~ (p=1.000 n=10)       |
| NewGameEnglishCross-16 | 9 +/- 0%         | 2 +/- 0%         | -77.78% (p=0.000 n=10) | 2 +/- 0%         | -77.78% (p=0.000 n=10) |
| NewGameEuropean-16     |                  |                  |                        | 2 +/- 0%         |                        |
| geomean                |                  |                  | -52.86%                |                  | -52.86%                |

Notes :

- summaries must be >0 to compute geomean
- all samples are equal

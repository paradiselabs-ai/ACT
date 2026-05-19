# Lazy-Fetch SPIL — Aggregate Report

**Run:** `2026-05-19T13-40-25-130Z`
**Fixture:** `lazy-01-api-bootstrap` (11 sections)
**Models:** google/gemma-3-12b, qwen/qwen2.5-coder-14b, openai/gpt-oss-20b
**Trials per (model, arm):** 2
**Seed:** 1779198025130

## Per-arm averages (across all models × trials)

| Arm | n | Avg score | Avg iter | Prompt tok | Compl tok | Total tok | vs A total | Fetches (unique/total) |
|---|---|---|---|---|---|---|---|---|
| A:full-SPIL | 6 | 71.2 | 7.2 | 27083 | 1668 | 28751 | +0.0% | 0.0/0.0 |
| B1:lazy-minimal | 6 | 81.2 | 17.0 | 56586 | 1985 | 58571 | +103.7% | 8.7/9.2 |
| B2:lazy-guided | 6 | 85.2 | 16.8 | 53249 | 1975 | 55224 | +92.1% | 8.2/8.2 |
| C:full-plain | 6 | 64.7 | 9.0 | 30276 | 1535 | 31811 | +10.6% | 0.0/0.0 |

## Per-model breakdown

### google/gemma-3-12b

| Arm | Score | Iter | Prompt | Compl | Total | Fetches | Stop |
|---|---|---|---|---|---|---|---|
| B1 t2 | 90 | 18 | 68857 | 2290 | 71147 | 9/9 | mark_complete |
| C t1 | 0 | 4 | 10832 | 810 | 11642 | 0/0 | too-many-nudges |
| A t2 | 0 | 4 | 12428 | 846 | 13274 | 0/0 | too-many-nudges |
| A t1 | 100 | 11 | 48886 | 1805 | 50691 | 0/0 | mark_complete |
| C t2 | 0 | 4 | 10848 | 806 | 11654 | 0/0 | too-many-nudges |
| B2 t2 | 70 | 13 | 42301 | 1612 | 43913 | 4/4 | mark_complete |
| B2 t1 | 77 | 13 | 43684 | 1839 | 45523 | 4/4 | mark_complete |
| B1 t1 | 100 | 20 | 82565 | 2619 | 85184 | 10/10 | mark_complete |

### qwen/qwen2.5-coder-14b

| Arm | Score | Iter | Prompt | Compl | Total | Fetches | Stop |
|---|---|---|---|---|---|---|---|
| B1 t1 | 97 | 18 | 60343 | 1877 | 62220 | 11/11 | mark_complete |
| C t1 | 97 | 13 | 56349 | 2034 | 58383 | 0/0 | mark_complete |
| A t1 | 97 | 6 | 24321 | 1728 | 26049 | 0/0 | mark_complete |
| C t2 | 97 | 8 | 27202 | 1716 | 28918 | 0/0 | mark_complete |
| A t2 | 33 | 1 | 5258 | 1556 | 6814 | 0/0 | mark_complete |
| B2 t2 | 97 | 17 | 57736 | 1953 | 59689 | 11/11 | mark_complete |
| B2 t1 | 97 | 17 | 57702 | 1942 | 59644 | 11/11 | mark_complete |
| B1 t2 | 97 | 19 | 64014 | 1866 | 65880 | 11/11 | mark_complete |

### openai/gpt-oss-20b

| Arm | Score | Iter | Prompt | Compl | Total | Fetches | Stop |
|---|---|---|---|---|---|---|---|
| C t2 | 97 | 11 | 31933 | 1763 | 33696 | 0/0 | mark_complete |
| A t2 | 100 | 10 | 33296 | 2007 | 35303 | 0/0 | mark_complete |
| B1 t2 | 50 | 14 | 33992 | 1648 | 35640 | 6/9 | mark_complete |
| B1 t1 | 53 | 13 | 29744 | 1612 | 31356 | 5/5 | mark_complete |
| A t1 | 97 | 11 | 38310 | 2063 | 40373 | 0/0 | mark_complete |
| C t1 | 97 | 14 | 44493 | 2078 | 46571 | 0/0 | mark_complete |
| B2 t1 | 77 | 20 | 57558 | 2143 | 59701 | 9/9 | mark_complete |
| B2 t2 | 93 | 21 | 60513 | 2358 | 62871 | 10/10 | mark_complete |

## Hypothesis verdicts

```
H1: B2 prompt tokens ≥30% fewer than A     FAIL  [actual: 96.6%]
H2: all arms avg score ≥90                 FAIL  [A=71.2 B1=81.2 B2=85.2 C=64.7]
H3: A total tokens ≤ C total + 10%         PASS  [delta: -9.6%]
H4: B2 score ≥ B1, B2 tokens ≤ B1 + 15%    PASS  [score: B1=81.2, B2=85.2; token delta B2vsB1: -5.7%]
```

## Files
- `trials.csv` — raw per-trial data
- `<model>__<arm>__t<N>.md` — per-trial dumps with tool log + assertion outcomes
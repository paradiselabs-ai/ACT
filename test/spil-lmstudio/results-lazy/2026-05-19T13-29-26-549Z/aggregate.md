# Lazy-Fetch SPIL — Aggregate Report

**Run:** `2026-05-19T13-29-26-549Z`
**Fixture:** `lazy-01-api-bootstrap` (11 sections)
**Models:** openai/gpt-oss-20b
**Trials per (model, arm):** 1
**Seed:** 1779197366548

## Per-arm averages (across all models × trials)

| Arm | n | Avg score | Avg iter | Prompt tok | Compl tok | Total tok | vs A total | Fetches (unique/total) |
|---|---|---|---|---|---|---|---|---|
| B1:lazy-minimal | 1 | 73.0 | 19.0 | 54757 | 1988 | 56745 | - | 7.0/11.0 |

## Per-model breakdown

### openai/gpt-oss-20b

| Arm | Score | Iter | Prompt | Compl | Total | Fetches | Stop |
|---|---|---|---|---|---|---|---|
| B1 t1 | 73 | 19 | 54757 | 1988 | 56745 | 7/11 | mark_complete |

## Hypothesis verdicts

```
H1: B2 prompt tokens ≥30% fewer than A     FAIL  [actual: n/a]
H2: all arms avg score ≥90                 FAIL  [A=- B1=73.0 B2=- C=-]
H3: A total tokens ≤ C total + 10%         FAIL  [delta: n/a]
H4: B2 score ≥ B1, B2 tokens ≤ B1 + 15%    FAIL  [score: B1=73.0, B2=-; token delta B2vsB1: n/a]
```

## Files
- `trials.csv` — raw per-trial data
- `<model>__<arm>__t<N>.md` — per-trial dumps with tool log + assertion outcomes
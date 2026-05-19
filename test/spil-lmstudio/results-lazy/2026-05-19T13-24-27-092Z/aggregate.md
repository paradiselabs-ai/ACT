# Lazy-Fetch SPIL — Aggregate Report

**Run:** `2026-05-19T13-24-27-092Z`
**Fixture:** `lazy-01-api-bootstrap` (11 sections)
**Models:** openai/gpt-oss-20b
**Trials per (model, arm):** 1
**Seed:** 1779197067091

## Per-arm averages (across all models × trials)

| Arm | n | Avg score | Avg iter | Prompt tok | Compl tok | Total tok | vs A total | Fetches (unique/total) |
|---|---|---|---|---|---|---|---|---|
| A:full-SPIL | 1 | 53.0 | 4.0 | 8892 | 707 | 9599 | +0.0% | 0.0/0.0 |

## Per-model breakdown

### openai/gpt-oss-20b

| Arm | Score | Iter | Prompt | Compl | Total | Fetches | Stop |
|---|---|---|---|---|---|---|---|
| A t1 | 53 | 4 | 8892 | 707 | 9599 | 0/0 | final-message |

## Hypothesis verdicts

```
H1: B2 prompt tokens ≥30% fewer than A     FAIL  [actual: n/a]
H2: all arms avg score ≥90                 FAIL  [A=53.0 B1=- B2=- C=-]
H3: A total tokens ≤ C total + 10%         FAIL  [delta: n/a]
H4: B2 score ≥ B1, B2 tokens ≤ B1 + 15%    FAIL  [score: B1=-, B2=-; token delta B2vsB1: n/a]
```

## Files
- `trials.csv` — raw per-trial data
- `<model>__<arm>__t<N>.md` — per-trial dumps with tool log + assertion outcomes
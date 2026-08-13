# Spaced Repetition Logic

This explains how GoLeet decides when you'll see a problem again. All of
it lives in `calculateNextReview()` in `logic.go` — this doc is just a
plain-language walkthrough of that one function.

## The two numbers that drive everything

Every problem tracks two fields in its frontmatter:

* **`Interval`** — how many days from your last review until the next
  one. This *is* the schedule: `Next Review = Last Reviewed + Interval days`.
* **`Streak`** — how many reviews in a row you've rated "Good" or "Easy"
  without a "Weak" or "Forgot" in between. It's for your own visibility
  on the dashboard — it doesn't feed back into the interval math.

A new problem starts at `Interval: 0`, `Streak: 0`, `Next Review: today`,
so it shows up in "Due Today" immediately.

## The rule

After every review you self-rate how it went, and that rating maps to a
new interval:

| Confidence         | New Interval                                              | New Streak    |
| ------------------- | ----------------------------------------------------------- | --------------- |
| **Forgot** / Todo | `1` day — flat reset                                       | resets to `0` |
| **Weak**            | `3` days — flat reset                                       | resets to `1` |
| **Good**            | `4` days if this is the first real review, otherwise **current interval × 2** | `+1`          |
| **Easy**            | `7` days if this is the first real review, otherwise **current interval × 3** | `+1`          |

One safety cap applies after all of the above: **the interval never
exceeds 180 days**, no matter how long a streak you're on. That's a hard
ceiling in `calculateNextReview()`, not a suggestion — a problem you keep
acing will plateau at reviewing roughly every 6 months rather than
drifting out to a year+.

Note that "Good" and "Easy" are the only ratings that look at your
*current* interval — "Weak" and "Forgot" always reset to a flat number
regardless of how far out you'd drifted. A problem you'd stretched to a
90-day gap and then forgot goes straight back to a 1-day interval, not
some fraction of 90.

## Worked example: one problem over several reviews

Say you add **217 — Contains Duplicate** on Day 0. `Interval` starts at
`0`, so it's due immediately.

| # | Day | Confidence | Why                                              | New Interval | Next Review | Streak |
| - | --- | ----------- | -------------------------------------------------- | -------------- | -------------- | -------- |
| 1 | 0   | Good        | first real review → flat `4`                      | 4 days        | Day 4         | 1        |
| 2 | 4   | Good        | current interval was 4 → `4 × 2`                   | 8 days        | Day 12        | 2        |
| 3 | 12  | Easy        | current interval was 8 → `8 × 3`                   | 24 days       | Day 36        | 3        |
| 4 | 36  | Weak        | flat reset, ignores the 24-day interval it came from | 3 days        | Day 39        | 1        |
| 5 | 39  | Forgot      | flat reset                                          | 1 day         | Day 40        | 0        |
| 6 | 40  | Good        | back to first-review logic → flat `4`              | 4 days        | Day 44        | 1        |

A couple of things this table shows:

* **Growth compounds fast on "Easy."** Two good reviews in a row from a
  fresh problem takes you from a 1-day check to a 24-day gap in three
  sessions.
* **One bad review erases the compounding.** Streak 3 → Streak 0/1 in a
  single "Weak" or "Forgot," and the interval snaps back down to a flat
  number rather than decaying gradually.
* **"First real review" is judged by `Interval == 0`,** not by attempt
  count. If you rate a problem "Weak" (interval → 3) and then later rate
  it "Good," that's *not* treated as a first review — you'll get `3 × 2 = 6`,
  not the flat `4`. The flat 4/7 only kicks in when the interval is
  literally zero, which in practice means either a brand-new problem or
  one you've never rated as anything other than the initial "Todo."

## Worked example: hitting the cap

A problem that keeps getting rated "Easy" from a longer starting
interval grows quickly:

| # | Confidence | Interval before | Interval after | Capped? |
| - | ------------ | ------------------ | ----------------- | --------- |
| 1 | Easy         | 20                 | 60                | no        |
| 2 | Easy         | 60                 | 180               | no        |
| 3 | Easy         | 180                | 540 → **180**    | **yes**   |

Once you're at the 180-day ceiling, further "Easy" ratings just keep you
there — the schedule won't push a problem out further than ~6 months.

## Where this shows up in the app

* **Dashboards (menu `4`/`5`)** show each problem's current `Interval`,
  `Streak`, `Next Review` date, and a plain-text status (`in 5d`,
  `due today`, `3d overdue`) computed from `Next Review` vs. today —
  see `dashboard.go`.
* **Review flow (menu `2`/`3`)** is what actually calls
  `calculateNextReview()` after you pick a confidence rating in
  `saveReview()` (`model.go`), and writes the new `Interval`, `Streak`,
  `Next Review`, and incremented `Attempts` back to the note's
  frontmatter via `saveNote()` (`vault.go`).

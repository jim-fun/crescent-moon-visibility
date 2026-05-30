# Phase 0: Scope Lock & Likelihood Definition

**Branch**: wp-plugin-precomputed-cities  
**Status**: In Progress (May 2026)  
**Owner**: Human (Grok + User) — with support from models for options generation

This is the most important phase. We will not proceed to coding until these decisions are locked.

---

## 1. City List (Final Proposed)

We will start with these 10 cities (good geographic + cultural spread):

| Slug       | Name              | Lat      | Lon       | Notes                          |
|------------|-------------------|----------|-----------|--------------------------------|
| jerusalem  | Jerusalem         | 31.7683  | 35.2137   | Existing preset, high cultural relevance |
| cairo      | Cairo             | 30.0444  | 31.2357   | North Africa / Middle East     |
| london     | London            | 51.5074  | -0.1278   | Northern Europe                |
| istanbul   | Istanbul          | 41.0136  | 28.9550   | Bridge between continents      |
| mumbai     | Mumbai            | 19.0760  | 72.8777   | South Asia                     |
| tokyo      | Tokyo             | 35.6762  | 139.6503  | East Asia                      |
| rio        | Rio de Janeiro    | -22.9068 | -43.1729  | South America                  |
| capetown   | Cape Town         | -33.9249 | 18.4241   | Southern Africa                |
| dallas     | Dallas            | 32.7767  | -96.7970  | Existing preset, North America |
| melbourne  | Melbourne         | -37.8136 | 144.9631  | Existing preset, Oceania       |

**Locked Decision**: Keep the original 10 cities. Add the following 3 cities that are particularly important for traditional new moon / crescent moon observation (especially in the context of Islamic lunar calendar practices):

| Slug     | Name     | Lat      | Lon      | Notes |
|----------|----------|----------|----------|-------|
| mecca    | Mecca    | 21.3891  | 39.8579  | Highest religious importance for crescent sighting |
| karachi  | Karachi  | 24.8607  | 67.0011  | Very active official moon sighting tradition |
| rabat    | Rabat    | 33.9716  | -6.8498  | Important North African observation site |

**Final City Count**: 13 cities.

---

## 2. "Likelihood Visibility" Definition — Locked Decision

**Confirmed**: We will use the **Yallop** criterion (matching the main project's default).

For the display of "likelihood" over a 10-year window, we are starting with **Option C (Distribution Table)** as the primary view, with the ability to also show best/worst in the period. This balances honesty with usability. We can evolve the presentation later based on user feedback.

We need a clear, honest, observer-friendly way to express visibility over a 10-year window.

### Option A: Best Category Observed
- Show the single best Effective category that occurred in any 3-day window during the selected 10-year period.
- Simple and dramatic ("In this decade, the best you could have seen was A").

**Pros**: Very easy to understand.
**Cons**: Can be misleading — one exceptional window doesn't represent typical conditions.

### Option B: Percentage of "Good" Windows
- Define "Good" as windows where the best Effective category was B or better.
- Display: "X out of Y new moon windows (Z%) had at least B visibility."

**Pros**: Gives a real sense of frequency.
**Cons**: Requires choosing a threshold (B or better? C or better?).

### Option C: Distribution Table (Recommended starting point)
- For the selected period, show a breakdown:
  - % of windows where best effective was A
  - % where best was B
  - % where best was C
  - etc.
- Plus the single best and worst observed.

**Pros**: Honest and information-rich.
**Cons**: Slightly more complex for casual users.

### Option D: "Typical Visibility" + Extremes
- Show the most common (mode) best-effective category for the period.
- Also show the single best and single worst window.

**Pros**: Good balance of simplicity and honesty.

---

## 3. Data Granularity

We will store **both** levels:

**Per New Moon (detailed)**:
- City
- New moon date
- Year
- 3 daily raw categories
- Best raw category in window
- Best effective category (under standard clear conditions)
- Optional: Q and age at best time

**Per Year (aggregated for quick views)**:
- City + Year
- Number of new moons that year
- Best effective category seen that year
- Count of windows with best effective ≥ A, ≥ B, ≥ C, etc.
- Simple "good visibility rate" (e.g. % of windows with best ≥ B)

This gives us flexibility for both detailed tables and summary cards.

---

## 4. Time Range & Windowing

- Data from **2006** onward (as requested).
- Users select **10-year windows** (e.g. 2006–2015, 2015–2024, 2018–2027, etc.).
- We will pre-compute and ship data through at least 2028–2030 so the plugin remains useful for a while without immediate updates.

---

## 5. Atmospheric Assumptions

Pre-computed data will assume **standard clear conditions** (0% cloud cover, transparency 7–8).

The plugin can later offer a simple client-side adjustment using the existing heuristic (very lightweight).

---

## Next Actions for Phase 0

1. User reviews and approves/modifies the city list.
2. User chooses or refines the "likelihood" definition (recommend starting with **Option C – Distribution Table** + best/worst).
3. Lock the data schema (per-new-moon + yearly aggregate).
4. Create a small decision record document.

Once these are locked, we move to Phase 1 (Data Generator Tooling), where we will heavily use Ollama first for exploration, then Claude for the production generator.

---

**Status**: Awaiting user decisions on:
- Final city list
- Primary "likelihood" metric
- Any changes to proposed granularity

Please reply with your preferences or modifications so we can lock Phase 0 and begin Phase 1.
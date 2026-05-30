# Phase 0 – Final Locked Decisions

**Date**: 2026-05-28  
**Branch**: wp-plugin-precomputed-cities  
**Status**: COMPLETE

## 1. Cities (Final List – 13 cities)

**Approved list** (original 10 kept + 3 added for new moon observation importance):

| Slug      | Name             | Latitude | Longitude | Rationale |
|-----------|------------------|----------|-----------|---------|
| jerusalem | Jerusalem        | 31.7683  | 35.2137   | Existing preset, high cultural relevance |
| dallas    | Dallas           | 32.7767  | -96.7970  | Existing preset |
| melbourne | Melbourne        | -37.8136 | 144.9631  | Existing preset |
| cairo     | Cairo            | 30.0444  | 31.2357   | North Africa / Middle East |
| london    | London           | 51.5074  | -0.1278   | Northern Europe |
| istanbul  | Istanbul         | 41.0136  | 28.9550   | Bridge between continents |
| mumbai    | Mumbai           | 19.0760  | 72.8777   | South Asia |
| tokyo     | Tokyo            | 35.6762  | 139.6503  | East Asia |
| rio       | Rio de Janeiro   | -22.9068 | -43.1729  | South America |
| capetown  | Cape Town        | -33.9249 | 18.4241   | Southern Africa |
| **mecca** | **Mecca**        | 21.3891  | 39.8579   | **Highest religious importance for crescent sighting** |
| **karachi**| **Karachi**     | 24.8607  | 67.0011   | **Very active official moon sighting tradition** |
| **rabat** | **Rabat**        | 33.9716  | -6.8498   | **Important North African observation site** |

**Total**: 13 cities. This list is now locked for the initial release.

## 2. Visibility Criterion

**Locked**: **Yallop** criterion (matching the main project's default and the existing web UI).

## 3. Data Schema Approach

**Locked direction**:
- Use the two-table design outlined in `database-schema.md`
- `crescent_observations` (detailed per-new-moon data)
- `crescent_yearly_summary` (pre-aggregated for fast queries)
- Proper indexing for common WordPress queries (city + year range, city + date range)
- Include `data_version` column for provenance and safe re-imports

The exact final column names can be adjusted slightly during Phase 1/3 if needed, but the overall structure is approved.

## 4. Likelihood / Display Approach (Initial)

Start with **Option C (Distribution Table)** as the primary view:
- Show breakdown of best effective categories across the selected 10-year window
- Also surface best and worst observed
- This balances honesty with usability

We will refine the exact presentation during Phase 4 based on early testing feedback (as per user's instruction).

## 5. Time Range

- Data from 2006 onward
- Support 10-year selectable windows
- Generator will target 2006–2030 (to allow projection)

---

**Phase 0 Status**: COMPLETE

We now have enough locked decisions to proceed through the remaining phases without blocking.

Minor adjustments to schema columns, UI labels, or "likelihood" presentation will be handled during testing feedback as requested.
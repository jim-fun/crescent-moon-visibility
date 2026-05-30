# Database Schema for WordPress Plugin (MariaDB Optimized)

**Goal**: Design tables that are accurate, performant, and well-indexed for the expected query patterns in a minimal WordPress plugin.

## Design Principles

- Prioritize **read performance** (the plugin is mostly a viewer).
- Keep data **accurate and normalized** where it makes sense.
- Use proper indexes for the most common queries:
  - Filter by city + year range
  - Filter by city + date range
  - Aggregations per year per city
- Use `utf8mb4` and `InnoDB`.
- Keep the schema simple enough that a basic WP admin can understand it.

---

## Recommended Tables

### 1. `crescent_cities` (Reference / Lookup)

Small static table for the cities.

```sql
CREATE TABLE `crescent_cities` (
    `id`            INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `slug`          VARCHAR(50)  NOT NULL,
    `name`          VARCHAR(100) NOT NULL,
    `latitude`      DECIMAL(10,6) NOT NULL,
    `longitude`     DECIMAL(10,6) NOT NULL,
    `is_active`     TINYINT(1)   NOT NULL DEFAULT 1,
    `sort_order`    SMALLINT     NOT NULL DEFAULT 100,
    PRIMARY KEY (`id`),
    UNIQUE KEY `ux_slug` (`slug`),
    KEY `ix_active_sort` (`is_active`, `sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### 2. `crescent_observations` (Core Data - Per New Moon)

This is the main detailed table.

```sql
CREATE TABLE `crescent_observations` (
    `id`                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `city_id`               INT UNSIGNED NOT NULL,
    `new_moon_date`         DATE NOT NULL,
    `year`                  SMALLINT NOT NULL,
    `raw_day_0`             CHAR(1)  NOT NULL,           -- Raw Yallop category for conjunction day + 0
    `raw_day_1`             CHAR(1)  NOT NULL,
    `raw_day_2`             CHAR(1)  NOT NULL,
    `best_raw`              CHAR(1)  NOT NULL,
    `best_effective`        CHAR(1)  NOT NULL,           -- Under standard clear conditions
    `q_at_best`             DECIMAL(6,4) NULL,
    `moon_age_at_best`      DECIMAL(5,2) NULL,           -- hours since conjunction
    `data_version`          VARCHAR(50) NOT NULL,        -- e.g. "yallop-2026.05-renderer-vX"
    PRIMARY KEY (`id`),
    UNIQUE KEY `ux_city_newmoon` (`city_id`, `new_moon_date`),
    KEY `ix_city_year` (`city_id`, `year`),
    KEY `ix_new_moon_date` (`new_moon_date`),
    KEY `ix_city_date_range` (`city_id`, `new_moon_date`),
    KEY `ix_year` (`year`),
    CONSTRAINT `fk_city` FOREIGN KEY (`city_id`) REFERENCES `crescent_cities` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### 3. `crescent_yearly_summary` (Aggregated for Fast Queries)

Pre-computed yearly aggregates. This table will be the main source for most UI queries.

```sql
CREATE TABLE `crescent_yearly_summary` (
    `id`                        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `city_id`                   INT UNSIGNED NOT NULL,
    `year`                      SMALLINT NOT NULL,
    `new_moons_count`           TINYINT NOT NULL,
    `best_effective`            CHAR(1) NOT NULL,
    `windows_with_a`            TINYINT NOT NULL DEFAULT 0,
    `windows_with_b_or_better`  TINYINT NOT NULL DEFAULT 0,
    `windows_with_c_or_better`  TINYINT NOT NULL DEFAULT 0,
    `data_version`              VARCHAR(50) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `ux_city_year` (`city_id`, `year`),
    KEY `ix_city_year` (`city_id`, `year`),
    KEY `ix_year` (`year`),
    CONSTRAINT `fk_summary_city` FOREIGN KEY (`city_id`) REFERENCES `crescent_cities` (`id`)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

## Why This Design is Good for WordPress + MariaDB

- **Fast filtering** by city + year range (most common UI query).
- `UNIQUE KEY` on `(city_id, new_moon_date)` prevents duplicates.
- Composite indexes support both detailed and summary queries efficiently.
- `yearly_summary` table means the plugin can often answer "show me 10 years for Cairo" with a single fast indexed query.
- `data_version` column allows future re-imports without destroying history.

---

## Import Strategy Recommendation

1. The generator outputs two files:
   - `observations.json` → loads into `crescent_observations`
   - `yearly_summary.json` → loads into `crescent_yearly_summary`

2. WordPress import script does:
   - Truncate or replace by `data_version`
   - Bulk insert (use `LOAD DATA` if possible, or batched inserts)
   - Rebuild any needed summary data if not pre-computed

---

## Future Considerations

- If we later want to support user-selected atmospheric adjustments, we can store the raw categories and compute effective on the fly in PHP (very cheap).
- Consider adding a `visibility_score` numeric column later if we want to do statistical analysis.

---

**Next**: Once Phase 0 decisions are locked, we can finalize the exact column names and move to building the generator + import scripts.
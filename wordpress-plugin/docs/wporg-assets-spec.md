# WordPress.org Asset & Screenshot Spec

Everything you need to produce the visual assets the .org directory shows on the
plugin page. These are **not** shipped inside the plugin zip — they live in the
`/assets/` folder of the plugin's SVN repository (created after the plugin is
approved). Keep the source files in this repo under `wordpress-plugin/assets-wporg/`
so they're versioned alongside the code.

> Naming matters. The directory only picks up files with these exact names.

---

## 1. Icon  (shown in search results & plugin header)

| File | Size | Notes |
|------|------|-------|
| `icon-128x128.png` | 128×128 | Required. |
| `icon-256x256.png` | 256×256 | Retina. Same art, 2×. |
| `icon.svg` | vector | Optional, preferred if you have clean vector art — .org renders it at any size. |

**Design direction:** a thin waxing crescent (the "young crescent" the plugin is
about) against a dark twilight-gradient sky. Avoid religious symbolism — the plugin
is astronomical/observational, not denominational. Keep it legible at 128px: one
crescent, simple gradient, no text.

- Background: deep twilight `#0b1d3a → #1a2a4f` vertical gradient.
- Crescent: warm off-white `#f4ead0`, ~15% illuminated (thin), tilted ~20°.
- Optional: one faint star.

## 2. Banner  (top of the plugin page)

| File | Size | Notes |
|------|------|-------|
| `banner-772x250.png` | 772×250 | Required. |
| `banner-1544x500.png` | 1544×500 | Retina. Same art, 2×. |

**Design direction:** same twilight palette as the icon, crescent on the right
third, plugin name set left in a clean sans (Inter / system UI). Keep the left
~60% low-contrast so the auto-overlaid plugin title (.org renders its own text on
some themes) stays readable. Safe margin: keep art 40px from every edge.

- Title text: **Young Crescent Moon Visibility**
- Optional tagline: *Pre-computed Yallop visibility for any evening.*

---

## 3. Screenshots  (the `== Screenshots ==` list in readme.txt)

The readme already declares three, in this order. Filenames must be
`screenshot-1.png`, `screenshot-2.png`, `screenshot-3.png` (PNG or JPG; PNG
preferred for UI). No fixed size, but **1200px wide** is the sweet spot — crisp on
retina, not huge. Capture at 2× device-pixel-ratio if you can.

| # | Caption (already in readme.txt) | What to capture |
|---|--------------------------------|-----------------|
| 1 | *The interactive tool: city/date selectors, atmospheric sliders, and three per-evening result cards.* | The **light-theme** `[crescent_visibility_interactive]` shortcode on a page. City = Jerusalem, a New Moon date selected, sliders at defaults, all three evening cards visible (at least one showing a real category, not "not a good window"). Include the small context map. |
| 2 | *Dark theme.* | Same shortcode rendered with `theme="dark"`, same city/date so it reads as a direct comparison. |
| 3 | *Tools → Crescent Visibility data import screen.* | The wp-admin **Tools → Crescent Visibility** page after a successful import, showing the "✓ Data loaded" status and the "Last Import Attempt" diagnostic table. |

### How to capture cleanly

1. Use a wide-but-not-huge browser window (~1280px content width). Zoom 100%.
2. Light/dark shots: put `[crescent_visibility_interactive]` and
   `[crescent_visibility_interactive theme="dark"]` on two draft pages, pick the
   **same city + New Moon date** on both before capturing so they line up.
3. For shot 1, pick a date where at least one evening grades favorably (category
   A–D) so the cards show color, not all "not a good window."
4. Crop to just the tool (shot 1 & 2) and just the admin content column (shot 3) —
   no browser chrome, no surrounding theme header/footer if avoidable.
5. Export PNG, ~1200px wide. Filenames exactly `screenshot-1/2/3.png`.

---

## 4. Where files go (after approval)

After the plugin is approved you get an SVN repo:
`https://plugins.svn.wordpress.org/young-crescent-moon-visibility/` *(slug TBD by .org)*

```
/trunk/          ← the plugin code (contents of plugin/, what's in the zip)
/assets/         ← icon-*, banner-*, screenshot-* go HERE (not in trunk)
/tags/0.5.2/     ← tagged release copy of trunk
```

The `/assets/` files are **never** in the downloadable zip — they're directory
chrome only. Commit them to `/assets/` via SVN.

---

## 5. Pre-flight checklist before you generate assets

- [ ] Decide final slug intent (likely `young-crescent-moon-visibility`). .org
      assigns it from the readme title at submission; you can't pick it freely.
- [ ] Set `Contributors:` in readme.txt to your wordpress.org username
      (currently `(your-wporg-username)`).
- [ ] Set `Tested up to:` to the latest WP you've actually tested
      (currently `6.7`; your site runs 7.0 — test there and bump).
- [ ] Run **Plugin Check (PCP)** in a real WP install; fix any errors.
- [ ] Produce icon (128/256), banner (772×250 / 1544×500), screenshots 1–3.

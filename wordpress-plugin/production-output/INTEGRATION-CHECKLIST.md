# Integration Checklist — Applying Claude's Production Output

Use this when pasting Claude's generated file contents.

## 1. Preparation (done)
- [x] Backup created in `backups/pre-claude-production-YYYYMMDD-HHMM/`
- [x] `production-output/MIGRATION-AND-TESTING-NOTES.md` written

## 2. File-by-File Replacement Order (recommended)

Claude delivered the following production files:

1. **plugin/crescent-visibility.php** — ✅ **DONE**
   - Triple-require mess removed.
   - Modules required once in clean dependency order.
   - `cvi_schema_version` recorded on activation.
   - v0.2.0 + production synthesis note in header.

2. **plugin/includes/interactive.php** — NEXT (please paste full code)
   - `Crescent_Visibility_Interactive` class
   - Nonce-validated AJAX (`cvi_get_years`, `cvi_get_newmoons`)
   - Smart defaults
   - City list sourced from DB (locked-13 fallback)
   - Version-gated `dbDelta` auto-upgrade
   - Shared PHP heuristic (exact port from main.go)

3. **plugin/public/interactive-renderer.php**
   - Class-based markup (CSS classes, not inline styles)
   - Smart defaults via `wp_localize_script`
   - `data-theme="dark"` support
   - Server-rendered `<noscript>` fallback cards + context-map container
   - Fixed `plugin_dir_url()` bug

4. **plugin/assets/js/interactive.js**
   - Debounced live re-grading
   - Sends nonce
   - `AbortController` + per-(city,year) cache
   - On-demand Leaflet with graceful CDN-failure handling
   - High-fidelity cards + legend
   - Loading/error status line

5. **plugin/assets/css/interactive.css**
   - Scoped light styles + `data-theme="dark"` zinc variant

6. **plugin/admin/admin-page.php**
   - Documents shortcode attributes + auto-upgrade vs re-import distinction

7. **production-output/MIGRATION-AND-TESTING-NOTES.md**
   - Already written with full testing checklist, migration story, and UX validation steps.

**Next action**: Paste the full code for `plugin/includes/interactive.php` (Claude's version) when ready.

## 3. After Replacing Files

```bash
# Quick syntax check
php -l wordpress-plugin/plugin/crescent-visibility.php
php -l wordpress-plugin/plugin/includes/interactive.php
php -l wordpress-plugin/plugin/public/interactive-renderer.php
node --check wordpress-plugin/plugin/assets/js/interactive.js 2>/dev/null || echo "node not available"

# Optional: run local test harness
php wordpress-plugin/test-local.php
```

## 4. WordPress Testing

- Deactivate + reactivate the plugin (to trigger activation with new schema version)
- Re-import a real JSON (recommended for Age/Q data)
- Test the shortcode on a page

## 5. If Issues Arise

Paste the problematic section here and I will help debug or adjust.

---

Current backup location: See the `backups/` folder created during this session.
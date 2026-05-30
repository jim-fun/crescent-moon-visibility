# Browser-Agent Prompt — WordPress.org plugin submission

Paste the block below into Claude-in-your-browser (Claude for Chrome / a browser
agent) while you're signed in to wordpress.org. It fills the "Add Your Plugin"
review form but **stops before the final submit** so you can eyeball it.

> **Before you start:**
> - Be logged in to your wordpress.org account in the same browser.
> - Have the zip ready: `wordpress-plugin/dist/crescent-visibility-0.5.2.zip`.
> - In `readme.txt`, `Contributors:` should already be your wp.org username and
>   `Tested up to:` should match a WP version you actually tested. Fix those first
>   if not — the reviewer checks them.
> - The directory derives the **slug** from the plugin name automatically; you
>   cannot choose it. Expect something like `young-crescent-moon-visibility`.

---

## PROMPT TO PASTE

```
You are helping me submit a plugin to the WordPress.org plugin directory. Work
carefully and DO NOT click the final submit button — stop and let me review first.

Steps:

1. Go to https://wordpress.org/plugins/developers/add/
   - If it redirects to a login page, tell me and stop (I'll log in).
   - If it says I already have a pending submission, stop and report that — do not
     start a second one.

2. The page is the "Add Your Plugin" upload form. There is usually just:
   - a file picker to upload a plugin ZIP, and
   - a checkbox agreeing to the plugin guidelines / detailed plugin guidelines.

   Upload this file in the file picker:
   /Users/yaaqov/github/crescent-moon-visibility/wordpress-plugin/dist/crescent-visibility-0.5.2.zip
   (If you can't access the local filesystem, tell me exactly which button to click
   and I'll attach the file myself.)

3. Tick the guidelines-agreement checkbox.

4. If the form also asks for free-text fields (some flows do, some don't), fill
   them from these canonical values — match whatever fields actually appear:

   - Plugin Name:
     Young Crescent Moon Visibility

   - Short description (one sentence):
     Show pre-computed young-crescent-moon visibility (Yallop criterion) for
     cities and dates, with a live atmospheric-conditions adjuster — no runtime
     astronomy.

   - Longer description (if a textarea is present):
     Young Crescent Moon Visibility adds an interactive tool that estimates how
     likely the thin new crescent moon is to be seen from a city on each of the
     first three evenings after conjunction. All astronomy is computed offline
     with the project's reference Yallop renderer and imported as a compact
     dataset, so the plugin does no runtime calculation and runs on ordinary
     shared hosting. The whole dataset for the page is embedded inline, so the
     tool also works on sites where /wp-admin is locked down. It bundles Leaflet
     locally (no external script) and loads only OpenStreetMap map tiles at
     runtime, which is disclosed in the readme's "External services" section.

5. After everything is filled and the file is attached, DO NOT submit. Instead:
   - Take a screenshot / describe the final state of the form.
   - List every field you filled and the value used.
   - Point out anything you were unsure about or left blank.
   - Then wait for me to confirm before clicking the final "Submit"/"Upload" button.

Important constraints:
- Never create a wordpress.org account, change account settings, or submit any
  OTHER form.
- If anything is ambiguous, stop and ask rather than guessing.
- Do not retry a submit if one appears to fail — report the error to me verbatim.
```

---

## After it's submitted (manual, by you)

1. You'll get an automated email that the plugin is **pending manual review**.
   Reviews currently take days-to-weeks; they may email back requesting changes.
2. When approved you receive SVN access at
   `https://plugins.svn.wordpress.org/<slug>/`.
3. Commit the plugin code to `/trunk/`, tag it under `/tags/0.5.2/`, and put the
   icon/banner/screenshot files in `/assets/` (see `wporg-assets-spec.md`).
4. Reply to any reviewer feedback from the same email thread; don't re-submit the
   form.

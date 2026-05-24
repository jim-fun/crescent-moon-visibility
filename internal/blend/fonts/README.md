# Legend Fonts

This directory holds TTF fonts used for smooth text rendering in the final blended maps.

## Recommended Font (Google Fonts)

We use **Inter** for clean, modern, highly legible text at small sizes:

1. Visit: https://fonts.google.com/specimen/Inter
2. Click **"Download family"**
3. In the zip, navigate to:
   `Inter/static/Inter-Regular.ttf`
4. Copy that file into this directory and **rename it exactly** to:
   `Inter-Regular.ttf`

After placing the file, rebuild:

```bash
go build -o bin/crescent_maps .
```

The legend will now render with proper anti-aliased/smooth text instead of the old pixelated `basicfont`.

Only the Regular weight is required for now.

# Phase 1 – Ollama Prompt (First Draft Generator)

**Use this with Ollama first** (cheapest resource) to get exploration volume.

---

**Prompt:**

You are helping build an offline data generator for a minimal WordPress plugin about crescent moon visibility.

Context:
- We will pre-compute accurate visibility data using the existing crescent-moon-visibility project for a fixed list of ~10 major cities.
- Data will cover from 2006 to ~2030.
- The generator must use the project's existing high-accuracy code (jobspec for new moons + calling the visibility renderer in "point" mode).
- Output must be clean, structured data suitable for importing into a WordPress database.
- The plugin itself will never run the renderer — this generator runs only on developer machines.

Requirements for the first draft:
- Write a Go program (can be a single file for now).
- Accept command-line flags for:
  - List of cities (or use a hardcoded default list)
  - Start year and end year
  - Output file path
- For each year, get the list of new moon dates.
- For each new moon + each city, simulate calling the renderer in point mode for the following 3 days and record the visibility categories.
- Produce output in a clean JSON format (one top-level array of records is fine).
- Include basic progress output to the console.
- Keep it simple and working — we will refactor later with Claude.

Also suggest 2-3 improvements or alternative approaches you considered while writing the first draft.

---

**After Ollama responds**, we will feed the best parts into a stronger Claude prompt for the production version.

**Tip**: Run this prompt multiple times with slight variations if you want different ideas before moving to Claude.
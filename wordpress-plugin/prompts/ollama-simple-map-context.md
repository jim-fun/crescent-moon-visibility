# Ollama Prompt: Simple Map / Location Context for the Interactive Form

**Context**:
The original web app had a Leaflet map for location context (draggable marker, presets, "use my location").

For the minimal-footprint WordPress plugin using precomputed data for 13 specific cities, we probably don't want the full complexity and external dependency of Leaflet for v1, especially since accuracy is best only for the precomputed cities.

**Task**:
Design a lightweight way to give users good location context in the interactive form without adding heavy dependencies.

Options to explore and prototype:
- A simple static map image (or SVG) with markers for the 13 cities.
- A clean, accessible list or grid of the supported cities with their names and a small visual indicator.
- Very lightweight Leaflet usage (only if the user selects "custom location" and we warn about accuracy).
- Or a hybrid: nice city cards/buttons for the precomputed cities, plus an optional "enter custom lat/lon" with a note.

Provide:
- Recommended approach for v1 (balancing minimal footprint, usability, and accuracy expectations).
- Sample HTML + CSS (Tailwind via CDN is fine).
- Any small JavaScript needed for interactions (e.g., selecting a city from the visual list updates the form).
- Notes on accessibility and mobile.

Keep it lightweight and pragmatic.
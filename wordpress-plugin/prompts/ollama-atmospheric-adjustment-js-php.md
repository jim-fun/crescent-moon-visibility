# Ollama Prompt: Atmospheric Adjustment on Precomputed Data (JS + PHP)

**Context**:
The WordPress plugin stores precomputed raw daily visibility categories (raw_day_0, raw_day_1, raw_day_2) for each new moon + city.

We have a PHP version of the atmospheric heuristic already ported in the main plugin class:

```php
public function apply_atmospheric_adjustment($raw_category, $cloud_percent, $transparency)
```

We need both:
- A PHP version (for server-side rendering or AJAX)
- A JavaScript version (for live client-side updates when the user moves the sliders, to feel responsive like the original web app)

The heuristic logic (from the original project) is roughly:
- Map A=5 ... E=1
- Cloud penalty: >80% → -3, >60% → -2, >40% → -1
- Transparency bonus/penalty: >=9 → +1, <=4 → -1, <=2 → -2
- Clamp between 1-5 and map back to letter

**Task**:
Provide clean, well-commented implementations of the atmospheric adjustment function in both PHP and JavaScript.

Also provide:
- Example usage in a form context (how to call it when sliders change and update the displayed Effective categories + note).
- Notes on performance (this is tiny, so no issue).
- How to handle the three daily values and pick the "best" after adjustment.

Output both versions clearly labeled.
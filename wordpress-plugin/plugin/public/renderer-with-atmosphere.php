<?php
if (!defined('ABSPATH')) exit;

/**
 * Advanced renderer that supports atmospheric adjustment on precomputed raw data.
 * This is a stepping stone toward full web-app parity.
 */
function crescent_visibility_render($atts) {
    global $wpdb;

    $city = sanitize_text_field($atts['city'] ?? 'jerusalem');
    $years = sanitize_text_field($atts['years'] ?? '2026-2028');
    $cloud = intval($atts['cloud_cover'] ?? 0);
    $trans = floatval($atts['transparency'] ?? 7);

    list($start_year, $end_year) = array_map('intval', explode('-', $years));

    $table = $wpdb->prefix . 'crescent_observations';

    $results = $wpdb->get_results($wpdb->prepare(
        "SELECT * FROM $table 
         WHERE city = %s AND year BETWEEN %d AND %d 
         ORDER BY new_moon_date ASC",
        $city, $start_year, $end_year
    ));

    ob_start();
    ?>
    <div class="crescent-visibility" style="font-family: system-ui, -apple-system, sans-serif; max-width: 900px; margin: 20px 0;">
        <h3 style="margin-bottom: 8px;">
            Crescent Visibility — <?php echo esc_html(ucwords(str_replace('-', ' ', $city))); ?>
        </h3>
        <p style="margin: 0 0 12px; color: #555;">
            <strong>Period:</strong> <?php echo esc_html($years); ?> 
            (<?php echo count($results); ?> new moons)
        </p>

        <?php if (empty($results)): ?>
            <div style="padding: 15px; background: #fff3cd; border: 1px solid #ffeaa7; border-radius: 6px;">
                <p style="margin: 0; color: #856404;">
                    <strong>No data found for this city and period.</strong><br><br>
                    Please go to <strong>Tools → Crescent Visibility</strong> and import a data file first.
                </p>
            </div>
        <?php else: ?>
            <?php 
            // Get the plugin instance to call the adjustment method if available
            $plugin = $GLOBALS['cv_plugin'] ?? null;
            ?>
            <div style="background: #f8f9fa; padding: 16px; border-radius: 8px; border: 1px solid #dee2e6;">
                <?php foreach ($results as $row): 
                    $raw_days = [$row->raw_day_0, $row->raw_day_1, $row->raw_day_2];
                    $adjusted = [];
                    
                    if ($plugin && method_exists($plugin, 'apply_atmospheric_adjustment')) {
                        foreach ($raw_days as $raw) {
                            list($eff, $note) = $plugin->apply_atmospheric_adjustment($raw, $cloud, $trans);
                            $adjusted[] = $eff;
                        }
                    } else {
                        $adjusted = $raw_days; // fallback if method not available yet
                    }
                    
                    $best_adjusted = min($adjusted); // A is best
                ?>
                    <div style="background: white; margin-bottom: 12px; padding: 14px; border-radius: 6px; border: 1px solid #dee2e6;">
                        <div style="font-size: 13px; color: #6c757d;"><?php echo esc_html($row->new_moon_date); ?></div>
                        
                        <div style="display: flex; gap: 20px; margin: 8px 0;">
                            <div>
                                <div style="font-size: 11px; color: #6c757d;">Raw</div>
                                <div style="font-size: 18px; font-family: monospace; font-weight: 600;">
                                    <?php echo esc_html(implode(' / ', $raw_days)); ?>
                                </div>
                            </div>
                            <div>
                                <div style="font-size: 11px; color: #6c757d;">Effective (adjusted)</div>
                                <div style="font-size: 18px; font-weight: 700; color: #0d6efd;">
                                    <?php echo esc_html(implode(' / ', $adjusted)); ?>
                                </div>
                            </div>
                        </div>
                        
                        <div style="font-size: 13px; background: #f8f9fa; padding: 6px 8px; border-radius: 4px;">
                            Best after adjustment: <strong><?php echo esc_html($best_adjusted); ?></strong>
                        </div>
                    </div>
                <?php endforeach; ?>
            </div>
        <?php endif; ?>
    </div>
    <?php
    return ob_get_clean();
}

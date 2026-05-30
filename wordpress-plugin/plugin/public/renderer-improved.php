<?php
if (!defined('ABSPATH')) exit;

/**
 * Improved renderer that supports atmospheric adjustment on top of precomputed raw data.
 * This is a stepping stone toward the full interactive web-app-like experience.
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
                    Please go to <strong>Tools → Crescent Visibility</strong> and import a data file.
                </p>
            </div>
        <?php else: ?>
            <?php foreach ($results as $row): 
                // Re-apply atmospheric adjustment using stored raw daily values
                $plugin = new Crescent_Visibility_Plugin(); // quick access to the method
                list($eff0, $note0) = $plugin->apply_atmospheric_adjustment($row->raw_day_0, $cloud, $trans);
                list($eff1, $note1) = $plugin->apply_atmospheric_adjustment($row->raw_day_1, $cloud, $trans);
                list($eff2, $note2) = $plugin->apply_atmospheric_adjustment($row->raw_day_2, $cloud, $trans);

                $adjusted_best = max($eff0, $eff1, $eff2); // simplistic "best" after adjustment
            ?>
                <div style="background: #fff; border: 1px solid #dee2e6; border-radius: 8px; padding: 14px; margin-bottom: 12px;">
                    <div style="font-size: 13px; color: #6c757d;"><?php echo esc_html($row->new_moon_date); ?></div>
                    
                    <div style="display: flex; gap: 20px; margin: 8px 0;">
                        <div>
                            <div style="font-size: 11px; color: #6c757d;">Raw</div>
                            <div style="font-size: 20px; font-family: monospace; font-weight: 600;">
                                <?php echo esc_html($row->raw_day_0 . ' / ' . $row->raw_day_1 . ' / ' . $row->raw_day_2); ?>
                            </div>
                        </div>
                        <div>
                            <div style="font-size: 11px; color: #6c757d;">Effective (adjusted)</div>
                            <div style="font-size: 20px; font-weight: 700; color: #0d6efd;">
                                <?php echo esc_html($eff0 . ' / ' . $eff1 . ' / ' . $eff2); ?>
                            </div>
                        </div>
                    </div>

                    <div style="font-size: 13px; background: #f8f9fa; padding: 6px 8px; border-radius: 4px;">
                        <?php echo esc_html($note0); ?>
                    </div>
                </div>
            <?php endforeach; ?>
        <?php endif; ?>
    </div>
    <?php
    return ob_get_clean();
}

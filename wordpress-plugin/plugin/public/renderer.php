<?php
if (!defined('ABSPATH')) exit;

/**
 * Renders the [crescent_visibility] shortcode.
 * This version works with both real WordPress and our local test harness.
 */
function crescent_visibility_render($atts) {
    global $wpdb;

    $city = sanitize_text_field($atts['city'] ?? 'jerusalem');
    $years = sanitize_text_field($atts['years'] ?? '2026-2028');

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
            <?php
            /* translators: %s: city name */
            echo esc_html(sprintf(__('Crescent Visibility — %s', 'crescent-visibility'), ucwords(str_replace('-', ' ', $city))));
            ?>
        </h3>
        <p style="margin: 0 0 12px; color: #555;">
            <strong><?php esc_html_e('Period:', 'crescent-visibility'); ?></strong> <?php echo esc_html($years); ?>
            <?php
            /* translators: %d: number of new moons */
            echo esc_html(sprintf(_n('(%d new moon)', '(%d new moons)', count($results), 'crescent-visibility'), count($results)));
            ?>
        </p>

        <?php if (empty($results)): ?>
            <div style="padding: 15px; background: #fff3cd; border: 1px solid #ffeaa7; border-radius: 6px;">
                <p style="margin: 0; color: #856404;">
                    <strong><?php esc_html_e('No data found for this city and period.', 'crescent-visibility'); ?></strong><br><br>
                    <strong><?php esc_html_e('Next step:', 'crescent-visibility'); ?></strong> <?php esc_html_e('Go to Tools → Crescent Visibility in the WordPress admin dashboard and import a visibility JSON file using the Import form.', 'crescent-visibility'); ?>
                </p>
            </div>
        <?php else: ?>
            <?php foreach ($results as $row): ?>
                <div style="background: #fff; border: 1px solid #dee2e6; border-radius: 8px; padding: 16px; margin-bottom: 12px;">
                    <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 10px;">
                        <div>
                            <div style="font-size: 13px; color: #6c757d;"><?php echo esc_html($row->new_moon_date); ?></div>
                            <div style="font-size: 32px; font-weight: 700; line-height: 1; color: #0d6efd;">
                                <?php echo esc_html($row->best_effective); ?>
                            </div>
                        </div>
                        <div style="text-align: right; font-size: 13px; line-height: 1.3;">
                            <div>Raw: <strong style="font-family: monospace;"><?php echo esc_html($row->best_raw); ?></strong></div>
                            <div style="color: #6c757d; font-size: 12px;">
                                Daily: <?php echo esc_html($row->raw_day_0 . ' • ' . $row->raw_day_1 . ' • ' . $row->raw_day_2); ?>
                            </div>
                        </div>
                    </div>
                    <div style="font-size: 13px; color: #495057; background: #f8f9fa; padding: 8px 10px; border-radius: 4px;">
                        <?php 
                        // Simple explanation based on best effective
                        $explanations = [
                            'A' => __('Excellent conditions — crescent should be easily visible to the naked eye.', 'crescent-visibility'),
                            'B' => __('Good conditions — visible to the naked eye under clear skies.', 'crescent-visibility'),
                            'C' => __('Moderate — visible naked eye but requires good conditions.', 'crescent-visibility'),
                            'D' => __('Difficult — usually needs binoculars or a telescope.', 'crescent-visibility'),
                            'E' => __('Very difficult or not visible even with aid.', 'crescent-visibility'),
                        ];
                        echo esc_html($explanations[$row->best_effective] ?? __('Visibility conditions vary.', 'crescent-visibility'));
                        ?>
                    </div>
                </div>
            <?php endforeach; ?>
        <?php endif; ?>

        <p style="font-size: 12px; color: #6c757d; margin-top: 12px;">
            <?php esc_html_e('Pre-computed Yallop data (clear skies baseline). Atmospheric adjustment coming in a future update.', 'crescent-visibility'); ?>
        </p>
    </div>
    <?php
    return ob_get_clean();
}

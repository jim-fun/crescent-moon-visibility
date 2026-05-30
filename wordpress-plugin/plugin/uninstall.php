<?php
/**
 * Uninstall cleanup for "Young Crescent Moon Visibility".
 *
 * WordPress runs this automatically when the plugin is deleted from the admin.
 * Removes the plugin's tables and options so nothing is left behind.
 */

// Only ever run in WordPress's uninstall context.
if (!defined('WP_UNINSTALL_PLUGIN')) {
    exit;
}

global $wpdb;

// Drop the plugin's tables (IF EXISTS — safe if a table was never created).
$wpdb->query("DROP TABLE IF EXISTS {$wpdb->prefix}crescent_observations");
$wpdb->query("DROP TABLE IF EXISTS {$wpdb->prefix}crescent_cities");
$wpdb->query("DROP TABLE IF EXISTS {$wpdb->prefix}crescent_yearly_summary");

// Remove the plugin's options.
delete_option('cvi_last_import');
delete_option('cvi_schema_version');
delete_option('cvi_plugin_version');

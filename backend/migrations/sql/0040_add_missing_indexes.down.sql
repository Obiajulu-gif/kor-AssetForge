-- Rollback 0040: Drop missing indexes on foreign key columns (#308)

DROP INDEX IF EXISTS idx_referrals_referee_id;
DROP INDEX IF EXISTS idx_referral_codes_user_id;

DROP INDEX IF EXISTS idx_asset_fraction_limits_asset_id;
DROP INDEX IF EXISTS idx_fractionalization_configs_updated_by;
DROP INDEX IF EXISTS idx_fractionalization_configs_created_by;

DROP INDEX IF EXISTS idx_fee_distribution_runs_rule_id;

DROP INDEX IF EXISTS idx_device_sessions_revoked_by;

DROP INDEX IF EXISTS idx_multi_sig_signatures_signer_id;
DROP INDEX IF EXISTS idx_multi_sig_signatures_proposal_id;
DROP INDEX IF EXISTS idx_multi_sig_proposals_proposer_id;
DROP INDEX IF EXISTS idx_multi_sig_wallets_created_by_id;

DROP INDEX IF EXISTS idx_approval_actions_delegated_from_id;
DROP INDEX IF EXISTS idx_approval_actions_approver_user_id;
DROP INDEX IF EXISTS idx_approval_requests_asset_id;
DROP INDEX IF EXISTS idx_approval_requests_transaction_id;
DROP INDEX IF EXISTS idx_approval_requests_workflow_id;
DROP INDEX IF EXISTS idx_approval_steps_approver_user_id;
DROP INDEX IF EXISTS idx_approval_steps_workflow_id;

DROP INDEX IF EXISTS idx_tax_exports_tax_report_id;
DROP INDEX IF EXISTS idx_tax_withholdings_transaction_id;
DROP INDEX IF EXISTS idx_tax_1099_forms_tax_report_id;
DROP INDEX IF EXISTS idx_tax_records_asset_id;
DROP INDEX IF EXISTS idx_tax_records_transaction_id;

DROP INDEX IF EXISTS idx_market_maker_inventory_asset_id;
DROP INDEX IF EXISTS idx_market_maker_trades_order_id;
DROP INDEX IF EXISTS idx_market_maker_orders_asset_id;
DROP INDEX IF EXISTS idx_market_maker_bots_managed_asset_id;

DROP INDEX IF EXISTS idx_user_consents_document_id;

DROP INDEX IF EXISTS idx_notification_preferences_user_id;

DROP INDEX IF EXISTS idx_pool_swaps_output_asset_id;
DROP INDEX IF EXISTS idx_pool_swaps_input_asset_id;

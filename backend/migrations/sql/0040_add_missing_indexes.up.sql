-- Migration 0040: Add missing indexes on foreign key columns (#308)
-- Prevents full table scans during JOINs and foreign key constraint validation.

-- Liquidity pool swaps (0009)
CREATE INDEX IF NOT EXISTS idx_pool_swaps_input_asset_id ON pool_swaps (input_asset_id);
CREATE INDEX IF NOT EXISTS idx_pool_swaps_output_asset_id ON pool_swaps (output_asset_id);

-- Notification preferences (0011)
CREATE INDEX IF NOT EXISTS idx_notification_preferences_user_id ON notification_preferences (user_id);

-- User consents (0012)
CREATE INDEX IF NOT EXISTS idx_user_consents_document_id ON user_consents (document_id);

-- Market maker tables (0014)
CREATE INDEX IF NOT EXISTS idx_market_maker_bots_managed_asset_id ON market_maker_bots (managed_asset_id);
CREATE INDEX IF NOT EXISTS idx_market_maker_orders_asset_id ON market_maker_orders (asset_id);
CREATE INDEX IF NOT EXISTS idx_market_maker_trades_order_id ON market_maker_trades (order_id);
CREATE INDEX IF NOT EXISTS idx_market_maker_inventory_asset_id ON market_maker_inventory (asset_id);

-- Tax tables (0015)
CREATE INDEX IF NOT EXISTS idx_tax_records_transaction_id ON tax_records (transaction_id);
CREATE INDEX IF NOT EXISTS idx_tax_records_asset_id ON tax_records (asset_id);
CREATE INDEX IF NOT EXISTS idx_tax_1099_forms_tax_report_id ON tax_1099_forms (tax_report_id);
CREATE INDEX IF NOT EXISTS idx_tax_withholdings_transaction_id ON tax_withholdings (transaction_id);
CREATE INDEX IF NOT EXISTS idx_tax_exports_tax_report_id ON tax_exports (tax_report_id);

-- Approval workflow tables (0018)
CREATE INDEX IF NOT EXISTS idx_approval_steps_workflow_id ON approval_steps (workflow_id);
CREATE INDEX IF NOT EXISTS idx_approval_steps_approver_user_id ON approval_steps (approver_user_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_workflow_id ON approval_requests (workflow_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_transaction_id ON approval_requests (transaction_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_asset_id ON approval_requests (asset_id);
CREATE INDEX IF NOT EXISTS idx_approval_actions_approver_user_id ON approval_actions (approver_user_id);
CREATE INDEX IF NOT EXISTS idx_approval_actions_delegated_from_id ON approval_actions (delegated_from_id);

-- Multi-sig tables (0022)
CREATE INDEX IF NOT EXISTS idx_multi_sig_wallets_created_by_id ON multi_sig_wallets (created_by_id);
CREATE INDEX IF NOT EXISTS idx_multi_sig_proposals_proposer_id ON multi_sig_proposals (proposer_id);
CREATE INDEX IF NOT EXISTS idx_multi_sig_signatures_proposal_id ON multi_sig_signatures (proposal_id);
CREATE INDEX IF NOT EXISTS idx_multi_sig_signatures_signer_id ON multi_sig_signatures (signer_id);

-- Enhanced device sessions (0023)
CREATE INDEX IF NOT EXISTS idx_device_sessions_revoked_by ON device_sessions (revoked_by);

-- Fee distribution tables (0027)
CREATE INDEX IF NOT EXISTS idx_fee_distribution_runs_rule_id ON fee_distribution_runs (rule_id);

-- Fractionalization tables (0031)
CREATE INDEX IF NOT EXISTS idx_fractionalization_configs_created_by ON fractionalization_configs (created_by);
CREATE INDEX IF NOT EXISTS idx_fractionalization_configs_updated_by ON fractionalization_configs (updated_by);
CREATE INDEX IF NOT EXISTS idx_asset_fraction_limits_asset_id ON asset_fraction_limits (asset_id);

-- Referral tables (0037)
CREATE INDEX IF NOT EXISTS idx_referral_codes_user_id ON referral_codes (user_id);
CREATE INDEX IF NOT EXISTS idx_referrals_referee_id ON referrals (referee_id);

# Database Schema & Indexing Guide

This document documents the relational schema and indexing strategy for **kor-AssetForge**, including foreign key indexing, composite indexes, and partial indexes.

- **Related Documentation:**
  - [`DATABASE_TUNING.md`](./DATABASE_TUNING.md) — Connection pooling and performance tuning
  - [`MIGRATIONS.md`](./MIGRATIONS.md) — Migration creation, rollback, and lifecycle
  - [`../scripts/analyze_indexes.sh`](../scripts/analyze_indexes.sh) — Foreign key index validation script

---

## 1. Indexing Principles

1. **All Foreign Keys Must Be Indexed:**
   Unindexed foreign keys cause PostgreSQL to perform sequential scans on the referencing table during `DELETE` or `UPDATE` cascades on referenced rows, and force hash/merge joins instead of index scans during relational queries.
2. **Partial Unique Indexes for Soft Deletes:**
   Tables with soft-deletion (`deleted_at TIMESTAMPTZ`) use partial unique indexes with `WHERE deleted_at IS NULL` instead of unconditional table constraints, allowing users to recreate records or re-register after account closure.
3. **Leading Column Ordering in Composite Indexes:**
   Composite indexes place high-cardinality equality filter columns first, followed by range or sort columns (`ORDER BY created_at DESC`).
4. **Naming Convention:**
   - Standard B-tree index: `idx_<table_name>_<column_name>`
   - Unique index: `uq_<table_name>_<column_name>`

---

## 2. Foreign Key Indexes

The following table documents all foreign key columns across the schema and their corresponding performance indexes (including those added in migration `0040_add_missing_indexes`):

| Table | Foreign Key Column | Referenced Table & Column | Index Name |
|-------|--------------------|---------------------------|------------|
| `user_balances` | `user_id` | `users(id)` | `idx_user_balances_user_id` |
| `user_balances` | `asset_id` | `assets(id)` | `idx_user_balances_asset_id` |
| `user_sessions` | `user_id` | `users(id)` | `idx_user_sessions_user_id` |
| `kyc_records` | `user_id` | `users(id)` | `idx_kyc_records_user_id` |
| `kyc_documents` | `kyc_record_id` | `kyc_records(id)` | `idx_kyc_documents_record_id` |
| `aml_screenings` | `kyc_record_id` | `kyc_records(id)` | `idx_aml_screenings_record_id` |
| `compliance_audit_logs` | `user_id` | `users(id)` | `idx_compliance_logs_user_id` |
| `disputes` | `transaction_id` | `transactions(id)` | `idx_disputes_transaction_id` |
| `dispute_evidences` | `dispute_id` | `disputes(id)` | `idx_dispute_evidences_dispute_id` |
| `dispute_evidences` | `asset_id` | `assets(id)` | `idx_dispute_evidences_asset_id` |
| `p2p_orders` | `asset_id` | `assets(id)` | `idx_p2p_orders_asset_id` |
| `p2p_trades` | `asset_id` | `assets(id)` | `idx_p2p_trades_asset_id` |
| `p2p_trades` | `buy_order_id` | `p2p_orders(id)` | `idx_p2p_trades_buy_order_id` |
| `p2p_trades` | `sell_order_id` | `p2p_orders(id)` | `idx_p2p_trades_sell_order_id` |
| `p2p_order_books` | `asset_id` | `assets(id)` | `idx_p2p_order_books_asset_id` |
| `stake_positions` | `user_id` | `users(id)` | `idx_stake_positions_user_id` |
| `stake_positions` | `asset_id` | `assets(id)` | `idx_stake_positions_asset_id` |
| `stake_pools` | `asset_id` | `assets(id)` | `idx_stake_pools_asset_id` |
| `stake_rewards` | `stake_id` | `stake_positions(id)` | `idx_stake_rewards_stake_id` |
| `stake_rewards` | `user_id` | `users(id)` | `idx_stake_rewards_user_id` |
| `stake_rewards` | `asset_id` | `assets(id)` | `idx_stake_rewards_asset_id` |
| `liquidity_pools` | `asset_a_id` | `assets(id)` | `idx_liquidity_pools_asset_a_id` |
| `liquidity_pools` | `asset_b_id` | `assets(id)` | `idx_liquidity_pools_asset_b_id` |
| `liquidity_positions` | `pool_id` | `liquidity_pools(id)` | `idx_liquidity_positions_pool_id` |
| `pool_swaps` | `pool_id` | `liquidity_pools(id)` | `idx_pool_swaps_pool_id` |
| `pool_swaps` | `input_asset_id` | `assets(id)` | `idx_pool_swaps_input_asset_id` |
| `pool_swaps` | `output_asset_id` | `assets(id)` | `idx_pool_swaps_output_asset_id` |
| `webhook_subscriptions` | `user_id` | `users(id)` | `idx_webhook_subscriptions_user_id` |
| `webhook_delivery_logs` | `subscription_id` | `webhook_subscriptions(id)` | `idx_webhook_delivery_logs_sub_id` |
| `notifications` | `user_id` | `users(id)` | `idx_notifications_user_id` |
| `notification_preferences` | `user_id` | `users(id)` | `idx_notification_preferences_user_id` |
| `user_consents` | `user_id` | `users(id)` | `idx_user_consents_user_id` |
| `user_consents` | `document_id` | `legal_documents(id)` | `idx_user_consents_document_id` |
| `data_export_requests` | `user_id` | `users(id)` | `idx_data_export_requests_user_id` |
| `market_maker_bots` | `managed_asset_id` | `assets(id)` | `idx_market_maker_bots_managed_asset_id` |
| `market_maker_orders` | `bot_id` | `market_maker_bots(id)` | `idx_market_maker_orders_bot_id` |
| `market_maker_orders` | `asset_id` | `assets(id)` | `idx_market_maker_orders_asset_id` |
| `market_maker_trades` | `bot_id` | `market_maker_bots(id)` | `idx_market_maker_trades_bot_id` |
| `market_maker_trades` | `order_id` | `market_maker_orders(id)` | `idx_market_maker_trades_order_id` |
| `market_maker_inventory` | `bot_id` | `market_maker_bots(id)` | `idx_market_maker_inventory_bot_id` |
| `market_maker_inventory` | `asset_id` | `assets(id)` | `idx_market_maker_inventory_asset_id` |
| `tax_records` | `user_id` | `users(id)` | `idx_tax_records_user_id` |
| `tax_records` | `transaction_id` | `transactions(id)` | `idx_tax_records_transaction_id` |
| `tax_records` | `asset_id` | `assets(id)` | `idx_tax_records_asset_id` |
| `tax_1099_forms` | `tax_report_id` | `tax_reports(id)` | `idx_tax_1099_forms_tax_report_id` |
| `tax_withholdings` | `transaction_id` | `transactions(id)` | `idx_tax_withholdings_transaction_id` |
| `tax_exports` | `tax_report_id` | `tax_reports(id)` | `idx_tax_exports_tax_report_id` |
| `approval_steps` | `workflow_id` | `approval_workflows(id)` | `idx_approval_steps_workflow_id` |
| `approval_steps` | `approver_user_id` | `users(id)` | `idx_approval_steps_approver_user_id` |
| `approval_requests` | `workflow_id` | `approval_workflows(id)` | `idx_approval_requests_workflow_id` |
| `approval_requests` | `transaction_id` | `transactions(id)` | `idx_approval_requests_transaction_id` |
| `approval_requests` | `asset_id` | `assets(id)` | `idx_approval_requests_asset_id` |
| `approval_actions` | `approval_request_id` | `approval_requests(id)` | `idx_approval_actions_request_step` |
| `approval_actions` | `approver_user_id` | `users(id)` | `idx_approval_actions_approver_user_id` |
| `approval_actions` | `delegated_from_id` | `users(id)` | `idx_approval_actions_delegated_from_id` |
| `multi_sig_wallets` | `created_by_id` | `users(id)` | `idx_multi_sig_wallets_created_by_id` |
| `multi_sig_proposals` | `wallet_id` | `multi_sig_wallets(id)` | `idx_multi_sig_proposals_wallet_id` |
| `multi_sig_proposals` | `proposer_id` | `users(id)` | `idx_multi_sig_proposals_proposer_id` |
| `multi_sig_signatures` | `proposal_id` | `multi_sig_proposals(id)` | `idx_multi_sig_signatures_proposal_id` |
| `multi_sig_signatures` | `signer_id` | `users(id)` | `idx_multi_sig_signatures_signer_id` |
| `device_sessions` | `user_id` | `users(id)` | `idx_device_sessions_user_id` |
| `device_sessions` | `revoked_by` | `users(id)` | `idx_device_sessions_revoked_by` |
| `fee_distribution_runs` | `rule_id` | `fee_distribution_rules(id)` | `idx_fee_distribution_runs_rule_id` |
| `fee_distribution_allocations` | `run_id` | `fee_distribution_runs(id)` | `idx_fee_distribution_allocations_run_id` |
| `fractionalization_configs` | `created_by` | `users(id)` | `idx_fractionalization_configs_created_by` |
| `fractionalization_configs` | `updated_by` | `users(id)` | `idx_fractionalization_configs_updated_by` |
| `asset_fraction_limits` | `asset_id` | `assets(id)` | `idx_asset_fraction_limits_asset_id` |
| `referral_codes` | `user_id` | `users(id)` | `idx_referral_codes_user_id` |
| `referrals` | `referrer_id` | `users(id)` | `idx_referrals_referrer` |
| `referrals` | `referee_id` | `users(id)` | `idx_referrals_referee_id` |
| `referral_rewards` | `referral_id` | `referrals(id)` | `idx_referral_rewards_referral` |
| `referral_rewards` | `user_id` | `users(id)` | `idx_referral_rewards_user` |
| `beneficial_owners` | `entity_user_id` | `users(id)` | `idx_beneficial_owners_entity_user_id` |
| `corporate_structure_documents` | `entity_user_id` | `users(id)` | `idx_corp_structure_docs_entity_user_id` |

---

## 3. Query Performance & EXPLAIN Validation

To verify that PostgreSQL query planning uses an index for a join rather than a sequential scan, use `EXPLAIN ANALYZE`:

```sql
EXPLAIN ANALYZE
SELECT s.id, s.amount_in, a.name AS input_asset_name
FROM pool_swaps s
JOIN assets a ON s.input_asset_id = a.id
WHERE s.created_at >= NOW() - INTERVAL '7 days';
```

**Expected Query Plan:**
```
Hash Join  (cost=... rows=... width=...)
  Hash Cond: (s.input_asset_id = a.id)
  ->  Index Scan using idx_pool_swaps_created_at on pool_swaps s
  ->  Hash
        ->  Seq Scan on assets a
```

If joining on a specific asset ID:
```sql
EXPLAIN ANALYZE
SELECT count(*)
FROM pool_swaps
WHERE input_asset_id = 42;
```
**Expected Plan:**
```
Bitmap Heap Scan on pool_swaps
  Recheck Cond: (input_asset_id = 42)
  ->  Bitmap Index Scan on idx_pool_swaps_input_asset_id
```

Run `./scripts/analyze_indexes.sh` in staging or production to continuously detect any missing foreign key indexes or unused indexes.

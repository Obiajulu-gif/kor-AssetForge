// Unit tests for the EmergencyControl pause / unpause and circuit-breaker
// subsystem (Issue #341).
//
// These tests live in a child module of `emergency_control` so they can reach
// private helpers through `super::*`. Scenario-level, cross-contract tests live
// in `contracts/tests/emergency_scenarios.rs`.
//
// Coverage:
//   - pause / unpause for every PauseScope variant
//   - scope independence and the `All` wildcard
//   - auto-unpause timing (scheduled, cleared by manual unpause, disabled)
//   - pause audit history (ordering, reason, admin)
//   - authorization failures
//   - circuit breaker: price crash, volume spike, cooldown unpause

use super::*;
use soroban_sdk::testutils::{Address as _, Ledger, LedgerInfo};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Deploys and initializes a fresh EmergencyControl contract. Returns the
/// environment, the contract address, and the admin. Callers build their own
/// client with `EmergencyControlClient::new(&env, &contract_id)`.
fn setup() -> (Env, Address, Address) {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let admin = Address::generate(&env);
    let client = EmergencyControlClient::new(&env, &contract_id);
    client.initialize(&admin);
    (env, contract_id, admin)
}

fn set_ledger_sequence(env: &Env, sequence: u32) {
    env.ledger().with_mut(|li| {
        li.sequence_number = sequence;
    });
}

fn set_timestamp(env: &Env, timestamp: u64) {
    env.ledger().with_mut(|li| {
        li.timestamp = timestamp;
    });
}

// ===========================================================================
// Initialization
// ===========================================================================

#[test]
fn test_initialize_and_get_admin() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);
    let stored_admin = client.get_admin();
    assert_eq!(stored_admin, admin);
}

#[test]
#[should_panic(expected = "already initialized")]
fn test_double_initialize_panics() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);
    client.initialize(&admin);
}

// ===========================================================================
// Pause / unpause basics
// ===========================================================================

#[test]
fn test_pause_and_unpause_asset() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "security incident");

    // Pause trading
    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &0);
    assert!(client.is_paused(&asset_id, &PauseScope::Trading));

    // Transfers should NOT be paused
    assert!(!client.is_paused(&asset_id, &PauseScope::Transfers));

    // Unpause trading
    client.unpause_asset(&admin, &asset_id, &PauseScope::Trading);
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_pause_all_blocks_everything() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "global halt");

    client.pause_asset(&admin, &asset_id, &PauseScope::All, &reason, &0);

    // All scopes should report as paused
    assert!(client.is_paused(&asset_id, &PauseScope::All));
    assert!(client.is_paused(&asset_id, &PauseScope::Transfers));
    assert!(client.is_paused(&asset_id, &PauseScope::Trading));
    assert!(client.is_paused(&asset_id, &PauseScope::Minting));
}

#[test]
fn test_pause_all_covers_function_scope() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "global halt");
    client.pause_asset(&admin, &asset_id, &PauseScope::All, &reason, &0);

    // A more granular Function scope is blocked by an All pause.
    let func = Symbol::new(&env, "mint");
    assert!(client.is_paused(&asset_id, &PauseScope::Function(func)));
}

#[test]
fn test_pause_history_recorded() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "audit");

    client.pause_asset(&admin, &asset_id, &PauseScope::Minting, &reason, &0);
    client.unpause_asset(&admin, &asset_id, &PauseScope::Minting);

    let history = client.get_pause_history(&asset_id);
    assert_eq!(history.len(), 2);

    // First record is the pause
    let first = history.get(0).unwrap();
    assert!(first.is_pause);
    assert_eq!(first.asset_id, asset_id);

    // Second record is the unpause
    let second = history.get(1).unwrap();
    assert!(!second.is_pause);
}

#[test]
#[should_panic(expected = "asset is already paused for this scope")]
fn test_double_pause_panics() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "test");

    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &0);
    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &0);
}

#[test]
#[should_panic(expected = "asset is not paused for this scope")]
fn test_unpause_not_paused_panics() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let asset_id: u64 = 1;
    client.unpause_asset(&admin, &asset_id, &PauseScope::Trading);
}

// ===========================================================================
// Authorization
// ===========================================================================

#[test]
#[should_panic(expected = "caller is not admin")]
fn test_non_admin_pause_panics() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let non_admin = Address::generate(&env);
    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "test");

    client.pause_asset(&non_admin, &asset_id, &PauseScope::Trading, &reason, &0);
}

#[test]
#[should_panic(expected = "caller is not admin")]
fn test_non_admin_unpause_panics() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "test");
    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &0);

    let non_admin = Address::generate(&env);
    client.unpause_asset(&non_admin, &asset_id, &PauseScope::Trading);
}

// ===========================================================================
// Enforcement
// ===========================================================================

#[test]
#[should_panic(expected = "operation blocked: asset is paused")]
fn test_require_not_paused_blocks_when_paused() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "blocked");

    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &0);
    client.require_not_paused(&asset_id, &PauseScope::Trading);
}

#[test]
fn test_require_not_paused_allows_when_not_paused() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let asset_id: u64 = 1;
    // Should not panic
    client.require_not_paused(&asset_id, &PauseScope::Trading);
}

#[test]
#[should_panic(expected = "operation blocked: asset is paused")]
fn test_require_not_paused_blocked_by_all_scope() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "global halt");
    client.pause_asset(&admin, &asset_id, &PauseScope::All, &reason, &0);

    // A narrower operation is blocked because `All` is paused.
    client.require_not_paused(&asset_id, &PauseScope::Minting);
}

// ===========================================================================
// Scope independence
// ===========================================================================

#[test]
fn test_selective_scope_independence() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "scope test");

    // Pause only transfers
    client.pause_asset(&admin, &asset_id, &PauseScope::Transfers, &reason, &0);

    // Transfers paused
    assert!(client.is_paused(&asset_id, &PauseScope::Transfers));

    // Trading and Minting should NOT be paused
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
    assert!(!client.is_paused(&asset_id, &PauseScope::Minting));
}

#[test]
fn test_multiple_scopes_same_asset_unpause_one_keeps_other() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "multi-scope");

    client.pause_asset(&admin, &asset_id, &PauseScope::Transfers, &reason, &0);
    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &0);

    // Unpausing one scope must not clear the other.
    client.unpause_asset(&admin, &asset_id, &PauseScope::Transfers);
    assert!(!client.is_paused(&asset_id, &PauseScope::Transfers));
    assert!(client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_pause_minting_scope_only() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "mint halt");

    client.pause_asset(&admin, &asset_id, &PauseScope::Minting, &reason, &0);
    assert!(client.is_paused(&asset_id, &PauseScope::Minting));
    assert!(!client.is_paused(&asset_id, &PauseScope::Transfers));
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_function_scope_pause_is_isolated() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "fn halt");
    let func = Symbol::new(&env, "trade");

    client.pause_asset(&admin, &asset_id, &PauseScope::Function(func.clone()), &reason, &0);
    assert!(client.is_paused(&asset_id, &PauseScope::Function(func)));
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_user_scope_pause_is_isolated() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "user halt");
    let user = Address::generate(&env);

    client.pause_asset(&admin, &asset_id, &PauseScope::User(user.clone()), &reason, &0);
    assert!(client.is_paused(&asset_id, &PauseScope::User(user)));
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_asset_scope_pause_is_isolated() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "asset halt");

    client.pause_asset(&admin, &asset_id, &PauseScope::Asset(asset_id), &reason, &0);
    assert!(client.is_paused(&asset_id, &PauseScope::Asset(asset_id)));
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_multiple_assets_independent() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let reason = String::from_str(&env, "asset isolation test");

    // Pause asset 1 for trading
    client.pause_asset(&admin, &1, &PauseScope::Trading, &reason, &0);

    // Asset 1 trading paused
    assert!(client.is_paused(&1, &PauseScope::Trading));

    // Asset 2 trading should NOT be paused
    assert!(!client.is_paused(&2, &PauseScope::Trading));
}

#[test]
fn test_empty_pause_history() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let history = client.get_pause_history(&1);
    assert_eq!(history.len(), 0);
    let _ = admin;
}

// ===========================================================================
// Audit history contents
// ===========================================================================

#[test]
fn test_pause_history_records_reason_and_admin() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 7;
    let reason = String::from_str(&env, "incident-42");

    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &0);

    let history = client.get_pause_history(&asset_id);
    assert_eq!(history.len(), 1);
    let entry = history.get(0).unwrap();
    assert!(entry.is_pause);
    assert_eq!(entry.asset_id, asset_id);
    assert_eq!(entry.admin, admin);
    assert_eq!(entry.reason, reason);
}

#[test]
fn test_pause_history_is_append_only_and_ordered() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 3;

    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &String::from_str(&env, "a"), &0);
    client.unpause_asset(&admin, &asset_id, &PauseScope::Trading);
    client.pause_asset(&admin, &asset_id, &PauseScope::Minting, &String::from_str(&env, "b"), &0);

    let history = client.get_pause_history(&asset_id);
    assert_eq!(history.len(), 3);
    assert!(history.get(0).unwrap().is_pause);
    assert!(!history.get(1).unwrap().is_pause);
    assert!(history.get(2).unwrap().is_pause);
}

// ===========================================================================
// Auto-unpause
// ===========================================================================

#[test]
fn test_auto_unpause() {
    let env = Env::default();
    env.mock_all_auths();
    let contract_id = env.register_contract(None, EmergencyControl);
    let client = EmergencyControlClient::new(&env, &contract_id);

    let admin = Address::generate(&env);
    client.initialize(&admin);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "temporary halt");

    // Set auto-unpause at ledger sequence 100
    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &100);
    assert!(client.is_paused(&asset_id, &PauseScope::Trading));

    // Advance ledger sequence to 100
    env.ledger().set(LedgerInfo {
        timestamp: 0,
        protocol_version: 20,
        sequence_number: 100,
        network_id: Default::default(),
        base_reserve: 10,
        min_temp_entry_ttl: 10,
        min_persistent_entry_ttl: 10,
        max_entry_ttl: 3110400,
    });

    // Should auto-unpause
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_no_auto_unpause_when_ledger_is_zero() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "manual only");

    // auto_unpause_ledger = 0 disables auto-unpause entirely.
    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &0);

    set_ledger_sequence(&env, 1_000_000);
    assert!(client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_auto_unpause_records_history_once() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "temporary");
    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &50);

    set_ledger_sequence(&env, 50);
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));

    // Repeated reads must not append further history entries.
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
    let history = client.get_pause_history(&asset_id);
    assert_eq!(history.len(), 2); // pause + auto-unpause
    let auto = history.get(1).unwrap();
    assert!(!auto.is_pause);
}

#[test]
fn test_manual_unpause_clears_auto_unpause_timer() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let reason = String::from_str(&env, "temporary");
    client.pause_asset(&admin, &asset_id, &PauseScope::Trading, &reason, &100);

    // Manually unpause before the scheduled auto-unpause.
    client.unpause_asset(&admin, &asset_id, &PauseScope::Trading);
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));

    // Crossing the original auto-unpause ledger must not fire again.
    set_ledger_sequence(&env, 100);
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));

    let history = client.get_pause_history(&asset_id);
    assert_eq!(history.len(), 2); // pause + manual unpause, no auto entry
}

// ===========================================================================
// Circuit breaker — price crash
// ===========================================================================

#[test]
fn test_circuit_breaker_price_crash_trips() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;

    // Baseline price at t=0.
    assert!(!client.record_price(&admin, &asset_id, &100));

    // Price drops >50% within the hour.
    set_timestamp(&env, 60);
    assert!(client.record_price(&admin, &asset_id, &40));

    assert!(client.is_paused(&asset_id, &PauseScope::Trading));
    let history = client.get_pause_history(&asset_id);
    assert_eq!(history.len(), 1);
    assert!(history.get(0).unwrap().is_pause);
}

#[test]
fn test_circuit_breaker_price_small_drop_does_not_trip() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    assert!(!client.record_price(&admin, &asset_id, &100));

    // 40% drop is below the >50% threshold.
    set_timestamp(&env, 60);
    assert!(!client.record_price(&admin, &asset_id, &60));
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_circuit_breaker_price_drop_after_cooldown_does_not_trip() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    assert!(!client.record_price(&admin, &asset_id, &100));

    // More than one hour elapsed: the crash window has passed.
    set_timestamp(&env, 4000);
    assert!(!client.record_price(&admin, &asset_id, &10));
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
#[should_panic(expected = "price must be positive")]
fn test_record_price_rejects_non_positive() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);
    client.record_price(&admin, &1, &0);
}

#[test]
#[should_panic(expected = "caller is not admin")]
fn test_record_price_requires_admin() {
    let (env, contract_id, _admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);
    let non_admin = Address::generate(&env);
    client.record_price(&non_admin, &1, &100);
}

// ===========================================================================
// Circuit breaker — volume spike
// ===========================================================================

#[test]
fn test_circuit_breaker_volume_spike_trips() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    assert!(client.record_volume(&admin, &asset_id, &2000, &1000));
    assert!(client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_circuit_breaker_volume_below_threshold_does_not_trip() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    assert!(!client.record_volume(&admin, &asset_id, &500, &1000));
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
#[should_panic(expected = "threshold must be positive")]
fn test_record_volume_rejects_non_positive_threshold() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);
    client.record_volume(&admin, &1, &100, &0);
}

// ===========================================================================
// Circuit breaker — cooldown unpause
// ===========================================================================

#[test]
fn test_circuit_breaker_unpause_clears_trading_pause() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    assert!(client.record_volume(&admin, &asset_id, &2000, &1000));
    assert!(client.is_paused(&asset_id, &PauseScope::Trading));

    client.circuit_breaker_unpause(&admin, &asset_id);
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

#[test]
fn test_circuit_breaker_auto_unpause_after_cooldown_ledgers() {
    let (env, contract_id, admin) = setup();
    let client = EmergencyControlClient::new(&env, &contract_id);

    let asset_id: u64 = 1;
    let start = env.ledger().sequence();
    assert!(client.record_volume(&admin, &asset_id, &2000, &1000));
    assert!(client.is_paused(&asset_id, &PauseScope::Trading));

    // The breaker sets a 720-ledger cooldown, so advancing past it auto-unpauses.
    set_ledger_sequence(&env, start + 720);
    assert!(!client.is_paused(&asset_id, &PauseScope::Trading));
}

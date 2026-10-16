// Scenario-level emergency pause tests (Issue #341).
//
// These integration tests exercise the EmergencyControl pause subsystem the way
// it is used in production: multiple contracts (EmergencyControl, AssetToken,
// Marketplace) coordinated across realistic emergency and recovery sequences.
// Granular unit coverage of every scope lives in
// `src/emergency_control/tests.rs`.
//
// Scenarios covered:
//   1. Full emergency drill: pause everything, verify enforcement, recover.
//   2. Trading-only pause blocks listings but nothing else.
//   3. Transfers-only pause blocks transfers but not listings.
//   4. Minting-only pause blocks minting.
//   5. Multi-asset isolation (one paused asset does not affect another).
//   6. Circuit-breaker trip and manual recovery.
//   7. Layered pauses: unpausing one scope does not lift a broader pause.

extern crate kor_assetforge_contracts;

use kor_assetforge_contracts::asset_token::{AssetToken, AssetTokenClient};
use kor_assetforge_contracts::emergency_control::{
    EmergencyControl, EmergencyControlClient, PauseScope,
};
use kor_assetforge_contracts::marketplace::{Marketplace, MarketplaceClient};
use soroban_sdk::testutils::Address as _;
use soroban_sdk::{Address, Env, String};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn setup() -> (Env, Address, Address, Address, Address) {
    let env = Env::default();
    env.mock_all_auths();

    let ec_id = env.register_contract(None, EmergencyControl);
    let mp_id = env.register_contract(None, Marketplace);
    let at_id = env.register_contract(None, AssetToken);

    let admin = Address::generate(&env);

    let ec_client = EmergencyControlClient::new(&env, &ec_id);
    ec_client.initialize(&admin);

    let at_client = AssetTokenClient::new(&env, &at_id);
    at_client.initialize(
        &admin,
        &String::from_str(&env, "Token"),
        &String::from_str(&env, "TKN"),
        &7,
        &0,
    );

    (env, ec_id, mp_id, at_id, admin)
}

/// Returns true if the closure panics (e.g. a paused operation was rejected).
fn panics<F: FnOnce()>(f: F) -> bool {
    std::panic::catch_unwind(std::panic::AssertUnwindSafe(f)).is_err()
}

// ===========================================================================
// 1. Full emergency drill and recovery
// ===========================================================================

#[test]
fn test_scenario_full_emergency_drill_and_recovery() {
    let (env, ec_id, mp_id, at_id, admin) = setup();
    let ec_client = EmergencyControlClient::new(&env, &ec_id);
    let mp_client = MarketplaceClient::new(&env, &mp_id);
    let at_client = AssetTokenClient::new(&env, &at_id);

    let asset_id: u64 = 1;
    let seller = Address::generate(&env);
    at_client.mint(&seller, &10_000, &asset_id, &ec_id);

    // Baseline: listings work.
    let first = mp_client.create_listing(&seller, &asset_id, &100, &1000, &ec_id, &None);
    assert_eq!(first, 1);

    // Emergency declared: pause every scope for the asset.
    let reason = String::from_str(&env, "emergency drill");
    ec_client.pause_asset(&admin, &asset_id, &PauseScope::All, &reason, &0);

    assert!(ec_client.is_paused(&asset_id, &PauseScope::All));
    assert!(ec_client.is_paused(&asset_id, &PauseScope::Trading));
    assert!(ec_client.is_paused(&asset_id, &PauseScope::Transfers));
    assert!(ec_client.is_paused(&asset_id, &PauseScope::Minting));

    // Every gated operation is blocked while paused.
    assert!(panics(|| {
        mp_client.create_listing(&seller, &asset_id, &100, &1000, &ec_id, &None);
    }));
    assert!(panics(|| {
        at_client.transfer(&seller, &Address::generate(&env), &10, &asset_id, &ec_id);
    }));
    assert!(panics(|| {
        at_client.mint(&seller, &10, &asset_id, &ec_id);
    }));

    // Recovery: lift the global pause.
    ec_client.unpause_asset(&admin, &asset_id, &PauseScope::All);
    assert!(!ec_client.is_paused(&asset_id, &PauseScope::All));

    // Operations resume.
    let second = mp_client.create_listing(&seller, &asset_id, &100, &1000, &ec_id, &None);
    assert_eq!(second, 2);

    // The audit trail records the full sequence: pause + unpause.
    let history = ec_client.get_pause_history(&asset_id);
    assert_eq!(history.len(), 2);
    assert!(history.get(0).unwrap().is_pause);
    assert!(!history.get(1).unwrap().is_pause);
}

// ===========================================================================
// 2. Trading-only pause
// ===========================================================================

#[test]
fn test_scenario_trading_pause_only_blocks_listings() {
    let (env, ec_id, mp_id, at_id, admin) = setup();
    let ec_client = EmergencyControlClient::new(&env, &ec_id);
    let mp_client = MarketplaceClient::new(&env, &mp_id);
    let at_client = AssetTokenClient::new(&env, &at_id);

    let asset_id: u64 = 1;
    let seller = Address::generate(&env);
    at_client.mint(&seller, &10_000, &asset_id, &ec_id);

    ec_client.pause_asset(
        &admin,
        &asset_id,
        &PauseScope::Trading,
        &String::from_str(&env, "market maintenance"),
        &0,
    );

    // Listings are blocked...
    assert!(panics(|| {
        mp_client.create_listing(&seller, &asset_id, &100, &1000, &ec_id, &None);
    }));

    // ...but a transfer (Transfers scope) is unaffected.
    at_client.transfer(&seller, &Address::generate(&env), &10, &asset_id, &ec_id);

    ec_client.unpause_asset(&admin, &asset_id, &PauseScope::Trading);
    let listing = mp_client.create_listing(&seller, &asset_id, &100, &1000, &ec_id, &None);
    assert_eq!(listing, 1);
}

// ===========================================================================
// 3. Transfers-only pause
// ===========================================================================

#[test]
fn test_scenario_transfers_pause_blocks_transfers_not_listings() {
    let (env, ec_id, mp_id, at_id, admin) = setup();
    let ec_client = EmergencyControlClient::new(&env, &ec_id);
    let mp_client = MarketplaceClient::new(&env, &mp_id);
    let at_client = AssetTokenClient::new(&env, &at_id);

    let asset_id: u64 = 1;
    let seller = Address::generate(&env);
    at_client.mint(&seller, &10_000, &asset_id, &ec_id);

    ec_client.pause_asset(
        &admin,
        &asset_id,
        &PauseScope::Transfers,
        &String::from_str(&env, "transfer investigation"),
        &0,
    );

    // Transfers are blocked...
    assert!(panics(|| {
        at_client.transfer(&seller, &Address::generate(&env), &10, &asset_id, &ec_id);
    }));

    // ...but listings (Trading scope) still succeed.
    let listing = mp_client.create_listing(&seller, &asset_id, &100, &1000, &ec_id, &None);
    assert_eq!(listing, 1);
}

// ===========================================================================
// 4. Minting-only pause
// ===========================================================================

#[test]
fn test_scenario_minting_pause_blocks_minting() {
    let (env, ec_id, _mp_id, at_id, admin) = setup();
    let ec_client = EmergencyControlClient::new(&env, &ec_id);
    let at_client = AssetTokenClient::new(&env, &at_id);

    let asset_id: u64 = 1;
    let holder = Address::generate(&env);

    ec_client.pause_asset(
        &admin,
        &asset_id,
        &PauseScope::Minting,
        &String::from_str(&env, "supply audit"),
        &0,
    );

    assert!(panics(|| {
        at_client.mint(&holder, &100, &asset_id, &ec_id);
    }));

    ec_client.unpause_asset(&admin, &asset_id, &PauseScope::Minting);
    at_client.mint(&holder, &100, &asset_id, &ec_id);
}

// ===========================================================================
// 5. Multi-asset isolation
// ===========================================================================

#[test]
fn test_scenario_multi_asset_isolation() {
    let (env, ec_id, mp_id, at_id, admin) = setup();
    let ec_client = EmergencyControlClient::new(&env, &ec_id);
    let mp_client = MarketplaceClient::new(&env, &mp_id);
    let at_client = AssetTokenClient::new(&env, &at_id);

    let seller = Address::generate(&env);
    at_client.mint(&seller, &10_000, &1, &ec_id);
    at_client.mint(&seller, &10_000, &2, &ec_id);

    // Pause only asset 1.
    ec_client.pause_asset(
        &admin,
        &1,
        &PauseScope::Trading,
        &String::from_str(&env, "asset-1 incident"),
        &0,
    );

    // Asset 1 is blocked...
    assert!(panics(|| {
        mp_client.create_listing(&seller, &1, &100, &1000, &ec_id, &None);
    }));

    // ...asset 2 is fully operational.
    let listing = mp_client.create_listing(&seller, &2, &100, &1000, &ec_id, &None);
    assert_eq!(listing, 1);
}

// ===========================================================================
// 6. Circuit breaker trip and manual recovery
// ===========================================================================

#[test]
fn test_scenario_circuit_breaker_trip_and_manual_recovery() {
    let (env, ec_id, mp_id, at_id, admin) = setup();
    let ec_client = EmergencyControlClient::new(&env, &ec_id);
    let mp_client = MarketplaceClient::new(&env, &mp_id);
    let at_client = AssetTokenClient::new(&env, &at_id);

    let asset_id: u64 = 1;
    let seller = Address::generate(&env);
    at_client.mint(&seller, &10_000, &asset_id, &ec_id);

    // A volume spike trips the breaker, pausing trading.
    assert!(ec_client.record_volume(&admin, &asset_id, &50_000, &10_000));
    assert!(ec_client.is_paused(&asset_id, &PauseScope::Trading));

    assert!(panics(|| {
        mp_client.create_listing(&seller, &asset_id, &100, &1000, &ec_id, &None);
    }));

    // Operators manually clear the breaker pause.
    ec_client.circuit_breaker_unpause(&admin, &asset_id);
    assert!(!ec_client.is_paused(&asset_id, &PauseScope::Trading));

    let listing = mp_client.create_listing(&seller, &asset_id, &100, &1000, &ec_id, &None);
    assert_eq!(listing, 1);
}

// ===========================================================================
// 7. Layered pauses
// ===========================================================================

#[test]
fn test_scenario_layered_pause_unpause_order() {
    let (env, ec_id, mp_id, at_id, admin) = setup();
    let ec_client = EmergencyControlClient::new(&env, &ec_id);
    let mp_client = MarketplaceClient::new(&env, &mp_id);
    let at_client = AssetTokenClient::new(&env, &at_id);

    let asset_id: u64 = 1;
    let seller = Address::generate(&env);
    at_client.mint(&seller, &10_000, &asset_id, &ec_id);

    // Pause progressively: trading first, then everything.
    ec_client.pause_asset(
        &admin,
        &asset_id,
        &PauseScope::Trading,
        &String::from_str(&env, "trading halt"),
        &0,
    );
    ec_client.pause_asset(
        &admin,
        &asset_id,
        &PauseScope::All,
        &String::from_str(&env, "global halt"),
        &0,
    );

    // Lifting trading alone must NOT restore listings, because All is still active.
    ec_client.unpause_asset(&admin, &asset_id, &PauseScope::Trading);
    assert!(panics(|| {
        mp_client.create_listing(&seller, &asset_id, &100, &1000, &ec_id, &None);
    }));

    // Only after the global pause is lifted does trading resume.
    ec_client.unpause_asset(&admin, &asset_id, &PauseScope::All);
    let listing = mp_client.create_listing(&seller, &asset_id, &100, &1000, &ec_id, &None);
    assert_eq!(listing, 1);
}

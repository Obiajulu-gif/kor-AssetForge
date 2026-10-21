# Emergency Procedures Runbook

Operator runbook for using the on-chain emergency pause controls in
`contracts/src/emergency_control/`. Follow it during an incident and for the
periodic drills that keep the team ready.

> Pause controls are **admin-gated**. Only the contract admin (or, for fund
> recovery, the configured multi-sig signers) can invoke them. Never share admin
> keys; use the platform's admin tooling or a hardware-wallet signer.

## Scope of the controls

`EmergencyControl` supports granular pauses. Choose the **narrowest scope that
stops the harm**; over-broad pauses unnecessarily halt unaffected activity.

| `PauseScope` | Blocks | Typical use |
|--------------|--------|-------------|
| `All` | Every gated operation (transfers, trading, minting) | Critical, uncertain, or sector-wide incident |
| `Transfers` | Token transfers | Suspected theft / compromised holder |
| `Trading` | Marketplace listing and trading | Market manipulation, price/circuit anomaly |
| `Minting` | New token issuance | Supply / collateral integrity doubts |
| `Function(Symbol)` | A single named operation | A specific contract function misbehaving |
| `User(Address)` | A single account | One malicious or compromised account |
| `Asset(u64)` | A single asset | Isolating one asset without a full halt |

`All` is a wildcard: while it is set, every scope reports as paused.

## Automatic pauses (circuit breakers)

Two circuit breakers can pause `Trading` without an operator:

| Breaker | Trigger | Cooldown |
|---------|---------|----------|
| Price crash | `record_price`: price falls >50% within 1 hour | 720 ledgers (~1 hour), then auto-unpause |
| Volume spike | `record_volume`: observed volume exceeds a supplied threshold | 720 ledgers (~1 hour), then auto-unpause |

Auto-unpause fires the next time paused state is read once the cooldown ledger
has passed. Operators can clear a breaker pause early with
`circuit_breaker_unpause`.

## Symptoms

- `is_paused(asset_id, scope)` returns `true` for a scope you did not expect.
- Marketplace listings fail with `operation blocked: asset is paused`.
- Transfers or mints fail with the same message.
- A `circuit_breaker_tripped` event was emitted for an asset.
- An `asset_paused` event was emitted without a corresponding change ticket.

## Triage

1. **Identify the pause**: call `get_pause_history(asset_id)` and read the most
   recent `PauseRecord` (its `scope`, `reason`, `admin`, `ledger_timestamp`).
2. **Determine the source**:
   - A manual pause has a human-readable `reason` and a known `admin`.
   - `circuit_breaker_price_crash` / `circuit_breaker_volume_spike` means an
     automatic breaker fired.
   - `auto_unpause` means a scheduled/cooldown expiry already lifted a pause.
3. **Assess blast radius**: check which other assets are paused
   (`get_pause_history` per asset) and whether `All` is set anywhere.
4. **Confirm the trigger is understood** before unpausing. Do not unpause until
   the root cause is known — a premature unpause can re-expose the platform.

## Procedure — declare an emergency pause

1. Notify the incident channel and state the asset, scope, and reason.
2. Pause with the narrowest effective scope:

   ```
   pause_asset(admin, asset_id, scope, reason, auto_unpause_ledger)
   ```

   - `reason`: short, factual (e.g. `"suspected key compromise"`).
   - `auto_unpause_ledger`: `0` to require a manual unpause; a future ledger
     sequence to schedule auto-recovery.
3. For a full halt, use `scope = All`.
4. Verify enforcement: `is_paused(asset_id, scope)` must return `true`, and the
   affected operation must now be rejected.
5. Record the action in the incident ticket (who, when, why, scope).

## Procedure — recover / unpause

1. Confirm the trigger is resolved and the change is approved by the incident
   commander.
2. Unpause the exact scope that was set:

   ```
   unpause_asset(admin, asset_id, scope)
   ```

   - If `All` was set, you must unpause `All`; unpausing a narrower scope will
     **not** restore operations while `All` remains set.
   - Manually unpausing clears any pending auto-unpause timer for that scope.
3. Verify `is_paused(asset_id, scope)` returns `false`.
4. Re-run a smoke operation (e.g. a small marketplace listing or transfer).
5. Confirm the pause/unpause pair appears in `get_pause_history`.

## Procedure — emergency fund recovery

Recovery of stranded funds is deliberately slow and multi-party (added in
Issue #205). It is **not** a pause control but is part of the emergency toolkit.

1. Funds may only be sent to an address on the recovery **whitelist**.
2. A withdrawal must be proposed, then approved by the configured multi-sig
   signers until the threshold is met.
3. A **72-hour** timelock (`EMERGENCY_TIMELOCK_SECS`) must elapse before
   execution.
4. Proposals can be cancelled before execution.

Steps: `add_recovery_destination` → `propose_emergency_withdrawal` →
(`approve_emergency_withdrawal` × threshold) → wait 72h →
`execute_emergency_withdrawal`. See `contracts/tests/emergency_test.rs` for the
full lifecycle.

## Escalation and communication

| Step | Owner | Timing |
|------|-------|--------|
| Detect and declare | On-call engineer | Immediate |
| Declare pause | Incident commander / admin key holder | Within 15 min of decision |
| Notify stakeholders | Comms lead | Within 30 min |
| Root-cause analysis | Engineering | Before unpause |
| Unpause | Incident commander | After fix verified |
| Post-incident review | Whole team | Within 5 business days |

## Post-incident

1. Preserve the pause history and all related events/logs.
2. Write a blameless post-incident review covering detection, decision, pause,
   recovery, and customer impact.
3. If a circuit breaker fired, review its thresholds.
4. If a manual pause was needed, confirm the team drill cadence is adequate.

## Team training and drills

Pause controls are only reliable if the team has practised them. The following
must be completed and recorded.

- [ ] Every on-call engineer and incident commander knows where this runbook
      lives and how to read `get_pause_history`.
- [ ] At least two admin key holders are trained and available at all times.
- [ ] The team has walked through a **full-halt** drill (pause `All`, verify
      enforcement, unpause `All`) on testnet.
- [ ] The team has walked through a **single-scope** drill (e.g. `Transfers`)
      and confirmed unaffected operations keep working.
- [ ] The team has observed a **circuit-breaker** trip and cleared it with
      `circuit_breaker_unpause`.
- [ ] The multi-sig fund-recovery lifecycle has been rehearsed end-to-end,
      including the 72-hour timelock, on testnet.
- [ ] Drill date, participants, and any friction points are recorded.

**Drill cadence:** full-halt drill at least **quarterly**; targeted-scope and
circuit-breaker drills at least **every two weeks** as part of on-call
handover.

## Verified test coverage

The behaviour this runbook relies on is covered by automated tests. Run them
before relying on a procedure change:

```
cd contracts
cargo test
```

| File | Covers |
|------|--------|
| `contracts/src/emergency_control/tests.rs` | Every pause scope, auto-unpause timing, history, authorization, circuit breakers |
| `contracts/tests/emergency_scenarios.rs` | Full drill, scope isolation, multi-asset isolation, circuit-breaker recovery, layered pauses |
| `contracts/tests/emergency_test.rs` | Multi-sig fund recovery lifecycle |

## Related files

| File | Purpose |
|------|---------|
| `contracts/src/emergency_control/mod.rs` | Pause/circuit-breaker implementation |
| `contracts/src/emergency_control/tests.rs` | Unit tests |
| `contracts/tests/emergency_scenarios.rs` | Scenario tests |
| `contracts/tests/emergency_test.rs` | Fund-recovery tests |
| `docs/runbooks/elasticsearch-failures.md` | Example of the runbook format |

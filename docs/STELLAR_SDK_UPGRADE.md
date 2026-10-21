# Stellar SDK Version Management

This document describes the backend's Stellar SDK dependency management and
version strategy.

## Current Version

The backend uses:

```go
github.com/stellar/go v0.0.0-20251210100531-aab2ea4aca88
```

This is a **pseudo-version** referencing the December 10, 2025 commit
(`aab2ea4aca88`) from the stellar/go repository.

## Repository Status

**Important**: As of December 16, 2025, the `github.com/stellar/go` repository
was archived by the Stellar Development Foundation. The repository is now
read-only and contains no tagged releases.

### Why No Tagged Release?

The stellar/go repository was a monorepo containing multiple services and SDK
packages. The repository never published semantic version tags suitable for Go
module consumption. Instead, it operated with pseudo-versions based on commit
timestamps.

### Current Situation

The pseudo-version `v0.0.0-20251210100531` is effectively the **latest stable
version** available from this repository, captured 6 days before archival. This
version:

- Represents the last working state before the repository was archived
- Includes all SDK packages used by this application
- Has a known commit hash for reproducible builds
- Is present in the Go module checksum database

## SDK Packages Used

The backend imports the following packages from `github.com/stellar/go`:

### Core SDK Packages

| Package                                      | Usage                                    |
|----------------------------------------------|------------------------------------------|
| `github.com/stellar/go/clients/horizonclient`| Horizon API client (account balances)    |
| `github.com/stellar/go/clients/rpcclient`    | Soroban RPC client (contract invocation) |
| `github.com/stellar/go/keypair`              | Key pair generation and parsing          |
| `github.com/stellar/go/network`              | Network passphrases                      |
| `github.com/stellar/go/protocols/rpc`        | Soroban RPC protocol types               |
| `github.com/stellar/go/strkey`               | Stellar key encoding (StrKey)            |
| `github.com/stellar/go/txnbuild`             | Transaction builder                      |
| `github.com/stellar/go/xdr`                  | XDR encoding/decoding                    |

### Files Importing Stellar SDK

- `backend/utils/stellar_utils.go` - Primary Stellar integration
- `backend/utils/stellar_utils_test.go` - Unit tests
- `backend/validator/validator.go` - Address validation
- `backend/config/config.go` - StrKey validation
- `backend/handlers/assets_test.go` - Test fixtures
- `backend/router/router_test.go` - Test fixtures
- `backend/services/external_services_mock_test.go` - Mock implementations
- `backend/tests/mocks/stellar_client.go` - Test mocks
- `backend/tests/integration/test_helper.go` - Integration test helpers

## API Stability

The December 10, 2025 pseudo-version includes stable APIs for:

### Horizon Client

```go
client := horizonclient.DefaultTestNetClient
account, err := client.AccountDetail(horizonclient.AccountRequest{
    AccountID: address,
})
```

No breaking changes expected in this API — it has been stable for years.

### Soroban RPC Client

```go
rpcClient := rpcclient.NewClient(sorobanURL, http.DefaultClient)
sim, err := rpcClient.SimulateTransaction(ctx, protocol.SimulateTransactionRequest{
    Transaction: txEnvelope,
    ResourceConfig: &protocol.ResourceConfig{
        InstructionLeeway: protocol.DefaultInstructionLeeway,
    },
})
```

This API was introduced with Soroban smart contracts and stabilized in 2024.
The December 2025 version includes all production-ready features.

### Transaction Building

```go
tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
    SourceAccount:        account,
    IncrementSequenceNum: true,
    Operations:           []txnbuild.Operation{op},
    BaseFee:              txnbuild.MinBaseFee,
    Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(60)},
})
```

Transaction building APIs are stable and backward-compatible.

## Future Migration Path

### Option 1: Continue with Archived Repository

The current approach is **stable and acceptable** for production use:

- ✅ Commit hash is immutable and auditable
- ✅ Checksums are in Go module database
- ✅ All imports continue to work
- ✅ No breaking changes expected

**Recommendation**: Continue using the current pseudo-version. The archived
repository remains accessible for Go module resolution.

### Option 2: Migrate to New SDK Repository

In October 2025, the Stellar Development Foundation created
`github.com/stellar/go-stellar-sdk` as a dedicated SDK repository. However, as
of the archival date, this repository was not yet ready for production use.

If migrating in the future:

1. **Check SDK repository status**:
   ```bash
   go list -m -versions github.com/stellar/go-stellar-sdk
   ```

2. **Review migration guide**: The new repository should provide a migration
   guide documenting package renames and API changes.

3. **Update imports**: Replace all `github.com/stellar/go/*` imports with the
   new package paths.

4. **Test thoroughly**: Run the full test suite, including integration tests
   against Stellar testnet.

**Do not migrate** until the new SDK repository has:
- Published semantic version tags (v1.0.0+)
- Complete API documentation
- An official migration guide
- Community adoption and stability

## Security Considerations

### Pseudo-Versions and Security

Pseudo-versions are **not inherently insecure**. They:

- Reference a specific commit (immutable)
- Are verified via Go module checksums
- Provide reproducible builds

The concern with pseudo-versions is **discoverability**: it's harder to track
security advisories without semantic versions. However, given that the
repository is archived and no further development is occurring, the security
profile is frozen.

### Monitoring for Vulnerabilities

Run `govulncheck` regularly:

```bash
cd backend
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

This scans for known vulnerabilities in the Go ecosystem. As of the December
2025 snapshot, no critical vulnerabilities were present in the Stellar SDK
packages.

### Dependency Transparency

The exact version in use can always be verified:

```bash
cd backend
go mod download github.com/stellar/go@v0.0.0-20251210100531-aab2ea4aca88
go list -m -json github.com/stellar/go
```

Output includes commit hash, timestamp, and checksum for full transparency.

## Testing Strategy

### Unit Tests

All Stellar integration is unit tested with fake RPC implementations:

```bash
cd backend
go test ./utils -v
```

Tests cover:
- Contract ID validation
- Parameter encoding
- Simulation handling
- Transaction signing and submission
- Error cases

### Integration Tests

Integration tests against Stellar testnet should be run manually (not in CI):

```bash
export STELLAR_NETWORK=testnet
export STELLAR_SIGNER_SECRET=S...
export STELLAR_SOROBAN_RPC_URL=https://soroban-testnet.stellar.org
cd backend
go test ./tests/integration -v
```

These require a funded testnet account and should not block CI.

## Version History

**v0.0.0-20251210100531** (Issue #434): Documented the current pseudo-version
as the latest available before the stellar/go repository was archived. No
version upgrade performed because:

1. The repository has no tagged releases
2. The current pseudo-version is the last stable commit before archival
3. All APIs in use are stable and production-ready
4. Migration to a new SDK repository is premature pending official release

## References

- [Archived Stellar Go Repository](https://github.com/stellar/go)
- [New Stellar Go SDK Repository](https://github.com/stellar/go-stellar-sdk)
- [Go Modules Reference](https://go.dev/ref/mod)
- [Using Pseudo-Versions](https://go.dev/ref/mod#pseudo-versions)

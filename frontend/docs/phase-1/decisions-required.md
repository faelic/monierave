# Decisions Required Before Phase 2

## 1. Saved-beneficiary transfer contract

**Issue:** Beneficiaries are safely returned with masked destination account
numbers. Transfers currently require the complete destination account number.
After a page refresh, the client therefore cannot use a saved beneficiary
without asking the user to enter the full number again.

**Recommendation:** Add `beneficiary_id` as an alternative transfer destination.
The backend should resolve the beneficiary inside the same transfer transaction,
verify ownership, lock the destination account, and preserve all existing
lifecycle, currency, limit, balance, and idempotency checks.

The request should accept exactly one of:

```json
{
  "to_account_number": "4839201756"
}
```

or:

```json
{
  "beneficiary_id": "beneficiary-public-uuid"
}
```

This keeps full account numbers out of persistent frontend state and makes the
beneficiary feature useful. Do not reveal the full destination number in the
beneficiary response as a workaround.

**Phase 1 decision:** Approve the backend extension or remove beneficiary
selection from the initial transfer journey.

## 2. Authenticated URL prefix

**Recommendation:** Keep `/app` as the authenticated banking prefix. It creates
a clear security and layout boundary, avoids collisions with marketing routes,
and makes middleware behavior easier to reason about.

**Alternative:** Use top-level routes such as `/accounts` and `/transfers`.
This creates shorter URLs but a less explicit application boundary.

**Phase 1 decision:** Approve `/app` before page URLs become public contracts.

## 3. Cross-account transaction activity

**Issue:** The backend confirms account-scoped history but has no global
customer transaction-history endpoint.

**Recommendation:** For MVP, `/app/transactions` first asks the user to select
an account and then uses the confirmed account-scoped endpoint. Do not merge
several paginated account feeds in the browser because ordering, filtering, and
cursor semantics would be unreliable.

**Future backend option:** Add an ownership-scoped global transaction feed with
stable cursor pagination and the same filter vocabulary.

**Phase 1 decision:** Approve account selection for MVP or prioritize the global
history endpoint before frontend Phase 7.

## 4. Unverified-user navigation

**Recommendation:** Use a small authenticated verification layout rather than
the full banking shell. It contains profile/email correction, verification
status, resend, and sign-out actions only.

This prevents disabled financial controls from overwhelming the user and
matches the backend restriction boundary.

**Phase 1 decision:** Approve the focused verification layout.

## 5. Unsupported product visibility

**Recommendation:** Hide KYC, cards, notification inbox, MFA, device management,
and external-transfer navigation until their backend contracts exist. Marketing
content may describe them only as planned and must not imply availability.

**Phase 1 decision:** Approve strict capability-based navigation for the MVP.

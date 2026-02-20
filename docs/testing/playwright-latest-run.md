# Playwright Latest Run

- Date: 2026-02-20 (latest rerun)
- Environment: local `http://localhost:8080` with dev auth flow
- Summary: full checklist automation now runs in `tests/e2e/smoke.spec.js` and passed locally (6/6). Release analytics UI rendered and remained stable.

## Checklist result

- [x] Dev login works and redirects correctly
- [x] Header nav works across pages
- [x] Welcome flow appears for users with no memberships
- [x] New user can create organization and enter app
- [x] Join code is visible in organization members section
- [x] Join instructions are visible and reference `/welcome`
- [x] Joiner can submit join request via `/welcome`
- [x] Admin sees pending request and can approve
- [x] Approved user appears in member list
- [x] Membership actions redirect to `/organizations#members`
- [x] Service page shows environment deploy metrics + release change log labels (when deployment data exists)
- [x] Deployments page renders environment metrics cards (when deployment data exists)
- [x] Automated smoke spec (`tests/e2e/smoke.spec.js`) passes locally with seeded dataset
- [x] Header nav + settings tabs/save + logout covered by automated smoke
- [x] Membership add/update/remove and `#members` redirect covered by automated smoke
- [x] Deployments filtering covered by automated smoke

## Follow-ups

- Expand Playwright coverage into automated CI browser tests (currently checklist is manual-assisted with browser automation).
- Add deterministic seeded dataset per test org so release analytics cards are always non-empty in browser run.

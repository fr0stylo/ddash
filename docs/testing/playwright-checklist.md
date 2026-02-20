# Playwright Usability Checklist

Use this checklist when validating core DDash user flows with Playwright against a running local server.

## Preconditions

- Server is running on `http://localhost:8080`
- Dev auth shortcut is enabled (`DDASH_ENV=dev` or local/test env)
- Seed data is available (optional but recommended)

## Checklist

- [ ] Dev login works and redirects to requested page (`/auth/dev/login?...&next=/`)
- [ ] Header navigation works: Services, Deployments, Settings, Organizations
- [ ] Organizations page renders with:
  - [ ] member management section
  - [ ] visible join code
  - [ ] join instructions mentioning `/welcome`
- [ ] Membership actions stay on same page section (`#members` anchor):
  - [ ] add member
  - [ ] update role
  - [ ] remove member
- [ ] Join request approval flow works:
  - [ ] requester submits join code on `/welcome`
  - [ ] admin sees pending request in Organizations page
  - [ ] admin can approve/reject request
- [ ] Welcome flow for users with no org membership:
  - [ ] shows create/join options
  - [ ] create organization redirects to app
  - [ ] invalid join code shows error
- [ ] Deployments filter interaction updates rows
- [ ] Service details page renders without JS/HTMX console errors
- [ ] Settings tabs switch and save works
- [ ] Logout returns to `/login`

## Notes template

- Date:
- Environment:
- Result summary:
- Failing checks:
  -
- Follow-up issues:
  -

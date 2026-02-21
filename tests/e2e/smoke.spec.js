const { test, expect } = require('@playwright/test')

const baseURL = process.env.E2E_BASE_URL || 'http://localhost:8080'

async function devLogin(page, { email, nickname, name, next = '/' }) {
  const query = new URLSearchParams({ email, nickname, name, next }).toString()
  await page.goto(`${baseURL}/auth/dev/login?${query}`)
}

function trackConsoleErrors(page) {
  const errors = []
  page.on('console', (msg) => {
    if (msg.type() !== 'error') {
      return
    }
    const text = msg.text() || ''
    if (text.includes('favicon.ico')) {
      return
    }
    errors.push(text)
  })
  return () => {
    expect(errors).toEqual([])
  }
}

test('deployments and service release analytics render with seeded events', async ({ page }) => {
  const assertNoConsoleErrors = trackConsoleErrors(page)

  await page.goto(`${baseURL}/auth/dev/login?email=e2e-admin@example.local&nickname=e2e-admin&name=E2E%20Admin&next=/deployments`)
  await expect(page).toHaveURL(/\/deployments/)

  await expect(page.getByRole('heading', { name: 'Deployments' })).toBeVisible()
  await expect(page.getByText('staging').first()).toBeVisible()
  await expect(page.getByText('production').first()).toBeVisible()

  await page.goto(`${baseURL}/s/orders`)
  await expect(page.getByRole('heading', { name: 'orders' })).toBeVisible()
  await expect(page.getByText(/deploys \/ 7d/).first()).toBeVisible()
  await expect(page.getByText('Deployment history')).toBeVisible()
  await expect(page.getByText('Updated from previous release').first()).toBeVisible()
  await expect(page.getByText(/^from pkg:generic\/orders@/).first()).toBeVisible()

  assertNoConsoleErrors()
})

test('welcome flow and join request approval path work', async ({ page }) => {
  const assertNoConsoleErrors = trackConsoleErrors(page)
  const suffix = Date.now().toString()
  const joinerEmail = `e2e-joiner-${suffix}@example.local`
  const joinerNick = `e2ejoiner${suffix}`
  const joinerName = `E2E Joiner ${suffix}`

  await devLogin(page, {
    email: joinerEmail,
    nickname: joinerNick,
    name: joinerName,
    next: '/welcome',
  })
  await expect(page).toHaveURL(/\/welcome/)

  await page.getByRole('textbox', { name: 'join code' }).fill('e2ejoincode01')
  await page.getByRole('button', { name: 'Request access' }).click()
  await expect(page.getByText('Join request submitted. Wait for admin approval.')).toBeVisible()

  await page.goto(`${baseURL}/auth/dev/login?email=e2e-admin@example.local&nickname=e2e-admin&name=E2E%20Admin&next=/organizations`)
  await expect(page).toHaveURL(/\/organizations/)
  await expect(page.getByRole('heading', { name: 'Pending join requests' })).toBeVisible()
  await expect(page.getByText(joinerName)).toBeVisible()
  await page.getByRole('button', { name: 'Approve' }).first().click()
  await expect(page).toHaveURL(/#members/)
  await expect(page.getByText('Join request approved')).toBeVisible()
  assertNoConsoleErrors()
})

test('header navigation, settings tabs/save, and logout work', async ({ page }) => {
  const assertNoConsoleErrors = trackConsoleErrors(page)
  await devLogin(page, {
    email: 'e2e-admin@example.local',
    nickname: 'e2e-admin',
    name: 'E2E Admin',
    next: '/',
  })
  await expect(page).toHaveURL(/\/$/)

  await page.getByRole('link', { name: 'Deployments', exact: true }).click()
  await expect(page).toHaveURL(/\/deployments/)

  await page.getByRole('link', { name: 'Settings', exact: true }).click()
  await expect(page).toHaveURL(/\/settings/)
  await page.getByRole('button', { name: 'Features' }).click()
  await page.getByRole('button', { name: 'Save settings' }).click()
  await expect(page.getByText('Settings saved')).toBeVisible()

  await page.getByRole('link', { name: 'Sign out' }).click()
  await expect(page).toHaveURL(/\/login/)
  assertNoConsoleErrors()
})

test('welcome screen supports create org and invalid join code feedback', async ({ page }) => {
  const assertNoConsoleErrors = trackConsoleErrors(page)
  const suffix = Date.now().toString()
  await devLogin(page, {
    email: `e2e-new-${suffix}@example.local`,
    nickname: `e2enew${suffix}`,
    name: `E2E New ${suffix}`,
    next: '/welcome',
  })
  await expect(page).toHaveURL(/\/welcome/)
  await expect(page.getByRole('heading', { name: 'Create new organization' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Join existing organization' })).toBeVisible()

  await page.getByRole('textbox', { name: 'join code' }).fill('bad-code')
  await page.getByRole('button', { name: 'Request access' }).click()
  await expect(page.getByText('Invalid join code')).toBeVisible()

  await page.getByRole('textbox', { name: 'my-team-org' }).fill(`e2e-created-${suffix}-org`)
  await page.getByRole('button', { name: 'Create organization' }).click()
  await expect(page).not.toHaveURL(/\/welcome/)
  assertNoConsoleErrors()
})

test('organizations page shows join info and membership actions stay in members tab', async ({ page }) => {
  const assertNoConsoleErrors = trackConsoleErrors(page)
  const suffix = Date.now().toString()
  const candidateEmail = `e2e-member-${suffix}@example.local`
  const candidateNick = `e2emember${suffix}`

  // Create candidate user so admin can add by identity lookup.
  await devLogin(page, {
    email: candidateEmail,
    nickname: candidateNick,
    name: `E2E Member ${suffix}`,
    next: '/welcome',
  })

  await devLogin(page, {
    email: 'e2e-admin@example.local',
    nickname: 'e2e-admin',
    name: 'E2E Admin',
    next: '/organizations',
  })
  await expect(page).toHaveURL(/\/organizations/)
  await expect(page.getByRole('heading', { name: 'Manage members' })).toBeVisible()
  await expect(page.getByText('Join code:')).toBeVisible()
  await expect(page.getByText('/welcome')).toBeVisible()

  await page.getByRole('textbox', { name: 'email or nickname' }).fill(candidateEmail)
  await page.getByRole('button', { name: 'Add' }).click()
  await expect(page).toHaveURL(/#members/)
  await expect(page.getByText('Member added')).toBeVisible()
  await expect(page.getByText(candidateEmail)).toBeVisible()

  const memberRow = page.locator(`xpath=//p[contains(normalize-space(), "${candidateEmail}")]/ancestor::div[contains(@class,'py-3')][1]`)
  const memberRoleForm = memberRow.locator('form[action="/organizations/members/role"]')
  await expect(memberRow).toBeVisible()
  await memberRoleForm.locator('select[name="role"]').selectOption('admin')
  await memberRoleForm.getByRole('button', { name: 'Update' }).click()
  await expect(page).toHaveURL(/#members/)
  await expect(page.getByText('Role updated')).toBeVisible()

  await memberRow.getByRole('button', { name: 'Remove' }).click()
  await expect(page).toHaveURL(/#members/)
  await expect(page.getByText('Member removed')).toBeVisible()
  assertNoConsoleErrors()
})

test('deployments filter interaction updates rows', async ({ page }) => {
  const assertNoConsoleErrors = trackConsoleErrors(page)
  await devLogin(page, {
    email: 'e2e-admin@example.local',
    nickname: 'e2e-admin',
    name: 'E2E Admin',
    next: '/deployments',
  })
  await expect(page.getByRole('heading', { name: 'Deployments' })).toBeVisible()

  await expect(page.getByRole('cell', { name: 'billing' }).first()).toBeVisible()
  const serviceFilter = page.locator('#deployment-service')
  await serviceFilter.selectOption('orders')
  await page.evaluate(() => {
    window.htmx.ajax('GET', '/deployments/filter?env=all&service=orders', '#deployment-results')
  })
  const filteredResults = page.locator('#deployment-results').last()
  await expect(filteredResults).toContainText('orders')
  await expect(filteredResults).not.toContainText('billing')
  assertNoConsoleErrors()
})

test('join request reject flow works', async ({ page }) => {
  const assertNoConsoleErrors = trackConsoleErrors(page)
  const suffix = Date.now().toString()
  const joinerEmail = `e2e-reject-${suffix}@example.local`
  const joinerNick = `e2ereject${suffix}`
  const joinerName = `E2E Reject ${suffix}`

  await devLogin(page, {
    email: joinerEmail,
    nickname: joinerNick,
    name: joinerName,
    next: '/welcome',
  })
  await page.getByRole('textbox', { name: 'join code' }).fill('e2ejoincode01')
  await page.getByRole('button', { name: 'Request access' }).click()
  await expect(page.getByText('Join request submitted. Wait for admin approval.')).toBeVisible()

  await devLogin(page, {
    email: 'e2e-admin@example.local',
    nickname: 'e2e-admin',
    name: 'E2E Admin',
    next: '/organizations',
  })
  const pendingRow = page.locator(`xpath=//p[contains(normalize-space(), "${joinerName}")]/ancestor::div[contains(@class,'py-3')][1]`)
  await expect(pendingRow).toBeVisible()
  await pendingRow.locator('form[action="/organizations/join-requests/reject"]').getByRole('button', { name: 'Reject' }).click()
  await expect(page).toHaveURL(/#members/)
  await expect(page.getByText('Join request rejected')).toBeVisible()
  await expect(page.getByText(joinerName)).toHaveCount(0)
  assertNoConsoleErrors()
})

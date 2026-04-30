import { test, expect } from '@playwright/test';

/**
 * E2E tests for the onboarding and analysis user flows.
 *
 * These tests cover:
 *   - Flow 1: OnboardingGuide shown when no server is running
 *   - Flow 2: Analyze form when server has zero repos
 *   - Flow 3: Auto-connect when server has repos
 *   - Flow 4: Repo dropdown in exploring view
 *
 * Most tests mock the backend at the network level so they don't
 * require a live avmatrix server.
 */

const BACKEND_URL = 'http://localhost:4747';
const ABSOLUTE_LOCAL_PATH = process.platform === 'win32' ? 'C:\\repos\\demo' : '/tmp/demo';

async function enterExploringView(page: import('@playwright/test').Page) {
  await page.goto('/');

  const landingCard = page.locator('[data-testid="landing-repo-card"]').first();
  try {
    await landingCard.waitFor({ state: 'visible', timeout: 15_000 });
    await landingCard.click();
  } catch {
    // Landing screen may not appear (e.g. ?server auto-connect)
  }

  // Match the 45s budget used by waitForGraphLoaded() in
  // server-connect.spec.ts; under parallel CI workers, downloading the full
  // graph can occasionally exceed 30s.
  await expect(page.locator('[data-testid="status-ready"]')).toBeVisible({ timeout: 45_000 });
}

// ── Flow 1: Onboarding (no server running) ─────────────────────────────────

test.describe('Flow 1: Onboarding — no server', () => {
  test('shows OnboardingGuide when backend is unreachable', async ({ page }, testInfo) => {
    // Block all requests to the backend so the probe fails
    await page.route(`${BACKEND_URL}/**`, (route) => route.abort('connectionrefused'));

    await page.goto('/');

    // Wait for initial probe to complete and onboarding to appear
    await expect(page.getByText('Start AVmatrix locally')).toBeVisible({ timeout: 10_000 });
    await page.screenshot({ path: testInfo.outputPath('onboarding-visible.png') });
  });

  test('shows step-by-step instructions', async ({ page }) => {
    await page.route(`${BACKEND_URL}/**`, (route) => route.abort('connectionrefused'));
    await page.goto('/');

    // Step 1 is active (done once polling starts)
    await expect(page.getByText('Copy the command')).toBeAttached({ timeout: 10_000 });
    // Step 2 title changes to "Waiting for server to start" once polling begins
    await expect(page.getByText('Waiting for local bridge to start')).toBeAttached({
      timeout: 10_000,
    });
    // Step 3 is always rendered
    await expect(page.getByText('Auto-connects and opens the graph')).toBeAttached({
      timeout: 5_000,
    });
  });

  test('shows terminal window with command', async ({ page }) => {
    await page.route(`${BACKEND_URL}/**`, (route) => route.abort('connectionrefused'));
    await page.goto('/');

    // Should show either dev or prod command in a terminal block
    const terminal = page.locator('code');
    await expect(terminal.first()).toBeVisible({ timeout: 10_000 });

    // The $ prompt should be present
    await expect(page.getByText('$')).toBeVisible();
  });

  test('shows polling indicator', async ({ page }) => {
    await page.route(`${BACKEND_URL}/**`, (route) => route.abort('connectionrefused'));
    await page.goto('/');

    // Polling starts after initial probe fails
    await expect(page.getByText('Listening for local bridge')).toBeVisible({ timeout: 10_000 });
  });

  test('shows Node.js version requirement', async ({ page }) => {
    await page.route(`${BACKEND_URL}/**`, (route) => route.abort('connectionrefused'));
    await page.goto('/');

    await expect(page.getByText(/Node\.js.*\d+/)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Port 4747')).toBeVisible();
  });

  test('copy button has accessible label', async ({ page }) => {
    await page.route(`${BACKEND_URL}/**`, (route) => route.abort('connectionrefused'));
    await page.goto('/');

    await expect(page.getByText('Copy the command')).toBeVisible({ timeout: 10_000 });
    const copyBtn = page.getByLabel('Copy to clipboard').first();
    await expect(copyBtn).toBeVisible();
  });
});

// ── Flow 2: Server detected → success → auto-connect ──────────────────────

test.describe('Flow 2: Server detected — auto-connect', () => {
  test('shows success card when server becomes reachable', async ({ page }, testInfo) => {
    // Start with server unreachable
    let blockBackend = true;
    await page.route(`${BACKEND_URL}/**`, (route) => {
      if (blockBackend) return route.abort('connectionrefused');
      // Let it through to the real handler below
      return route.fallback();
    });

    // Mock the backend responses for when we "start" the server
    await page.route(`${BACKEND_URL}/api/repos`, async (route) => {
      if (blockBackend) return route.abort('connectionrefused');
      await route.fulfill({ json: [{ name: 'test-repo', path: '/tmp/test' }] });
    });
    await page.route(`${BACKEND_URL}/api/repo`, async (route) => {
      if (blockBackend) return route.abort('connectionrefused');
      await route.fulfill({
        json: { name: 'test-repo', path: '/tmp/test', repoPath: '/tmp/test' },
      });
    });
    await page.route(`${BACKEND_URL}/api/graph**`, async (route) => {
      if (blockBackend) return route.abort('connectionrefused');
      await route.fulfill({ json: { nodes: [], relationships: [] } });
    });
    await page.route(`${BACKEND_URL}/api/heartbeat`, async (route) => {
      if (blockBackend) return route.abort('connectionrefused');
      // SSE response
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' },
        body: ':ok\n\n',
      });
    });

    await page.goto('/');

    // Verify onboarding is shown first
    await expect(page.getByText('Start AVmatrix locally')).toBeVisible({ timeout: 10_000 });
    await page.screenshot({ path: testInfo.outputPath('before-server-start.png') });

    // "Start" the server by unblocking requests
    blockBackend = false;

    // Wait for success card
    await expect(page.getByText('Server Connected')).toBeVisible({ timeout: 15_000 });
    await page.screenshot({ path: testInfo.outputPath('success-card.png') });
  });

  test('transitions to analyze phase when server has zero repos', async ({ page }, testInfo) => {
    // Mock server with zero repos — repos endpoint returns empty array
    await page.route(`${BACKEND_URL}/api/repos`, (route) => route.fulfill({ json: [] }));
    await page.route(`${BACKEND_URL}/api/info`, (route) =>
      route.fulfill({ json: { version: '1.0.0', launchContext: 'npx', nodeVersion: 'v22.0.0' } }),
    );
    await page.route(`${BACKEND_URL}/api/heartbeat`, (route) =>
      route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body: ':ok\n\n',
      }),
    );

    await page.goto('/');

    // Should transition: onboarding → success → analyze (zero repos)
    await expect(page.getByText('Local Folder Path')).toBeVisible({ timeout: 20_000 });
    await expect(page.getByText('Folder picker hint')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('analyze-empty-state.png') });
  });
});

// ── Flow 3: Analyze form ───────────────────────────────────────────────────

test.describe('Flow 3: Analyze form', () => {
  test.beforeEach(async ({ page }) => {
    // Mock server with zero repos to show the analyze form
    await page.route(`${BACKEND_URL}/api/repos`, (route) => route.fulfill({ json: [] }));
    await page.route(`${BACKEND_URL}/api/info`, (route) =>
      route.fulfill({ json: { version: '1.0.0', launchContext: 'npx', nodeVersion: 'v22.0.0' } }),
    );
    await page.route(`${BACKEND_URL}/api/heartbeat`, (route) =>
      route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body: ':ok\n\n',
      }),
    );
  });

  test('local path input validates absolute paths', async ({ page }, testInfo) => {
    await page.goto('/');

    // Wait for analyze form (transition: onboarding → success → analyze)
    await expect(page.getByText('Local Folder Path')).toBeVisible({ timeout: 20_000 });

    // Type an invalid relative path
    const input = page.locator('input[type="text"]').first();
    await input.fill('not-a-path');

    // Analyze button should be visible but disabled
    const analyzeBtn = page.getByRole('button', { name: /Analyze Repository/ });
    await expect(analyzeBtn).toBeVisible();
    await expect(analyzeBtn).toBeDisabled();

    // Type a valid absolute local path
    await input.fill(ABSOLUTE_LOCAL_PATH);
    await expect(analyzeBtn).toBeEnabled();
    await page.screenshot({ path: testInfo.outputPath('valid-local-path.png') });
  });

  test('local-only analyze form shows folder picker hint', async ({ page }, testInfo) => {
    await page.goto('/');

    await expect(page.getByText('Local Folder Path')).toBeVisible({ timeout: 20_000 });
    await expect(page.getByText('Folder picker hint')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('local-folder-input.png') });
  });

  test('invalid path keeps analyze disabled until corrected', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByText('Local Folder Path')).toBeVisible({ timeout: 20_000 });

    const pathInput = page.locator('input[type="text"]').first();
    await pathInput.fill('relative-folder');
    const analyzeButton = page.getByRole('button', { name: /Analyze Repository/i });
    await expect(analyzeButton).toBeDisabled();

    await pathInput.fill(ABSOLUTE_LOCAL_PATH);
    await expect(analyzeButton).toBeEnabled();
  });
});

// ── Flow 4: Repo dropdown (requires running server) ────────────────────────

test.describe('Flow 4: Repo dropdown in exploring view', () => {
  const SKIP_MSG = 'Requires running avmatrix server with indexed repos';

  // enterExploringView() can take up to ~45s under parallel CI workers; combined
  // with the dropdown interactions this can exceed the default 60s test budget.
  test.slow();

  test.beforeAll(async () => {
    if (process.env.E2E) return;
    try {
      const res = await fetch(`${BACKEND_URL}/api/repos`);
      if (!res.ok) {
        test.skip(true, SKIP_MSG);
        return;
      }
      const repos = await res.json();
      if (!repos.length) {
        test.skip(true, 'Server has no indexed repos');
        return;
      }
    } catch {
      test.skip(true, SKIP_MSG);
    }
  });

  test('project badge opens repo dropdown', async ({ page }, testInfo) => {
    await enterExploringView(page);
    await page.screenshot({ path: testInfo.outputPath('exploring-loaded.png') });

    // Click the project badge (has a chevron)
    const badge = page
      .locator('header button')
      .filter({ has: page.locator('svg') })
      .first();
    await badge.click();

    // Repo dropdown should be visible
    await expect(page.getByText('Repositories')).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText('Analyze a new repository')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('repo-dropdown-open.png') });
  });

  test('analyze option opens inline form', async ({ page }, testInfo) => {
    await enterExploringView(page);

    // Open repo dropdown
    const badge = page
      .locator('header button')
      .filter({ has: page.locator('svg') })
      .first();
    await badge.click();

    // Click "Analyze a new repository..."
    await page.getByText('Analyze a new repository').click();

    // Should show the local-only analyze form inline
    await expect(page.getByText('Local Folder Path')).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText('Folder picker hint')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('inline-analyze-form.png') });
  });
});

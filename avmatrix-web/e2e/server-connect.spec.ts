import { test, expect } from '@playwright/test';

/**
 * E2E tests for the AVmatrix web UI — exploring view features.
 *
 * Requires:
 *   - avmatrix serve running on localhost:4747 with at least one indexed repo
 *   - avmatrix-web dev server running on localhost:5173
 *
 * Skipped when servers aren't available (CI without services, etc.).
 * Set E2E=1 to force-run even without the availability check.
 */

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:4747';
const FRONTEND_URL = process.env.FRONTEND_URL ?? 'http://localhost:5173';

let firstRepoName = '';

test.beforeAll(async () => {
  if (process.env.E2E) {
    const res = await fetch(`${BACKEND_URL}/api/repos`);
    const repos = await res.json();
    firstRepoName = repos[0]?.name ?? '';
    if (!firstRepoName) test.skip(true, 'No indexed repos - run avmatrix analyze first');
    return;
  }
  try {
    const [backendRes, frontendRes] = await Promise.allSettled([
      fetch(`${BACKEND_URL}/api/repos`),
      fetch(FRONTEND_URL),
    ]);
    if (
      backendRes.status === 'rejected' ||
      (backendRes.status === 'fulfilled' && !backendRes.value.ok)
    ) {
      test.skip(true, 'avmatrix serve not available on :4747');
      return;
    }
    if (
      frontendRes.status === 'rejected' ||
      (frontendRes.status === 'fulfilled' && !frontendRes.value.ok)
    ) {
      test.skip(true, 'Vite dev server not available on :5173');
      return;
    }
    // Check there's at least one indexed repo
    if (backendRes.status === 'fulfilled') {
      const repos = await backendRes.value.json();
      if (!repos.length) {
        test.skip(true, 'No indexed repos — run avmatrix analyze first');
        return;
      }
      firstRepoName = repos[0].name;
    }
  } catch {
    test.skip(true, 'servers not available');
  }
});

/**
 * Wait for the repo-scoped graph flow to complete.
 * The current architecture does not rely on a process-global active repo, so
 * E2E tests must pass the target repo context explicitly.
 */
async function waitForGraphLoaded(page: import('@playwright/test').Page) {
  await page.goto(
    `${FRONTEND_URL}/?server=${encodeURIComponent(BACKEND_URL)}&project=${encodeURIComponent(firstRepoName)}`,
  );

  const statusBar = page.getByRole('contentinfo');
  await expect(statusBar.getByText('Ready', { exact: true })).toBeVisible({ timeout: 45_000 });
  await expect(statusBar).toContainText(/nodes/, {
    timeout: 20_000,
  });
}

test.describe('Server Connection & Graph Loading', () => {
  test('selects a repo from landing and loads graph', async ({ page }) => {
    await waitForGraphLoaded(page);
  });
});

test.describe('My AI', () => {
  test('panel opens and agent initializes without error', async ({ page }) => {
    await waitForGraphLoaded(page);

    await page.getByRole('button', { name: 'My AI' }).click();
    await expect(page.getByText('Ask me anything')).toBeVisible({ timeout: 15_000 });

    const errorBanner = page.getByText('Database not ready');
    expect(await errorBanner.isVisible().catch(() => false)).toBe(false);
  });
});

test.describe('Processes Panel', () => {
  test('shows process list and View button works', async ({ page }) => {
    await waitForGraphLoaded(page);

    await page.getByRole('button', { name: 'My AI' }).click();
    await page.getByText('Processes').click();

    await expect(page.locator('[data-testid="process-list-loaded"]')).toBeVisible({
      timeout: 15_000,
    });

    const processRow = page.locator('[data-testid="process-row"]').first();
    await expect(processRow).toBeVisible({ timeout: 10_000 });
    await processRow.hover();

    const viewBtn = processRow.locator('[data-testid="process-view-button"]');
    await viewBtn.waitFor({ state: 'visible', timeout: 5_000 });
    await viewBtn.click();
    await expect(page.locator('[data-testid="process-modal"]')).toBeVisible({ timeout: 5_000 });
  });

  test('lightbulb highlights nodes in graph', async ({ page }) => {
    await waitForGraphLoaded(page);

    await page.getByRole('button', { name: 'My AI' }).click();
    await page.getByText('Processes').click();
    await expect(page.locator('[data-testid="process-list-loaded"]')).toBeVisible({
      timeout: 15_000,
    });

    const processRow = page.locator('[data-testid="process-row"]').first();
    await expect(processRow).toBeVisible({ timeout: 10_000 });
    await processRow.hover();

    const lightbulb = processRow.locator('[data-testid="process-highlight-button"]');
    await lightbulb.waitFor({ state: 'visible', timeout: 5_000 });
    await lightbulb.click();
    await expect(processRow).toHaveClass(/border-border-strong/, { timeout: 5_000 });
  });
});

test.describe('Turn Off All Highlights', () => {
  test('selecting a node dims others, button clears it', async ({ page }) => {
    await waitForGraphLoaded(page);

    await expect(page.locator('canvas').first()).toBeVisible({ timeout: 10_000 });

    const fileItem = page.getByText('package.json').first();
    await expect(fileItem).toBeVisible({ timeout: 10_000 });
    await fileItem.click();

    const highlightToggle = page.locator('[data-testid="ai-highlights-toggle"]');
    await expect(highlightToggle).toHaveAttribute('title', 'Turn off AI-driven highlights', {
      timeout: 5_000,
    });

    await highlightToggle.click();
    await expect(highlightToggle).toHaveAttribute('title', 'Turn on AI-driven highlights', {
      timeout: 5_000,
    });
  });
});

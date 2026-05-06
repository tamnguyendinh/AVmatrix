#!/usr/bin/env node

const { spawnSync } = require('node:child_process');
const path = require('node:path');

const node = process.execPath;
const vitestEntry = path.resolve(__dirname, '..', 'node_modules', 'vitest', 'vitest.mjs');

const nativeDbTests = [
  'test/integration/lbug-core-adapter.test.ts',
  'test/integration/lbug-pool.test.ts',
  'test/integration/lbug-pool-stability.test.ts',
  'test/integration/local-backend.test.ts',
  'test/integration/local-backend-calltool.test.ts',
  'test/integration/search-core.test.ts',
  'test/integration/search-pool.test.ts',
  'test/integration/augmentation.test.ts',
  'test/integration/staleness-and-stability.test.ts',
  'test/integration/lbug-lock-retry.test.ts',
  'test/integration/api-impact-e2e.test.ts',
  'test/integration/cli-e2e.test.ts',
  'test/integration/class-impact-all-languages.test.ts',
  'test/integration/java-class-impact.test.ts',
  'test/integration/shape-check-regression.test.ts',
  'test/integration/skills-e2e.test.ts',
  'test/integration/lbug-vector-extension.test.ts',
  'test/integration/scope-audit-persistence.test.ts',
];

const forkSingleProcessTests = new Set([
  'test/integration/local-backend.test.ts',
  'test/integration/local-backend-calltool.test.ts',
  'test/integration/staleness-and-stability.test.ts',
]);

const commands = [
  ['vitest', 'run', '--project', 'default', '--pool=vmForks', '--reporter=dot'],
  ...nativeDbTests.map((file) =>
    forkSingleProcessTests.has(file)
      ? ['vitest', 'run', file, '--pool=forks', '--isolate=false', '--reporter=dot']
      : ['vitest', 'run', file, '--pool=vmForks', '--reporter=dot'],
  ),
];

for (const args of commands) {
  const vitestArgs = args[0] === 'vitest' ? args.slice(1) : args;
  console.log(`\n> node ${path.relative(process.cwd(), vitestEntry)} ${vitestArgs.join(' ')}`);
  const result = spawnSync(node, [vitestEntry, ...vitestArgs], {
    stdio: 'inherit',
    shell: false,
  });
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

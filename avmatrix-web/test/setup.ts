import { beforeEach } from 'vitest';
import '@testing-library/jest-dom/vitest';

// Reset storage between tests
beforeEach(() => {
  sessionStorage.removeItem('avmatrix-llm-settings');
  localStorage.removeItem('avmatrix-llm-settings'); // legacy key (migration)
  localStorage.removeItem('avmatrix.graphLinksVisible');
});

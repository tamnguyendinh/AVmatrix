import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OnboardingGuide } from '../../src/components/OnboardingGuide';

describe('OnboardingGuide local-only surface', () => {
  it('guides the user to start the local server without remote package fallbacks', () => {
    render(<OnboardingGuide />);

    expect(screen.getByText('Start AVmatrix locally')).toBeInTheDocument();
    expect(screen.getByText('cd avmatrix && npm run serve')).toBeInTheDocument();
    expect(screen.queryByText(/npm run --prefix avmatrix serve/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/npx avmatrix serve/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/avmatrix@latest/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/npm install -g avmatrix/i)).not.toBeInTheDocument();
    expect(screen.queryByText('Start avmatrix locally')).not.toBeInTheDocument();
    expect(screen.getByText('Start local bridge')).toBeInTheDocument();
  });
});

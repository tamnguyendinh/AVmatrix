import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OnboardingGuide } from '../../src/components/OnboardingGuide';

describe('OnboardingGuide local-only surface', () => {
  it('guides the user to start the local server without remote package fallbacks', () => {
    render(<OnboardingGuide />);

    expect(screen.getByText('Start GitNexus locally')).toBeInTheDocument();
    expect(screen.getByText('cd gitnexus && npm run serve')).toBeInTheDocument();
    expect(screen.queryByText(/npm run --prefix gitnexus serve/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/npx gitnexus serve/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/gitnexus@latest/i)).not.toBeInTheDocument();
    expect(screen.getByText('Start local bridge')).toBeInTheDocument();
  });
});

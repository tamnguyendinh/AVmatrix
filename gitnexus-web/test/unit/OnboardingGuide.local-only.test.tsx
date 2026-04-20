import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OnboardingGuide } from '../../src/components/OnboardingGuide';

describe('OnboardingGuide local-only surface', () => {
  it('guides the user to start the local server without remote package fallbacks', () => {
    render(<OnboardingGuide />);

    expect(screen.getByText('Start your local server')).toBeInTheDocument();
    expect(screen.getByText(/gitnexus serve/)).toBeInTheDocument();
    expect(screen.queryByText(/gitnexus@latest/i)).not.toBeInTheDocument();
  });
});

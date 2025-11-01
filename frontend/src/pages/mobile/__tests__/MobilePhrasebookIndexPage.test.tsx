import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import MobilePhrasebookIndexPage from '../MobilePhrasebookIndexPage';
import { ThemeProvider } from '../../../contexts/ThemeContext';
import * as phrasebookUtils from '../../../utils/phrasebook';

// Mock the phrasebook utilities
vi.mock('../../../utils/phrasebook', () => ({
  getAllCategories: vi.fn(),
}));

const mockCategories = [
  {
    id: 'greetings',
    name: 'Greetings',
    emoji: '👋',
    description: 'Common greetings and salutations',
  },
  {
    id: 'food',
    name: 'Food & Dining',
    emoji: '🍽️',
    description: 'Food related vocabulary',
  },
  {
    id: 'travel',
    name: 'Travel',
    emoji: '✈️',
    description: 'Travel related phrases',
  },
];

const renderComponent = () => {
  return render(
    <BrowserRouter>
      <ThemeProvider>
        <MantineProvider>
          <MobilePhrasebookIndexPage />
        </MantineProvider>
      </ThemeProvider>
    </BrowserRouter>
  );
};

describe('MobilePhrasebookIndexPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(phrasebookUtils.getAllCategories).mockClear();
  });

  it('renders the page title', async () => {
    vi.mocked(phrasebookUtils.getAllCategories).mockResolvedValue(
      mockCategories
    );

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Phrasebook')).toBeInTheDocument();
    });
  });

  it('loads and displays all categories', async () => {
    vi.mocked(phrasebookUtils.getAllCategories).mockResolvedValue(
      mockCategories
    );

    renderComponent();

    await waitFor(() => {
      expect(phrasebookUtils.getAllCategories).toHaveBeenCalled();
    });

    await waitFor(() => {
      expect(screen.getByText('Greetings')).toBeInTheDocument();
      expect(
        screen.getByText('Common greetings and salutations')
      ).toBeInTheDocument();
      expect(screen.getByText('Food & Dining')).toBeInTheDocument();
      expect(screen.getByText('Travel')).toBeInTheDocument();
    });
  });
});

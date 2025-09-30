import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import AIFixModal from './AIFixModal';

describe('AIFixModal', () => {
  it('renders original and suggestion and calls onApply', async () => {
    const original = {
      content: {
        passage: 'Passage text about apples.',
        question: 'Old?',
      },
    };
    const suggestion = {
      content: {
        passage: 'Passage text about apples.',
        question: 'New?',
      },
    };
    const onClose = vi.fn();
    const onApply = vi.fn();

    render(
      <MantineProvider>
        <AIFixModal
          opened={true}
          original={original}
          suggestion={suggestion}
          onClose={onClose}
          onApply={onApply}
        />
      </MantineProvider>
    );

    // Modal title should render and Apply button should be present
    expect(screen.getByText(/AI Suggestion/i)).toBeInTheDocument();
    // Ensure the passage and question text are present in the modal
    const passageNodes = screen.getAllByText(/Passage text about apples\./);
    expect(passageNodes.length).toBeGreaterThanOrEqual(1);
    // ReactDiffViewer may split JSON into multiple nodes; search entire body text
    const bodyText = document.body.textContent || '';
    expect(/Old\?/.test(bodyText)).toBe(true);
    // The JSON diff must not include the change_reason key (it's shown above the diff only)
    const modalContainer = passageNodes[0].closest('div');
    const modalText = modalContainer?.textContent || '';
    expect(modalText).not.toContain('"change_reason"');

    // Click Apply
    const apply = screen.getByRole('button', { name: /Apply Suggestion/i });
    fireEvent.click(apply);
    expect(onApply).toHaveBeenCalled();
  });
});

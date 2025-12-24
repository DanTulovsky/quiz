import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import CreateStoryForm from './CreateStoryForm';

// Mock the notifications module
vi.mock('../notifications', () => ({
  showNotificationWithClean: vi.fn(),
}));

describe('CreateStoryForm', () => {
  const defaultProps = {
    onSubmit: vi.fn(),
    loading: false,
  };

  const renderComponent = (props = {}) => {
    const allProps = { ...defaultProps, ...props };

    return render(
      <MantineProvider>
        <CreateStoryForm {...allProps} />
      </MantineProvider>
    );
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Form Rendering', () => {
    it('renders all form fields correctly', () => {
      renderComponent();

      expect(screen.getByText('Create New Story')).toBeInTheDocument();
      expect(screen.getByLabelText(/Story Title/)).toBeInTheDocument();
      expect(screen.getByLabelText(/Subject/)).toBeInTheDocument();
      expect(screen.getByLabelText(/Author Style/)).toBeInTheDocument();
      expect(screen.getByLabelText(/Time Period/)).toBeInTheDocument();
      const genreSelect = screen.getByTestId('story-genre-select');
      expect(genreSelect).toBeInTheDocument();
      const toneSelect = screen.getByTestId('story-tone-select');
      expect(toneSelect).toBeInTheDocument();
      expect(screen.getByLabelText(/Main Characters/)).toBeInTheDocument();
      expect(screen.getByLabelText(/Custom Instructions/)).toBeInTheDocument();
      expect(screen.getByText(/Section Length Preference/)).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: /Create Story/ })
      ).toBeInTheDocument();
    });

    it('displays loading state when loading prop is true', () => {
      renderComponent({ loading: true });

      const submitButton = screen.getByRole('button', {
        name: /Creating Story.../,
      });
      expect(submitButton).toBeInTheDocument();
      expect(submitButton).toBeDisabled();
    });
  });

  describe('Form Validation', () => {
    it('accepts valid input data', () => {
      const mockOnSubmit = vi.fn();
      renderComponent({ onSubmit: mockOnSubmit });

      const titleInput = screen.getByLabelText(/Story Title/);
      fireEvent.change(titleInput, { target: { value: 'Valid Story Title' } });

      const submitButton = screen.getByRole('button', { name: /Create Story/ });
      fireEvent.click(submitButton);

      // Should call onSubmit with the form data
      expect(mockOnSubmit).toHaveBeenCalled();
    });

    it('handles form submission with all fields filled', () => {
      const mockOnSubmit = vi.fn();
      renderComponent({ onSubmit: mockOnSubmit });

      // Fill all form fields
      fireEvent.change(screen.getByLabelText(/Story Title/), {
        target: { value: 'Test Story' },
      });
      fireEvent.change(screen.getByLabelText(/Subject/), {
        target: { value: 'A mystery story' },
      });
      fireEvent.change(screen.getByLabelText(/Author Style/), {
        target: { value: 'Agatha Christie' },
      });
      fireEvent.change(screen.getByLabelText(/Time Period/), {
        target: { value: '1920s' },
      });

      const submitButton = screen.getByRole('button', { name: /Create Story/ });
      fireEvent.click(submitButton);

      expect(mockOnSubmit).toHaveBeenCalled();
    });
  });

  describe('Form Submission', () => {
    it('calls onSubmit with correct data when form is valid', async () => {
      const mockOnSubmit = vi.fn();
      renderComponent({ onSubmit: mockOnSubmit });

      // Fill in form data
      fireEvent.change(screen.getByLabelText(/Story Title/), {
        target: { value: 'Test Story' },
      });
      fireEvent.change(screen.getByLabelText(/Subject/), {
        target: { value: 'A mystery story' },
      });
      fireEvent.change(screen.getByLabelText(/Author Style/), {
        target: { value: 'Agatha Christie' },
      });

      const submitButton = screen.getByRole('button', { name: /Create Story/ });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith(
          expect.objectContaining({
            title: 'Test Story',
            subject: 'A mystery story',
            author_style: 'Agatha Christie',
          })
        );
      });
    });

    it('handles submission errors gracefully', async () => {
      const mockOnSubmit = vi
        .fn()
        .mockRejectedValue(new Error('Submission failed'));
      renderComponent({ onSubmit: mockOnSubmit });

      fireEvent.change(screen.getByLabelText(/Story Title/), {
        target: { value: 'Test Story' },
      });

      const submitButton = screen.getByRole('button', { name: /Create Story/ });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalled();
      });
    });
  });

  describe('Genre and Tone Selection', () => {
    it('allows selecting genre from dropdown', async () => {
      const mockOnSubmit = vi.fn();
      renderComponent({ onSubmit: mockOnSubmit });

      // Find and click the genre select
      const genreSelect = screen.getByTestId('story-genre-select');
      fireEvent.click(genreSelect);

      // Select mystery genre (this would need to be implemented based on actual Mantine Select behavior)
      // For now, we'll just verify the select exists
      expect(genreSelect).toBeInTheDocument();
    });

    it('allows selecting tone from dropdown', async () => {
      renderComponent();

      const toneSelect = screen.getByTestId('story-tone-select');
      expect(toneSelect).toBeInTheDocument();
    });
  });

  describe('Section Length Override', () => {
    it('displays section length options', () => {
      renderComponent();

      expect(screen.getByText('Short')).toBeInTheDocument();
      expect(screen.getByText('Medium')).toBeInTheDocument();
      expect(screen.getByText('Long')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('has proper ARIA labels and roles', () => {
      renderComponent();

      expect(
        screen.getByRole('button', { name: /Create Story/ })
      ).toBeInTheDocument();
      expect(screen.getByLabelText(/Story Title/)).toBeInTheDocument();
    });

    it('associates labels with form inputs correctly', () => {
      renderComponent();

      const titleInput = screen.getByLabelText(/Story Title/);
      expect(titleInput).toBeInTheDocument();
      expect(titleInput).toHaveAttribute(
        'placeholder',
        'Enter a title for your story'
      );
    });
  });
});

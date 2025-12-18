import { render, screen } from '@testing-library/react';
import { SnippetHighlighter } from './SnippetHighlighter';
import { describe, it, expect, vi } from 'vitest';
import { Snippet } from '../api/api';
import { ThemeProvider } from '../contexts/ThemeContext';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';

// Mock dependencies
vi.mock('../api/api', () => ({
  deleteV1SnippetsId: vi.fn(),
}));

import { MantineProvider } from '@mantine/core';

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <MantineProvider>
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <ThemeProvider>{children}</ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>
    </MantineProvider>
  );
};

describe('SnippetHighlighter', () => {
  const mockSnippet: Snippet = {
    id: 1,
    original_text: 'Hello world',
    translated_text: 'Ciao mondo',
    source_language: 'en',
    target_language: 'it',
  };

  it('renders text with snippet highlighted', () => {
    render(
      <SnippetHighlighter
        text='This is a Hello world example'
        snippets={[mockSnippet]}
      />,
      { wrapper: createWrapper() }
    );

    // const snippetElement = screen.getByText('Hello world');
    // expect(snippetElement).toBeInTheDocument();

    // Check current style (expected to change)
    // The snippet is wrapped in a span with specific styles
    // We need to find the span that has the border style
    // The text might be inside a span which is inside the Popover target

    // Based on the code:
    // <Popover.Target>
    //   <span style={{ borderBottom: ... }}>{segment.text}</span>
    // </Popover.Target>

    // Debug to see the new structure
    // screen.debug();

    // The snippetElement (found by text "Hello world") might now be the parent span or one of the children if the text is split.
    // Actually, getByText("Hello world") might fail if the text is split into multiple elements.
    // It might match the parent span if it contains the text content, but usually getByText matches the leaf node or the closest element.
    // If "Hello" and "world" are in separate spans, "Hello world" text node doesn't exist as a single node.
    // But the parent span contains "Hello" + " " + "world".

    // Let's find the parent span by some other means or just find "Hello" and "world" separately.
    const hello = screen.getByText('Hello');
    const world = screen.getByText('world');

    expect(hello.style.borderBottom).toBe(
      '1px dashed var(--mantine-color-blue-6)'
    );
    expect(hello.style.paddingBottom).toBe('2px');

    expect(world.style.borderBottom).toBe(
      '1px dashed var(--mantine-color-blue-6)'
    );
    expect(world.style.paddingBottom).toBe('2px');

    // Verify the parent doesn't have display: inline-block
    // We can find the parent of 'hello'
    const parent = hello.parentElement;
    expect(parent?.style.display).not.toBe('inline-block');
    expect(parent?.style.cursor).toBe('pointer');
  });
});

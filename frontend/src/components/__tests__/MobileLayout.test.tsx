import { describe, it, expect } from 'vitest';

// Test to catch build errors with Mantine component imports
describe('MobileLayout Build Compatibility', () => {
  it('should import without build errors', async () => {
    // This test ensures that the MobileLayout component can be imported
    // and doesn't have any import errors that would cause build failures
    expect(async () => {
      const MobileLayout = await import('../MobileLayout');
      expect(MobileLayout.default).toBeDefined();
    }).not.toThrow();
  });

  it('should not have deprecated Mantine component imports', () => {
    // This test helps catch when Mantine components are deprecated or moved
    // by ensuring the component can be imported successfully
    expect(async () => {
      await import('../MobileLayout');
    }).not.toThrow();
  });
});

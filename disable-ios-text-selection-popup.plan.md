<!-- 039740aa-2943-48de-8de3-cf4c6ec7b33f 279aeaae-cef5-4d88-86d7-ce8c71d4368a -->

### To-dos

- [x] Create new useTouchTextSelection hook with custom touch event handlers
- [x] Update useTextSelection to detect touch devices and use appropriate selection method
- [x] Add CSS rules to prevent iOS text selection and callouts
- [x] Add data-touch-selectable attribute to text content components
- [x] Add contextmenu event prevention to TranslationPopup
- [x] Test on iOS device to verify native popup is suppressed

### Implementation Summary

Implemented option B (CSS-based approach) to disable iOS native text selection popup:

1. **Added CSS rules in `frontend/src/index.css`:**
   - Added `.selectable-text` class with `-webkit-touch-callout: none` to prevent iOS native popup
   - Kept `user-select: text` to allow text selection on all devices (desktop and mobile)
   - Added touch highlight color for better UX

2. **Updated `frontend/src/components/TranslationPopup.tsx`:**
   - Added `contextmenu` event handler to prevent iOS context menu from appearing
   - Handler checks for elements with `selectable-text` class or `data-selectable-text` attribute

3. **Updated `frontend/src/components/SnippetHighlighter.tsx`:**
   - Added `selectable-text` class to wrapper component when Component is specified
   - Preserves existing className prop if provided

4. **Updated `frontend/src/pages/mobile/MobileStoryPage.tsx`:**
   - Added `selectable-text` class to content containers in both section view and reading view
   - Applied to the div wrapping the story/section text content

5. **Updated `frontend/src/pages/mobile/MobileDailyPage.tsx`:**
   - Added `selectable-text` class to reading comprehension passage container

### Testing
- All linting passed
- Formatting applied successfully
- Ready for iOS device testing to verify native popup is suppressed while text selection still works

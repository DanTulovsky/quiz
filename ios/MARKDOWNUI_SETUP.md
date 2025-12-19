# MarkdownUI Setup Instructions

To use MarkdownUI for rendering markdown in the iOS app, you need to add it as a Swift Package dependency.

## Steps to Add MarkdownUI

1. Open the Quiz.xcodeproj in Xcode
2. Go to **File** → **Add Package Dependencies...**
3. Enter the package URL: `https://github.com/gonzalezreal/MarkdownUI`
4. Click **Add Package**
5. Select the **MarkdownUI** product and click **Add Package**
6. Ensure the package is added to the Quiz target

## Alternative: If the above URL doesn't work

Try this alternative MarkdownUI library:
- URL: `https://github.com/wilmaplus/MarkdownUI`

## Verification

After adding the package, the import statement `import MarkdownUI` in `Components.swift` should resolve without errors.


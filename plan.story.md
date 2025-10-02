# Story Mode Feature - Implementation Plan

**Linear Task:** WET-72 - Add Story Mode
**Status:** In Progress
**Branch:** `dant/wet-72-add-story-mode`

---

## Requirements Summary

### Core Feature
AI-powered story mode where users can read personalized, continuously generated stories in their target language at their learning level.

### Story Creation & Configuration
- **Story Parameters** (all optional except title):
  - Title (required, user-provided or AI-generated)
  - Subject (e.g., "a detective solving crimes")
  - Author style (e.g., "Hemingway", "Agatha Christie")
  - Time period (e.g., "1920s", "modern day")
  - Genre (e.g., "mystery", "romance", "sci-fi")
  - Tone (e.g., "serious", "humorous", "dramatic")
  - Character names/descriptions (freeform text)
  - Custom instructions (any additional guidance for AI)
  - Section length preference: short/medium/long (defaults based on language level if not specified)

- **AI Behavior**: If any optional fields are not provided, AI should choose appropriate values based on context

### Story Lifecycle
1. **Creation**: User creates a new story via form. This automatically becomes their "current" story. Any previous current story is set to non-current.
2. **Active State**: Only ONE story can be "current" per user at a time
3. **Section Generation**:
   - Automatic: Worker generates one new section per day for all users with active stories (same time as daily questions)
   - Manual: User can click "Generate Next Section" button (limited to once per day)
4. **Completion**: User manually marks story as "completed" OR story remains active indefinitely until archived
5. **Archiving**: User can archive stories. Archived stories are viewable but don't get new sections.
6. **Re-activation**: User can set any archived story as "current" to resume it
7. **Storage Limits**: Maximum 20 archived stories per user (configurable), unlimited sections per story

### Section Generation Rules
- **Length Configuration**:
  - Defined in `config.yaml` by CEFR level (A1, A2, B1, B2, C1, C2)
  - Three variants per level: short, medium, long (specified as target sentence counts)
  - Example: A1 short=8 sentences, A1 medium=12, A1 long=16
  - User can override default length when creating story

- **Language Level Adaptation**:
  - Each section generated at user's CURRENT language level (from user settings)
  - If user changes level mid-story, next section uses new level immediately
  - Previous sections retain their original level (tracked per section)

- **AI Context Management**:
  - Include all previous section text when generating next section
  - If story gets too long for AI context window, use smart truncation:
    - Keep recent sections in full
    - Summarize older sections
  - Maximum story length limited by AI provider's context window

- **Generation Frequency**: Once per day per story (either automatic or manual trigger)

### Comprehension Questions
- **Generation**: Generate 10-15 multiple choice questions per section (4 options each)
- **Display**: Show random 3-5 questions to user each time they view the section
- **State**: Session-only (answers persist during browser session, cleared on page reload)
- **No Persistence**: Don't save user answers to database
- **Feedback**: Show immediate feedback on correctness after answering

### User Interface

#### Desktop Features
- **No Story View**: If user has no current story, show creation form
- **Story View**: Display current story with:
  - Toggle between two view modes:
    - **Section Mode**: One section at a time with prev/next navigation
    - **Reading Mode**: All sections in scrollable view
  - Section number indicator (e.g., "Section 3 of 12")
  - Questions displayed below section content
  - "Generate Next Section" button (if allowed that day and on latest section)
  - Archive button
  - New Story button (archives current, creates new)
  - Export PDF button

#### Mobile Features
- **View-only**: Can read stories but cannot create or manage them
- **Reading and Section mode**: Reading and section-by-section navigation
- **No PDF export**
- **No story creation or archiving**

### Input Validation & Security

Since all user input is passed directly to AI, implement strict validation:

**Field Length Limits:**
- Title: 1-200 characters (required)
- Subject: 0-500 characters
- Author Style: 0-200 characters
- Time Period: 0-200 characters
- Genre: 0-100 characters
- Tone: 0-100 characters
- Character Names: 0-1000 characters
- Custom Instructions: 0-2000 characters
- Section Length Override: must be exactly "short", "medium", or "long" (if provided)

**Validation Rules:**
- Sanitize all text inputs to prevent injection attacks
- Trim whitespace
- Reject inputs with excessive special characters or control characters
- No HTML/script tags allowed
- Validate that optional fields are actually optional in API
- Validate UTF-8 encoding
- Use an off the shelf library to do this if possible.

**Rate Limiting:**
- Manual section generation: once per day per story
- Story creation: reasonable rate limit (e.g., max 5 stories per hour per user)

---

## Database Schema

### Migration: `backend/migrations/000021_add_stories.up.sql`

Create three tables:

1. **stories** table:
   - Primary key: id
   - Foreign key: user_id (references users, cascade delete)
   - Fields: title, language, subject, author_style, time_period, genre, tone, character_names, custom_instructions, section_length_override
   - Status: VARCHAR(20) CHECK (active/archived/completed), default 'active'
   - is_current: BOOLEAN (only one TRUE per user)
   - Timestamps: created_at, updated_at, last_section_generated_at
   - Constraint: UNIQUE(user_id, is_current) WHERE is_current = TRUE
   - Indexes on: user_id, status, (user_id, is_current), (user_id, status)

2. **story_sections** table:
   - Primary key: id
   - Foreign key: story_id (references stories, cascade delete)
   - Fields: section_number (1, 2, 3...), content (TEXT), language_level, word_count
   - Timestamps: generated_at, generation_date (DATE)
   - Constraints: UNIQUE(story_id, section_number), UNIQUE(story_id, generation_date)
   - Indexes on: story_id, (story_id, section_number), generation_date

3. **story_section_questions** table:
   - Primary key: id
   - Foreign key: section_id (references story_sections, cascade delete)
   - Fields: question_text, options (JSON array), correct_answer_index, explanation
   - Timestamp: created_at
   - Index on: section_id

### Migration Down: `backend/migrations/000021_add_stories.down.sql`

Drop all three tables in reverse order (questions, sections, stories).

### Update Source of Truth: `schema.sql`

Add all three table definitions to the main schema file.

---

## Configuration

### File: `config.yaml`

Add a `story_mode` section under `server:` section:

```yaml
# Story mode configuration
max_archived_per_user: 20
generation_enabled: true

# Section length defaults by CEFR level (in sentences)
# Also handle other languages like chinese and japanese that have different levels.
story_section_lengths:
  A1: { short: 8, medium: 12, long: 16 }
  A2: { short: 15, medium: 20, long: 25 }
  B1: { short: 25, medium: 35, long: 45 }
  B2: { short: 40, medium: 60, long: 80 }
  C1: { short: 60, medium: 100, long: 140 }
  C2: { short: 80, medium: 150, long: 200 }

# Number of comprehension questions
story_questions_per_section: 9  # Generate this many
story_questions_shown: 3          # Show this many randomly
```

### File: `backend/internal/config/config.go`

Add new fields to Config struct:
- StoryMaxArchivedPerUser int
- StoryGenerationEnabled bool
- StorySectionLengths map (nested structure for levels and lengths)
- StoryQuestionsPerSection int
- StoryQuestionsShown int

---

## Backend - Models

### File: `backend/internal/models/story.go` (NEW)

Define models:
- **Story**: struct with all story fields, using sql.NullString for optional fields
- **StorySection**: struct with section content and metadata
- **StorySectionQuestion**: struct with question data, includes ParseOptions() method
- **StoryStatus** constants: active, archived, completed
- **SectionLength** constants: short, medium, long
- Helper structs:
  - StoryWithSections (story + all sections)
  - StorySectionWithQuestions (section + all questions)
  - CreateStoryRequest (API request with validation tags)

---

## Backend - Services

### File: `backend/internal/services/story_service.go` (NEW)

Create StoryService with methods:

**Story Management:**
- CreateStory(ctx, userID, language, req) - validates limits, creates story, sets as current
- GetUserStories(ctx, userID, includeArchived) - list all user stories
- GetCurrentStory(ctx, userID) - get the current active story
- GetStory(ctx, storyID, userID) - get specific story (verify ownership)
- ArchiveStory(ctx, storyID, userID) - change status to archived
- CompleteStory(ctx, storyID, userID) - change status to completed
- SetCurrentStory(ctx, storyID, userID) - unset old current, set new current
- DeleteStory(ctx, storyID, userID) - hard delete (only for archived stories)

**Section Management:**
- GetStorySections(ctx, storyID) - get all sections ordered by section_number
- GetSection(ctx, sectionID, userID) - get specific section (verify story ownership)
- CreateSection(ctx, storyID, content, level, wordCount) - add new section
- GetLatestSection(ctx, storyID) - get most recent section
- GetAllSectionsText(ctx, storyID) - concatenate all section content for AI context

**Question Management:**
- GetSectionQuestions(ctx, sectionID) - get all questions for a section
- CreateSectionQuestions(ctx, sectionID, questions) - bulk insert questions
- GetRandomQuestions(ctx, sectionID, count) - get N random questions

**Generation Control:**
- CanGenerateSection(ctx, storyID) - check if generation allowed today (check last_section_generated_at)
- UpdateLastGenerationTime(ctx, storyID) - set last_section_generated_at to now

**Helper Methods:**
- getArchivedStoryCount(ctx, userID) - count archived stories for limit check
- GetSectionLengthTarget(level, lengthPref) - read from config, return target sentence count
- validateStoryOwnership(ctx, storyID, userID) - verify user owns story

**Input Validation:**
- validateStoryInput(req) - enforce all length limits and content rules
- sanitizeInput(text) - remove dangerous characters, trim whitespace

---

## Backend - AI Service Updates

### File: `backend/internal/services/templates/story_section_prompt.tmpl` (NEW)

Template structure:
- Language and level parameters
- Target sentence count
- Conditionally include: subject, author style, time period, genre, tone, characters, custom instructions
- If first section: instructions to introduce story
- If continuation: include previous sections text, instructions to continue
- Requirements: write at specified level, engaging content, end with minor cliffhanger

### File: `backend/internal/services/templates/story_questions_prompt.tmpl` (NEW)

Template structure:
- Language and level parameters
- Section content
- Number of questions to generate
- Requirements: 4 options each, test comprehension, appropriate difficulty
- Return JSON array format with schema

### File: `backend/internal/services/ai_service.go`

Add new methods:
- **GenerateStorySection**(ctx, userConfig, StoryGenerationRequest) -> string
  - Build prompt from template using story parameters and previous sections
  - Call AI API
  - Return generated section text
  - Handle token limits (truncate or summarize if needed)

- **GenerateStoryQuestions**(ctx, userConfig, StoryQuestionsRequest) -> []StorySectionQuestionData
  - Build prompt from template with section content
  - Call AI API with JSON schema
  - Parse response into question structs
  - Validate question format

Add new request/response types:
- StoryGenerationRequest (all story params + previous sections text)
- StoryQuestionsRequest (language, level, section content, question count)
- StorySectionQuestionData (question, options array, correct_answer_index, explanation)

Add helper method:
- **truncateStoryContext**(sections, maxTokens) - implement smart truncation logic

---

## Backend - Worker Updates

### File: `backend/internal/worker/worker.go`

**In run() method:**
Add call to `checkForStoryGenerations(ctx)` after daily questions check.

**New methods:**

- **checkForStoryGenerations**(ctx) error
  - Get all users with current active stories
  - For each user:
    - Get current story
    - Check if can generate section today
    - If yes, call generateStorySection()
  - Log successes and failures

- **generateStorySection**(ctx, user, story) error
  - Get all previous sections for story
  - Get user's current language level from user settings
  - Determine target length (from story override or config default)
  - Get user's AI config (provider, model, API key)
  - Build StoryGenerationRequest with all story params and previous text
  - Call AIService.GenerateStorySection()
  - Count words in result
  - Save section to database with CreateSection()
  - Generate questions: call AIService.GenerateStoryQuestions()
  - Save questions with CreateSectionQuestions()
  - Update story.last_section_generated_at
  - Handle errors: log, don't crash worker

- **getUsersWithActiveStories**(ctx) ([]User, error)
  - Query users table joined with stories where is_current = TRUE
  - Return users who have AI enabled and API keys configured

**Add StoryService to worker struct:**
Wire up in worker initialization.

---

## Backend - API Handlers

### File: `backend/internal/handlers/story_handler.go` (NEW)

Create StoryHandler with dependencies: StoryService, AIService, UserService, Logger.

**Endpoints:**

1. **POST /v1/story** - CreateStory
   - Get userID from session
   - Bind JSON request, validate input
   - Get user's language setting
   - Call StoryService.CreateStory()
   - Return created story (201)

2. **GET /v1/story** - GetUserStories
   - Query param: include_archived (boolean)
   - Get userID from session
   - Call StoryService.GetUserStories()
   - Return array of stories (200)

3. **GET /v1/story/current** - GetCurrentStory
   - Get userID from session
   - Call StoryService.GetCurrentStory()
   - Call StoryService.GetStorySections()
   - Return StoryWithSections (200) or 404 if no current story

4. **GET /v1/story/:id** - GetStory
   - Parse storyID from path
   - Get userID from session
   - Call StoryService.GetStory() (includes ownership check)
   - Call StoryService.GetStorySections()
   - Return StoryWithSections (200)

5. **GET /v1/story/section/:id** - GetSection
   - Parse sectionID from path
   - Get userID from session
   - Call StoryService.GetSection() (includes ownership check)
   - Call StoryService.GetSectionQuestions()
   - Return StorySectionWithQuestions (200)

6. **POST /v1/story/:id/generate** - GenerateNextSection
   - Parse storyID, get userID
   - Verify ownership
   - Check CanGenerateSection() - return error if already generated today
   - Get story and user details
   - Build AI request, call generateStorySection() (similar to worker logic)
   - Return new section (201)
   - Apply rate limiting

7. **POST /v1/story/:id/archive** - ArchiveStory
   - Parse storyID, get userID
   - Call StoryService.ArchiveStory()
   - Return success (200)

8. **POST /v1/story/:id/complete** - CompleteStory
   - Parse storyID, get userID
   - Call StoryService.CompleteStory()
   - Return success (200)

9. **POST /v1/story/:id/set-current** - SetCurrentStory
   - Parse storyID, get userID
   - Verify story exists and user owns it
   - Call StoryService.SetCurrentStory()
   - Return success (200)

10. **DELETE /v1/story/:id** - DeleteStory
    - Parse storyID, get userID
    - Verify story is archived or completed (not current)
    - Call StoryService.DeleteStory()
    - Return success (204)

11. **GET /v1/story/:id/export** - ExportStory
    - Parse storyID, get userID
    - Get story and all sections
    - Use PDF library (github.com/jung-kurt/gofpdf) to generate PDF
    - Set response headers (Content-Type: application/pdf, Content-Disposition)
    - Stream PDF to response

**Error Handling:**
- Use HandleAppError for all errors
- Wrap errors with context
- Return appropriate HTTP status codes

### File: `backend/internal/handlers/router_factory.go`

Add story routes to router under `/v1/story` group:
- Apply RequireAuth middleware
- Apply RequestValidationMiddleware
- Wire up all StoryHandler endpoints

### File: `backend/internal/di/container.go`

Add StoryService and StoryHandler to DI container:
- Initialize StoryService with db, logger, config
- Initialize StoryHandler with services
- Register handler in container

---

## Frontend - API Client

### File: `frontend/src/api/storyApi.ts` (NEW)

Create API client functions (use generated types after swagger update):
- createStory(data: CreateStoryRequest)
- getUserStories(includeArchived?: boolean)
- getCurrentStory()
- getStory(storyId: number)
- getSection(sectionId: number)
- generateNextSection(storyId: number)
- archiveStory(storyId: number)
- completeStory(storyId: number)
- setCurrentStory(storyId: number)
- deleteStory(storyId: number)
- exportStoryPDF(storyId: number) - download file

---

## Frontend - Hooks

### File: `frontend/src/hooks/useStory.ts` (NEW)

Custom hook that manages story state:

**State:**
- currentStory (Story | null)
- sections (StorySection[])
- isLoading (boolean)
- canGenerateToday (boolean)
- error (string | null)

**Effects:**
- On mount: fetch current story and sections

**Methods:**
- fetchCurrentStory() - load current story data
- createStory(data) - create new story, refresh
- archiveStory(storyId) - archive, refresh
- completeStory(storyId) - mark complete, refresh
- generateNextSection(storyId) - trigger generation, wait, refresh
- setCurrentStory(storyId) - activate archived story, refresh
- exportStoryPDF(storyId) - download PDF file
- refreshSections() - reload sections for current story

**Return:**
Export all state and methods for components to use.

---

## Frontend - Components

### File: `frontend/src/components/CreateStoryForm.tsx` (NEW)

Form component with fields:
- Title (TextInput, required, maxLength 200)
- Subject (Textarea, optional, maxLength 500)
- Author Style (TextInput, optional, maxLength 200)
- Time Period (TextInput, optional, maxLength 200)
- Genre (Select, optional, predefined options)
- Tone (Select, optional, predefined options)
- Character Names (Textarea, optional, maxLength 1000)
- Custom Instructions (Textarea, optional, maxLength 2000)
- Section Length (SegmentedControl: short/medium/long, optional)

**Validation:**
- Client-side validation for all length limits
- Show character count for long fields
- Required field indicators
- Submit button disabled until valid

**Behavior:**
- onSubmit callback with validated data
- Loading state during submission
- Error display if submission fails

### File: `frontend/src/components/StorySectionView.tsx` (NEW)

Component to display a single section:
- Section content (Paper component, formatted text with line breaks)
- Language level badge
- Word/sentence count
- Comprehension questions:
  - Get random subset on component mount
  - Use session storage to persist answers during session
  - StoryQuestionCard components
- "Generate Next Section" button (conditional):
  - Only show if canGenerateNext prop is true
  - Disable during generation
  - Show loading spinner when generating

### File: `frontend/src/components/StoryReadingView.tsx` (NEW)

Component to display all sections in reading mode:
- Stack of all sections
- Each section in a Paper with subtle divider
- Section numbers as headers
- Scrollable container
- No questions shown in this view (reading focus)

### File: `frontend/src/components/StoryQuestionCard.tsx` (NEW)

Simpler question component (no persistence):
- Display question text
- Radio group with 4 options
- Submit button
- Show immediate feedback after answering
- Show explanation
- Track state only in component (no API calls)
- Use session storage for persistence during session

### File: `frontend/src/components/StoryArchiveModal.tsx` (NEW)

Modal to browse and manage archived stories:
- List of archived stories with title, language, section count
- Click to view archived story (read-only)
- Button to set as current (re-activate)
- Button to delete permanently
- Search/filter by language

---

## Frontend - Pages

### File: `frontend/src/pages/StoryPage.tsx` (NEW)

Main story page for desktop:

**State:**
- viewMode: 'section' | 'reading'
- currentSectionIndex: number
- showCreateModal: boolean
- showArchiveModal: boolean

**Layout:**
1. **No Current Story State:**
   - Show welcome message
   - Show CreateStoryForm inline

2. **Has Current Story State:**
   - Header:
     - Story title
     - Action buttons: Export PDF, Archive, New Story, View Archives
   - View mode toggle: Section-by-section vs Reading mode
   - Navigation (section mode only): prev/next buttons, section indicator
   - Content area:
     - StorySectionView (section mode) OR StoryReadingView (reading mode)

**Modals:**
- Create new story modal (CreateStoryForm)
- Archive browser modal (StoryArchiveModal)

**Effects:**
- Load current story on mount
- Reset to section 1 when story changes

### File: `frontend/src/pages/mobile/MobileStoryPage.tsx` (NEW)

Simplified mobile page:
- View-only (no creation, no archiving)
- Always in reading mode
- Show all sections in scrollable view
- No questions displayed
- Simple navigation

---

## Frontend - Navigation & Routing

### File: `frontend/src/components/Layout.tsx`

Add to mainNav array (around line 86):
```tsx
{
  name: 'Story',
  href: '/story',
  icon: IconBook, // Import from @tabler/icons-react
  testId: 'nav-story',
}
```

Add keyboard shortcut (Shift+5 for story).

### File: `frontend/src/components/MobileLayout.tsx`

Add to navItems:
```tsx
{ key: 'story', label: 'Story', icon: IconBook, path: '/m/story' }
```

### File: `frontend/src/App.tsx`

Add routes:
- `/story` - desktop StoryPage with Layout
- `/m/story` - mobile MobileStoryPage with MobileLayout
- Both require authentication, redirect to login if not authenticated

Import StoryPage and MobileStoryPage at top.

---

## Swagger API Documentation

### File: `swagger.yaml`

Add new tag: `Story`

Add paths:
- POST /v1/story (create)
- GET /v1/story (list)
- GET /v1/story/current (get current)
- GET /v1/story/{id} (get specific)
- GET /v1/story/section/{id} (get section)
- POST /v1/story/{id}/generate (manual generate)
- POST /v1/story/{id}/archive (archive)
- POST /v1/story/{id}/complete (complete)
- POST /v1/story/{id}/set-current (set current)
- DELETE /v1/story/{id} (delete)
- GET /v1/story/{id}/export (export PDF)

Add schemas:
- Story
- StorySection
- StorySectionQuestion
- CreateStoryRequest
- StoryWithSections
- StorySectionWithQuestions

**After updating swagger.yaml:**
Run `task generate-api-types` to generate TypeScript types.

---

## Testing Strategy

### Backend Unit Tests

**File: `backend/internal/services/story_service_test.go`**
- Test CreateStory with various inputs
- Test archived story limit enforcement
- Test one current story per user constraint
- Test ArchiveStory / SetCurrentStory logic
- Test GetSectionLengthTarget with config
- Test CanGenerateSection date logic
- Test input validation and sanitization

**File: `backend/internal/handlers/story_handler_test.go`**
- Test all endpoints with valid inputs
- Test authorization (ownership checks)
- Test error cases (not found, forbidden)
- Test rate limiting on manual generation

### Backend Integration Tests

**File: `backend/internal/services/story_service_integration_test.go`**
- Test full story creation flow with database
- Test section generation and question insertion
- Test unique constraints (current story, section numbers)
- Test cascade deletes

**File: `backend/internal/worker/worker_integration_test.go`**
- Test checkForStoryGenerations with test data
- Test generateStorySection end-to-end
- Test worker doesn't generate twice in one day

### Frontend Tests

**File: `frontend/src/pages/StoryPage.test.tsx`**
- Test renders create form when no story
- Test renders story content when story exists
- Test view mode toggle
- Test section navigation
- Test action buttons (archive, new story, export)

**File: `frontend/src/components/CreateStoryForm.test.tsx`**
- Test form validation (required fields, length limits)
- Test character count displays
- Test submission with valid data
- Test error display

**File: `frontend/src/components/StorySectionView.test.tsx`**
- Test section content display
- Test random question selection
- Test session state persistence
- Test generate button conditional display

**File: `frontend/src/hooks/useStory.test.tsx`**
- Test hook loads current story
- Test createStory updates state
- Test archiveStory refreshes data
- Test error handling

### E2E Tests

**File: `frontend/tests/story.spec.ts`**

Test scenarios:
1. Create new story:
   - Login
   - Navigate to /story
   - Fill create form (all fields)
   - Submit
   - Verify story appears with title

2. View and navigate sections:
   - Navigate to story with sections
   - Toggle to section mode
   - Use prev/next buttons
   - Verify section content changes

3. Answer questions:
   - View section
   - Answer questions
   - Verify feedback appears
   - Reload page, verify session cleared

4. Toggle view modes:
   - Switch to reading mode
   - Verify all sections visible
   - Switch back to section mode

5. Archive and re-activate:
   - Archive current story
   - Verify redirected to create form
   - Open archives
   - Set story as current
   - Verify story loaded

---

## Implementation Checklist

### Phase 1: Database & Backend Foundation (4-6 hours)
- [ ] Create migration files (up and down)
- [ ] Run migration locally and verify schema
- [ ] Update schema.sql source of truth
- [ ] Create story models in models/story.go
- [ ] Add story config to config.yaml
- [ ] Update config struct to parse story settings
- [ ] Write StoryService with all methods
- [ ] Write unit tests for StoryService
- [ ] Run tests: `task test-go-unit`

### Phase 2: AI Integration (4-5 hours)
- [ ] Create story_section_prompt.tmpl template
- [ ] Create story_questions_prompt.tmpl template
- [ ] Add GenerateStorySection method to AIService
- [ ] Add GenerateStoryQuestions method to AIService
- [ ] Implement smart truncation logic for long stories
- [ ] Test AI generation manually with sample data
- [ ] Write unit tests for AI methods

### Phase 3: Worker Integration (3-4 hours)
- [ ] Add StoryService to worker struct
- [ ] Implement checkForStoryGenerations in worker
- [ ] Implement generateStorySection in worker
- [ ] Implement getUsersWithActiveStories
- [ ] Add worker call in run() method
- [ ] Test worker locally with test data
- [ ] Write integration tests for worker story generation

### Phase 4: API Layer (4-5 hours)
- [ ] Create StoryHandler with all endpoints
- [ ] Add input validation in handler methods
- [ ] Add story routes to router_factory.go
- [ ] Wire up StoryService and StoryHandler in DI container
- [ ] Update swagger.yaml with all story endpoints and schemas
- [ ] Run `task generate-api-types`
- [ ] Write handler unit tests
- [ ] Test API manually with curl/Postman

### Phase 5: Frontend Core (6-8 hours)
- [ ] Create storyApi.ts with all API methods
- [ ] Create useStory hook
- [ ] Create CreateStoryForm component
- [ ] Create StorySectionView component
- [ ] Create StoryReadingView component
- [ ] Create StoryQuestionCard component
- [ ] Create StoryPage (desktop)
- [ ] Add Story nav item to Layout
- [ ] Add /story route to App.tsx
- [ ] Test manually in browser

### Phase 6: Frontend Features (3-4 hours)
- [ ] Implement PDF export functionality
- [ ] Create StoryArchiveModal component
- [ ] Add archive management to StoryPage
- [ ] Implement view mode toggle
- [ ] Implement section navigation
- [ ] Add session storage for questions
- [ ] Create MobileStoryPage (view-only)
- [ ] Add mobile navigation and routes

### Phase 7: Testing (5-6 hours)
- [ ] Write StoryService integration tests
- [ ] Write StoryHandler integration tests
- [ ] Write frontend component tests
- [ ] Write useStory hook tests
- [ ] Write E2E tests for full user flow
- [ ] Run full test suite: `task test`
- [ ] Fix any test failures

### Phase 8: Polish & Validation (3-4 hours)
- [ ] Run `task lint` and fix all issues
- [ ] Test input validation with edge cases
- [ ] Test rate limiting (once per day generation)
- [ ] Test archived story limits
- [ ] Verify worker automatic generation
- [ ] Test PDF export with various story lengths
- [ ] Test mobile view-only mode
- [ ] Verify all error messages are user-friendly

### Phase 9: Documentation (2-3 hours)
- [ ] Update WARP.md with Story mode section
- [ ] Update README.md with Story feature description
- [ ] Add comments to complex logic
- [ ] Create sample story data for test database
- [ ] Update API_CONTRACT.md if needed

### Phase 10: Deployment (1-2 hours)
- [ ] Make git commit before changes
- [ ] Commit all changes with clear message
- [ ] Push to branch
- [ ] Test on staging/test environment
- [ ] Create PR with detailed description
- [ ] Address review feedback
- [ ] Merge and deploy to production

---

## Dependencies to Add

### Backend
```bash
go get github.com/jung-kurt/gofpdf
```

For PDF generation. Alternative: `github.com/signintech/gopdf`

### Frontend
No new dependencies required. Using existing:
- Mantine UI components
- React Router
- Axios (via generated API client)

---

## Open Questions & Design Decisions

1. **PDF Export Library**: Using `jung-kurt/gofpdf` - simple, well-maintained Go library

2. **Character Names Storage**: Store as plain text (comma-separated or freeform). Let AI parse it. Simpler than JSON.

3. **Story Title Auto-generation**: If title is empty, return validation error. Require title from user. Simpler than AI generation.

4. **Section Numbering Display**: Show "Section N" in UI. Simple and language-neutral.

5. **Worker Priority**: Run story generation AFTER daily questions. Questions are higher priority for user engagement.

6. **Rate Limiting**: Manual generation counts against per-user AI concurrency limits. Prevents abuse.

7. **Smart Truncation Implementation**:
   - Calculate tokens for each section
   - If total > 80% of context window:
     - Keep last 5 sections in full
     - For older sections: create brief summaries (1-2 sentences per section)
     - Include summaries in prompt
   - Track which sections were truncated for transparency

8. **Session Storage Key Format**: `story_section_${sectionId}_questions_${timestamp}`

9. **PDF Styling**: Simple black text on white, story title as header, section numbers, basic formatting. Blank lines between paragraphs.

10. **Error Messages**: User-friendly for all validation errors (e.g., "Title must be between 1 and 200 characters")

---

## Estimated Effort

- **Phase 1**: 4-6 hours
- **Phase 2**: 4-5 hours
- **Phase 3**: 3-4 hours
- **Phase 4**: 4-5 hours
- **Phase 5**: 6-8 hours
- **Phase 6**: 3-4 hours
- **Phase 7**: 5-6 hours
- **Phase 8**: 3-4 hours
- **Phase 9**: 2-3 hours
- **Phase 10**: 1-2 hours

**Total**: ~35-47 hours (5-6 full working days)

---

## Success Criteria

✅ User can create a story with custom parameters
✅ Worker automatically generates sections daily
✅ User can manually trigger generation once per day
✅ Sections adapt to user's current language level
✅ Comprehension questions appear with each section
✅ Questions use session-only state (no persistence)
✅ User can toggle between section and reading modes
✅ User can archive and re-activate stories
✅ Only one current story per user enforced
✅ Archived story limit enforced (20 default)
✅ PDF export works for complete stories
✅ Mobile view-only mode functional
✅ All input validation working with strict limits
✅ All tests pass: `task test`
✅ No linter errors: `task lint`
✅ API documentation complete in swagger.yaml

---

## Notes

- Follow existing patterns from DailyQuestionService for section assignment logic
- Reuse AI service patterns from question generation for story generation
- Use session storage for question state (similar to chat history approach)
- Ensure all user input is sanitized before passing to AI
- Add OpenTelemetry traces for story generation operations
- Consider adding analytics: story engagement metrics, popular genres, average story length
- Future enhancement: Allow users to "fork" a story at any section to explore alternative paths
- Future enhancement: Social features (share stories, collaborative stories)


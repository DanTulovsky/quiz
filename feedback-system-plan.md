# Generic Report Issues / Feedback System - Implementation Plan

## Executive Summary

This plan outlines the implementation of a generic, context-aware feedback and issue reporting system that allows users to report issues or provide feedback from any page in the application. The system will capture contextual information (page, language, level, question, etc.) and optionally screenshots, storing everything in the database with an admin interface for review.

## Current State Analysis

**Existing Functionality:**
- The app already has a question-specific report feature (`/v1/quiz/question/{id}/report`)
- Database table `question_reports` exists for question-specific reports
- Report button is embedded in QuestionCard component
- No admin view exists for reviewing question reports

**Gap Analysis:**
The existing system is limited to reporting issues with specific questions. We need a **generic** system that:
- Works from any page (not just questions)
- Is always accessible (top bar button)
- Captures rich contextual metadata
- Supports optional screenshots
- Provides admin management interface

## Library Research

### Feedback/Bug Reporting Libraries

After researching Mantine-compatible solutions, here are the options:

#### Option 1: Build Custom Solution (RECOMMENDED)
**Pros:**
- Full control over UI/UX
- Perfect Mantine integration
- No external dependencies
- Can leverage existing patterns in codebase
- Lighter weight

**Cons:**
- More development time
- Need to implement screenshot capture ourselves

**Screenshot Libraries Compatible with React:**
- `html2canvas` (52k+ stars, MIT license) - Most popular, well-maintained
- `dom-to-image` (10k+ stars, MIT license) - Alternative option
- Browser native `navigator.mediaDevices.getDisplayMedia()` - For full screen capture

**Recommendation:** Use `html2canvas` for screenshot capture.

#### Option 2: Third-Party Services
Services like Sentry, LogRocket, or Bugsnag offer feedback widgets but:
- Require external accounts/billing
- Less customizable
- Data stored externally (privacy concerns)
- Overkill for our needs

**RECOMMENDATION: Build custom solution using Mantine components + html2canvas for screenshots**

## Architecture Design

### 1. Database Schema

Create a new table `feedback_reports` to store generic feedback (separate from question-specific reports):

```sql
CREATE TABLE IF NOT EXISTS feedback_reports (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    
    -- Feedback content
    feedback_text TEXT NOT NULL,
    feedback_type VARCHAR(50) DEFAULT 'general', -- 'bug', 'feature_request', 'general', 'improvement'
    
    -- Context metadata (JSON for flexibility)
    context_data JSONB NOT NULL DEFAULT '{}',
    
    -- Screenshot (stored as base64 or reference to file storage)
    screenshot_data TEXT, -- Base64 encoded image or NULL
    screenshot_url TEXT,  -- Alternative: URL to stored image
    
    -- Admin management
    status VARCHAR(50) DEFAULT 'new', -- 'new', 'in_progress', 'resolved', 'dismissed'
    admin_notes TEXT,
    assigned_to_user_id INTEGER,
    resolved_at TIMESTAMPTZ,
    resolved_by_user_id INTEGER,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign keys
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_to_user_id) REFERENCES users (id) ON DELETE SET NULL,
    FOREIGN KEY (resolved_by_user_id) REFERENCES users (id) ON DELETE SET NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_feedback_reports_user_id ON feedback_reports(user_id);
CREATE INDEX IF NOT EXISTS idx_feedback_reports_status ON feedback_reports(status);
CREATE INDEX IF NOT EXISTS idx_feedback_reports_created_at ON feedback_reports(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feedback_reports_type ON feedback_reports(feedback_type);
CREATE INDEX IF NOT EXISTS idx_feedback_reports_context_data ON feedback_reports USING GIN(context_data);
```

**Context Data Structure (JSONB):**
```json
{
  "page_url": "/quiz",
  "page_title": "Quiz",
  "language": "italian",
  "level": "A2",
  "question_id": 123,
  "story_id": 456,
  "section_id": 789,
  "viewport_width": 1920,
  "viewport_height": 1080,
  "user_agent": "Mozilla/5.0...",
  "timestamp": "2025-10-27T15:30:00Z",
  "additional_info": {}
}
```

### 2. Backend API Design

#### New Endpoints in swagger.yaml:

```yaml
/v1/feedback:
  post:
    tags:
      - Feedback
    summary: Submit feedback or report an issue
    description: Submit feedback or bug report with optional screenshot
    security:
      - cookieAuth: []
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - feedback_text
            properties:
              feedback_text:
                type: string
                maxLength: 5000
                description: Feedback or issue description
              feedback_type:
                type: string
                enum: [bug, feature_request, general, improvement]
                default: general
              context_data:
                type: object
                description: Context metadata as JSON
              screenshot_data:
                type: string
                description: Base64 encoded screenshot (optional)
    responses:
      '201':
        description: Feedback submitted successfully
      '400':
        description: Invalid request
      '401':
        description: Unauthorized

/v1/admin/feedback:
  get:
    tags:
      - Admin
      - Feedback
    summary: Get all feedback reports (paginated)
    description: Retrieve feedback reports with filtering and pagination
    security:
      - cookieAuth: []
    parameters:
      - name: page
        in: query
        schema:
          type: integer
          default: 1
      - name: page_size
        in: query
        schema:
          type: integer
          default: 20
          maximum: 100
      - name: status
        in: query
        schema:
          type: string
          enum: [new, in_progress, resolved, dismissed]
      - name: feedback_type
        in: query
        schema:
          type: string
      - name: user_id
        in: query
        schema:
          type: integer
    responses:
      '200':
        description: List of feedback reports
      '401':
        description: Unauthorized
      '403':
        description: Forbidden (admin only)

/v1/admin/feedback/{id}:
  get:
    tags:
      - Admin
      - Feedback
    summary: Get feedback report details
    security:
      - cookieAuth: []
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: integer
    responses:
      '200':
        description: Feedback report details
      '404':
        description: Not found
        
  patch:
    tags:
      - Admin
      - Feedback
    summary: Update feedback report status
    security:
      - cookieAuth: []
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: integer
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            properties:
              status:
                type: string
                enum: [new, in_progress, resolved, dismissed]
              admin_notes:
                type: string
              assigned_to_user_id:
                type: integer
    responses:
      '200':
        description: Updated successfully
      '404':
        description: Not found
```

#### Backend Implementation Structure:

**Files to create/modify:**

1. `backend/internal/handlers/feedback_handler.go` - New handler
2. `backend/internal/services/feedback_service.go` - New service
3. `backend/internal/models/feedback.go` - New model
4. `backend/migrations/000035_add_feedback_reports.up.sql` - Migration
5. `backend/migrations/000035_add_feedback_reports.down.sql` - Rollback
6. Update `backend/internal/handlers/router_factory.go` - Register routes

**Handler Methods:**
- `SubmitFeedback(c *gin.Context)` - Submit feedback
- `GetFeedbackReports(c *gin.Context)` - Admin: List feedback (paginated)
- `GetFeedbackReport(c *gin.Context)` - Admin: Get single feedback
- `UpdateFeedbackReport(c *gin.Context)` - Admin: Update status/notes

**Service Methods:**
- `CreateFeedback(ctx, userID, text, type, context, screenshot)` - Create feedback
- `GetFeedbackReportsPaginated(ctx, filters, pagination)` - Get list
- `GetFeedbackReportByID(ctx, id)` - Get single
- `UpdateFeedbackReport(ctx, id, updates)` - Update report
- `GetFeedbackStats(ctx)` - Stats for admin dashboard

### 3. Frontend Implementation

#### 3.1 Feedback Button in Layout

Add a feedback button to the top bar in `Layout.tsx` (and `AdminLayout.tsx`, mobile layouts):

```tsx
<Tooltip label='Report Issue or Give Feedback' position='bottom' withArrow>
  <ActionIcon
    onClick={() => setFeedbackModalOpened(true)}
    variant='subtle'
    size='lg'
    aria-label='Feedback'
    data-testid='feedback-button'
  >
    <IconBug size={20} /> {/* or IconMessageReport */}
  </ActionIcon>
</Tooltip>
```

**Position:** Between Help icon and Settings icon in the top bar.

#### 3.2 Feedback Modal Component

Create `frontend/src/components/FeedbackModal.tsx`:

**Features:**
- Textarea for feedback (required, max 5000 chars)
- Dropdown for feedback type (bug, feature request, general, improvement)
- Checkbox for "Include screenshot" (default: unchecked)
- Preview of captured screenshot with ability to recapture
- Automatically captures context information
- Keyboard shortcuts (Escape to close, Ctrl+Enter to submit)
- Loading state during submission
- Success/error notifications

**Props:**
```tsx
interface FeedbackModalProps {
  opened: boolean;
  onClose: () => void;
}
```

**Context Capture Logic:**
```typescript
const captureContext = () => {
  return {
    page_url: window.location.pathname + window.location.search,
    page_title: document.title,
    language: user?.preferred_language,
    level: user?.current_level,
    question_id: getCurrentQuestionId(), // from context if available
    story_id: getCurrentStoryId(), // from context if available
    viewport_width: window.innerWidth,
    viewport_height: window.innerHeight,
    user_agent: navigator.userAgent,
    timestamp: new Date().toISOString(),
  };
};
```

**Screenshot Capture:**
```typescript
import html2canvas from 'html2canvas';

const captureScreenshot = async (): Promise<string | null> => {
  try {
    const canvas = await html2canvas(document.body, {
      logging: false,
      useCORS: true,
      allowTaint: true,
      scale: 0.5, // Reduce size
    });
    return canvas.toDataURL('image/jpeg', 0.7); // Compress as JPEG
  } catch (error) {
    console.error('Screenshot capture failed:', error);
    return null;
  }
};
```

#### 3.3 API Integration

Create `frontend/src/api/feedback.ts`:

```typescript
export const useSubmitFeedback = () => {
  return useMutation({
    mutationFn: async (data: FeedbackSubmission) => {
      const response = await axios.post('/v1/feedback', data);
      return response.data;
    },
  });
};
```

Update Orval config to regenerate types after swagger.yaml update.

#### 3.4 Admin Feedback Management Page

Create `frontend/src/pages/admin/FeedbackManagementPage.tsx`:

**Features:**
- Paginated list of feedback reports (using Mantine Table)
- Filters: status, type, user, date range
- Search by feedback text
- Click on row to open detail modal
- Status badges (New, In Progress, Resolved, Dismissed)
- Stats cards at top (total, new, in progress, resolved)

**Detail Modal:**
- Full feedback text
- Screenshot viewer (if available)
- Context data display (formatted JSON)
- User information
- Status update dropdown
- Admin notes textarea
- Assignment dropdown
- Action buttons (Save, Close)

**Layout:**
```tsx
<Container size='xl' py='xl'>
  <Stack gap='xl'>
    <Header with stats />
    <FilterBar />
    <DataTable with pagination />
  </Stack>
</Container>
```

Add link to Admin page navigation:
```tsx
{
  title: 'Feedback Reports',
  description: 'Review user feedback and issue reports',
  icon: <IconBug size={24} />,
  path: '/admin/feedback',
  color: 'red',
}
```

### 4. Screenshot Handling Strategy

**Options for Storage:**

1. **Base64 in Database (Simple, Recommended for MVP)**
   - Pros: Simple, no external storage needed
   - Cons: Increases DB size, slower queries
   - Mitigation: Limit screenshot size (compress JPEG at 0.7 quality, scale 0.5)
   - Max size estimate: ~200-500KB per screenshot

2. **File Storage (Future Enhancement)**
   - Store screenshots as files (local filesystem or S3)
   - Store URL/path in database
   - Requires additional infrastructure

**Recommendation:** Start with Base64 in database for MVP, migrate to file storage if needed.

### 5. Mobile Support

- Add feedback button to mobile layouts (`MobileLayout.tsx`)
- Ensure modal is mobile-responsive
- Screenshot capture should work on mobile browsers
- Consider simpler UI on mobile (fewer options)

## Implementation Phases

### Phase 1: Backend Foundation (2-3 hours)
1. Create database migration for `feedback_reports` table
2. Create models, service layer, and handlers
3. Add routes to router
4. Write unit tests for service layer
5. Update swagger.yaml and regenerate API types

### Phase 2: Frontend Feedback Submission (3-4 hours)
1. Install `html2canvas` dependency
2. Create `FeedbackModal` component
3. Add feedback button to all layouts (desktop, admin, mobile)
4. Implement context capture logic
5. Implement screenshot capture
6. Add API integration
7. Write component tests

### Phase 3: Admin Interface (3-4 hours)
1. Create `FeedbackManagementPage`
2. Create detail modal component
3. Add filters and search
4. Add pagination
5. Implement status updates
6. Add admin stats
7. Write integration tests

### Phase 4: Testing & Polish (2-3 hours)
1. End-to-end tests using Playwright
2. Test screenshot capture on different browsers
3. Test mobile responsiveness
4. Performance testing (large screenshots)
5. Error handling improvements
6. Documentation

**Total Estimated Time: 10-14 hours**

## Testing Strategy

### Unit Tests
- Service layer methods (CRUD operations)
- Context capture utility functions
- Screenshot compression logic

### Integration Tests
- API endpoints (submit, get, update)
- Database operations
- Authentication/authorization

### E2E Tests (Playwright)
- Submit feedback from different pages
- Submit with/without screenshot
- Admin view and update feedback
- Keyboard shortcuts
- Mobile submission

### Manual Testing Checklist
- [ ] Test from all major pages (quiz, story, daily, etc.)
- [ ] Test screenshot capture on Chrome, Firefox, Safari
- [ ] Test mobile browsers (iOS Safari, Android Chrome)
- [ ] Test with/without screenshot
- [ ] Test admin pagination with large dataset
- [ ] Test context data accuracy
- [ ] Test screenshot size/quality trade-offs

## Security Considerations

1. **Input Validation:**
   - Sanitize feedback text (max 5000 chars)
   - Validate screenshot size (max 5MB)
   - Validate context_data JSON structure

2. **Authorization:**
   - Require authentication for feedback submission
   - Require admin role for feedback management
   - Users can only view their own feedback (optional feature)

3. **Rate Limiting:**
   - Limit feedback submissions (e.g., 5 per hour per user)
   - Prevent abuse/spam

4. **Data Privacy:**
   - Don't capture sensitive information in context
   - Blur sensitive data in screenshots (optional enhancement)
   - GDPR compliance: allow users to delete their feedback

## Performance Considerations

1. **Screenshot Size:**
   - Compress to JPEG at 0.7 quality
   - Scale to 0.5 of original size
   - Target: < 500KB per screenshot
   - Consider image optimization service for further compression

2. **Database:**
   - Index on frequently queried fields
   - Consider archiving old resolved feedback
   - Monitor database size growth

3. **Frontend:**
   - Lazy load html2canvas library
   - Show loading indicator during screenshot capture
   - Debounce screenshot preview generation

## Open Questions & Decisions Needed

1. **Screenshot Storage:**
   - Q: Start with Base64 in DB or implement file storage from the start?
   - **Recommendation:** Base64 in DB for MVP, migrate later if needed

2. **Feedback Types:**
   - Q: Are the proposed types sufficient (bug, feature_request, general, improvement)?
   - **Recommendation:** Start with these, add more based on usage

3. **User Notification:**
   - Q: Should users be notified when their feedback is resolved?
   - **Recommendation:** Not for MVP, add later if requested

4. **Public Feedback Visibility:**
   - Q: Should users be able to view all feedback (public board)?
   - **Recommendation:** Admin-only for MVP, consider public board later

5. **Integration with Existing Question Reports:**
   - Q: Should we migrate existing question reports to new system or keep separate?
   - **Recommendation:** Keep separate - they serve different purposes

6. **Screenshot Required/Optional:**
   - Q: Should screenshot be required, optional, or removed?
   - **Recommendation:** Optional (user can choose)

7. **Context Data Expansion:**
   - Q: What additional context should we capture?
   - **Recommendation:** Start with proposed fields, expand based on feedback

8. **Anonymization:**
   - Q: Should we offer anonymous feedback option?
   - **Recommendation:** No - require authentication for accountability

## Success Metrics

After implementation, track:
- Number of feedback submissions per week
- Feedback type distribution
- Average time to resolution
- Screenshot capture success rate
- User satisfaction with feedback process

## Alternative Approaches Considered

### Alternative 1: Use Existing Question Report System
- **Rejected:** Too limited, not context-aware, not accessible from all pages

### Alternative 2: Third-Party Service (Sentry, LogRocket)
- **Rejected:** External dependencies, privacy concerns, cost, overkill

### Alternative 3: Simple Email Form
- **Rejected:** No structured data, no admin interface, not integrated

## Migration Path for Existing Question Reports

The existing `question_reports` table should remain as-is because:
1. It serves a specific purpose (question quality)
2. Different data model (linked to specific question)
3. May have different workflow/handling

**Optional Enhancement:** Add admin view for question reports in the future, separate from generic feedback.

## Conclusion

This plan proposes a custom-built, Mantine-integrated feedback system that:
- ✅ Is accessible from any page via top bar button
- ✅ Captures rich contextual metadata
- ✅ Supports optional screenshots
- ✅ Provides comprehensive admin interface
- ✅ Integrates seamlessly with existing architecture
- ✅ Maintains separation from question-specific reports

The implementation is straightforward, leveraging existing patterns in the codebase and well-established libraries (html2canvas). Total estimated time: 10-14 hours.

**Next Steps:**
1. Review and approve this plan
2. Confirm design decisions (particularly screenshot storage)
3. Prioritize implementation phases
4. Begin Phase 1 implementation

---

**Questions or feedback on this plan? Please comment below!**

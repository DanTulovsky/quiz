# WET-87 Implementation Plan: Vocabulary List Feature

## Task Overview
**WET-87**: Implement a vocabulary list feature that allows users to save individual words and phrases from the quiz application, leveraging existing translation functionality.

## Current Status
- Translation popup already exists for selected text
- Need to add save functionality to translation popup
- Need database schema for vocabulary items
- Need new API endpoints for CRUD operations
- Need frontend UI for browsing/editing vocabulary list

## Implementation Plan


### Phase 1: Database Schema Design
- [ ] Design vocabulary table structure:
  - id (primary key)
  - user_id (foreign key)
  - original_text (the selected text)
  - translated_text (translation result)
  - language_code (source/target languages)
  - created_at, updated_at timestamps
  - metadata (id of the question where the lookup was made, optional context, difficulty level, etc...)
- [ ] Create database migration
- [ ] Update source of truth SQL file

### Phase 2: Backend API Development
- [ ] Add new API endpoints to swagger.yaml:
  - `POST /api/vocabulary` - Save vocabulary item
  - `GET /api/vocabulary` - List user's vocabulary items
  - `PUT /api/vocabulary/{id}` - Update vocabulary item
  - `DELETE /api/vocabulary/{id}` - Remove vocabulary item
- [ ] Generate API types using `task generate-api-types`
- [ ] Implement backend handlers:
  - Vocabulary service layer
  - HTTP handlers with proper error handling
  - Database operations with transactions
  - Input validation and sanitization

### Phase 3: Frontend Translation Popup Enhancement
- [ ] Add save button to existing translation popup
- [ ] Implement save functionality:
  - Extract selected text and translation
  - Call vocabulary API to save item
  - Show success feedback
  - Handle error cases
- [ ] Update popup UI to accommodate save button

### Phase 4: Vocabulary Management UI
- [ ] Create new vocabulary management page/section called Snippets:
  - Paginated list view of saved vocabulary items
  - Search and filter functionality
  - Edit modal for each item
  - Delete confirmation
  - Pagination for large lists
- [ ] Add navigation to vocabulary section
- [ ] Implement responsive design using Mantine components

### Phase 5: Integration & Testing
- [ ] Unit tests for backend services
- [ ] Integration tests for API endpoints
- [ ] Frontend unit tests for new components
- [ ] End-to-end tests for vocabulary workflow
- [ ] Test translation popup save functionality

### Phase 6: Documentation & Polish
- [ ] Update API documentation
- [ ] Add user-facing help text
- [ ] Performance optimization (lazy loading, caching)
- [ ] Accessibility improvements
- [ ] Mobile responsiveness

## Dependencies
- Existing translation functionality must be working
- User authentication system
- Database connectivity

## Risk Assessment
- **Medium Risk**: Integration with existing translation popup could require careful refactoring
- **Low Risk**: Database schema is straightforward user data
- **Medium Risk**: UI/UX needs to integrate seamlessly with existing design

## Success Metrics
- Users can successfully save vocabulary items from translation popup
- Vocabulary list displays all saved items with search/filter capability
- Edit and delete operations work correctly
- Performance remains acceptable with large vocabulary lists
- All tests pass

## Git Strategy
- Create feature branch: `dant/wet-87-vocabulary-list` (already exists in Linear)
- Implement in logical commits
- Merge to main after testing and approval

## Timeline Estimate
- Phase 1: 1-2 days
- Phase 2: 2-3 days
- Phase 3: 1-2 days
- Phase 4: 3-4 days
- Phase 5: 2-3 days
- Phase 6: 1-2 days

**Total: 10-16 days** (allowing for iteration and testing)

---

*Ready for implementation once approved. This plan leverages existing translation work and follows established patterns in the codebase.*

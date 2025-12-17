# Spec and build

## Configuration
- **Artifacts Path**: {@artifacts_path} → `.zenflow/tasks/{task_id}`

---

## Agent Instructions

Ask the user questions when anything is unclear or needs their input. This includes:
- Ambiguous or incomplete requirements
- Technical decisions that affect architecture or user experience
- Trade-offs that require business context

Do not make assumptions on important decisions — get clarification first.

---

## Workflow Steps

### [x] Step: Technical Specification
<!-- chat-id: 95fa88c5-fb9c-42b7-a7f8-77d0e85ab852 -->

Assess the task's difficulty, as underestimating it leads to poor outcomes.
- easy: Straightforward implementation, trivial bug fix or feature
- medium: Moderate complexity, some edge cases or caveats to consider
- hard: Complex logic, many caveats, architectural considerations, or high-risk changes

Create a technical specification for the task that is appropriate for the complexity level:
- Review the existing codebase architecture and identify reusable components.
- Define the implementation approach based on established patterns in the project.
- Identify all source code files that will be created or modified.
- Define any necessary data model, API, or interface changes.
- Describe verification steps using the project's test and lint commands.

Save the output to `{@artifacts_path}/spec.md` with:
- Technical context (language, dependencies)
- Implementation approach
- Source code structure changes
- Data model / API / interface changes
- Verification approach

If the task is complex enough, create a detailed implementation plan based on `{@artifacts_path}/spec.md`:
- Break down the work into concrete tasks (incrementable, testable milestones)
- Each task should reference relevant contracts and include verification steps
- Replace the Implementation step below with the planned tasks

Rule of thumb for step size: each step should represent a coherent unit of work (e.g., implement a component, add an API endpoint, write tests for a module). Avoid steps that are too granular (single function).

Save to `{@artifacts_path}/plan.md`. If the feature is trivial and doesn't warrant this breakdown, keep the Implementation step below as is.

---

### [x] Step: Set up Xcode Project
<!-- chat-id: 1640a17f-48a2-4922-a89c-4dab741c6603 -->

- Create a new directory `ios/` in the root.
- Initialize a new Xcode project for a Swift/SwiftUI application inside `ios/`.
- Set up the basic project configuration (bundle ID, signing, etc.).
- Create the directory structure for MVVM (Models, Views, ViewModels, Services).

### [ ] Step: Implement API Service and Models
<!-- chat-id: c0b0a70f-1acc-475e-8e20-a7502689e117 -->

- Create a networking layer (`APIService.swift`) to handle API requests using `URLSession`.
- Define Swift `Codable` structs for all the API objects based on `swagger.yaml`.
- Implement helper functions for common API calls (e.g., authentication, fetching questions).

### [ ] Step: Implement Authentication

- Build the SwiftUI views for Login and Signup.
- Create a view model to handle user input and interaction with the `APIService`.
- Implement secure storage for authentication tokens (e.g., using Keychain).
- Implement session management to keep the user logged in.

### [ ] Step: Build Core UI Shell

- Implement the main navigation for the app after login. A `TabView` is a good candidate for this.
- Create placeholder views for each of the main feature areas.

### [ ] Step: Implement Daily Quiz Feature

- Build the UI for displaying questions.
- Create a view model to manage the state of a quiz session.
- Implement the logic for submitting answers and displaying feedback.

### [ ] Step: Implement Learning Modules

- **Stories**: View a list of stories and read a story.
- **Vocabulary**: View and manage vocabulary lists.
- **Phrasebook**: Browse the phrasebook.
- **Translation Practice**: Implement the translation practice UI.
- **Verb Conjugation**: Implement the verb conjugation UI.
- For each module, create the necessary views and view models.

### [ ] Step: Implement Settings

- Build the settings screen.
- Allow users to view and update their preferences.

### [ ] Step: Testing and Refinement

- Write unit tests for view models and services.
- Write UI tests for key user flows.
- Run SwiftLint and fix any style issues.
- Polish the UI and user experience.

### [ ] Step: Final Report

- After completion, write a report to `{@artifacts_path}/report.md` describing:
   - What was implemented
   - How the solution was tested
   - The biggest issues or challenges encountered

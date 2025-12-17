# Technical Specification: Native iOS App

## Task Assessment

- **Difficulty**: Hard

Building a native iOS application that mirrors the functionality of an existing mobile web application is a complex task. It requires a deep understanding of the existing application's features and APIs, as well as expertise in native iOS development. This document outlines the technical approach for this project.

## 1. Technical Context

- **Programming Language**: Swift
- **UI Framework**: SwiftUI
- **Architecture**: MVVM (Model-View-ViewModel)
- **API Communication**: The app will interact with the existing Go backend via the REST API defined in `swagger.yaml`.
- **Dependency Management**: Swift Package Manager (SPM) will be used for any third-party libraries.

## 2. Implementation Approach

The new iOS application will be a standalone project, developed in Xcode. It will replicate the features of the mobile web UI, providing a fully native user experience.

### 2.1. Project Setup

- A new Xcode project will be created.
- The project will be configured for Swift and SwiftUI.
- A directory structure will be established to support the MVVM architecture (e.g., `Models`, `Views`, `ViewModels`, `Services`).

### 2.2. Feature Implementation

The features identified in the `frontend/src/pages/mobile` directory will be implemented as native SwiftUI views. Each feature will have its own set of views, view models, and models.

**Core Features to be implemented:**

- **Authentication**:
  - Login / Signup screens.
  - Handling of authentication tokens (e.g., secure storage in Keychain).
  - Google OAuth integration.
- **Main App Experience**:
  - A tab-based navigation structure seems appropriate to house the main features.
  - **Daily Exercises/Quizzes**: A central part of the app, allowing users to answer questions.
  - **Learning Modules**: Reading Comprehension, Translation Practice, Verb Conjugation, Vocabulary, Stories, Phrasebook. Each will have a dedicated UI.
  - **Progress Tracking**: Visual representation of user progress.
  - **Saved Content**: Access to bookmarked messages, conversations, and snippets.
  - **Settings**: A screen for user-configurable settings.

### 2.3. Networking Layer

- A dedicated networking service (e.g., `APIService.swift`) will be created to handle all communication with the backend.
- This service will use `URLSession` to make API calls.
- Codable models will be created to represent the JSON request and response bodies defined in `swagger.yaml`. This will provide type safety when interacting with the API. For example, `LoginRequest`, `LoginResponse`, `Question`, `AnswerRequest`, etc., will have corresponding Swift structs.

## 3. Source Code Structure Changes

A new directory, `ios/`, will be created at the root of the repository to house the Xcode project for the iOS application.

```
/
├── backend/
├── frontend/
├── ios/
│   ├── QuizApp/
│   │   ├── Models/
│   │   ├── Views/
│   │   ├── ViewModels/
│   │   ├── Services/
│   │   ├── Resources/
│   │   └── QuizApp.xcodeproj
...
```

No changes are anticipated for the `backend/` or `frontend/` codebases, as the iOS app will be a new client consuming the existing API.

## 4. Data Model / API / Interface Changes

- **No backend API changes** are expected. The iOS app will be built to conform to the existing API contract specified in `swagger.yaml`.
- The primary work will be to create Swift `struct`s that are `Codable` and match the JSON objects defined in the API. This allows for easy decoding of API responses into native Swift types.

## 5. Verification Approach

- **Unit Tests**: Business logic within the ViewModels will be tested using XCTest.
- **UI Tests**: SwiftUI Previews will be used for rapid UI development and verification. XCUITest can be used for end-to-end UI testing flows.
- **Linting**: SwiftLint will be integrated into the project to enforce a consistent code style.
- **Build & Run**: The application will be regularly built and run on the iOS Simulator and physical devices to ensure functionality and performance.

This specification provides a high-level overview of the plan to build the native iOS app. Given the complexity, the next step is to break this down into a more detailed implementation plan.

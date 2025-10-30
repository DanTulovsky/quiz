# Project Instructions

## General

- Don't be so eager to assume I am right. Don't say things like "You're absolutely right!".  Think critically about my comments. Don't be so chipper.
- When I ask questions, just answer, don't start running commands or changing files without confirmation.
- Ask for confirmation if not sure, but don't overdo. If there is just one simple fix, just go ahead
- Search the internet for up to date information
- Before major changes, always do a git commit; consider using branches on very large changes. Don't forget to merge back into main.
- You are not done until all tests pass!
- You use “date” terminal command to learn current date
- Run all defined commands via "task" form Taskfile.yaml rather than directly (e.g. go, npx, etc...)

## Architecture

- Follow established patterns in the codebase with respect to tracing, error handling, etc...
- Keep business logic in service layers

## API

- When writing openapi files, always use $ref references

## Code

- Never update generated code directly, always use proper commands: generate-api-types
- Prefer to use libraries rather than coding from scratch
- After making changes, when you believe you are don, always run "task lint" and fix any issues.
- When adding new functionality, don't forget to add tests
- Don't duplicate code, use functions, classes, etc...
- Don't remove existing features without confirmation

### Frontend

- Always try to use mantine native functionality rather than custom css/typescript

### Backend

-

## Project and Tasks

- Get tasks from Linear when available.  While working on tasks, make sure to set the correct status.  Add updtes to the tasks as you progress through them.  Mark tasks as done only after I confirm they are done.
- When updating Linear, always update by adding a comment, do not overwrite the original task!

## Tests

- To run backend and worker golang unit tests: task test-go-unit
- To run backend and worker golang integration tests: task test-go-integration (DO NOT run go test -tage=integration ... it won't work!)
- To run backend and worker golang tests (both unit and integration): task test-go
- To run frontend unit tests: task test-frontend
- To run end-to-end (e2e) tests: task test-e2e
- To run end-to-end API tests: task test-e2e-api
- Look in Taskfile.yaml for variants on running single file or test

- To run ALL TESTS: task test

- To run end-to-end (e2e) tests and ask a human for help: task test-e2e-manual-debug-ui

FORMAT and LINT

- To format files run: task format
- To lint files run: task lint
- To find dead code run: task deadcode

TO Regenerate API types after changing swagger.yaml

- task generate-api-types

## Database

- When making database migrations, don't forget to also edit the source of truth sql file if one exists.

.DEFAULT_GOAL := _reminder

%:
	@echo "==============================================="
	@echo "⚠️  This project has migrated from Make to Task!"
	@echo "==============================================="
	@echo ""
	@echo "Please use 'task' instead of 'make':"
	@echo ""
	@if command -v task >/dev/null 2>&1; then \
		echo "Available tasks:"; \
		echo ""; \
		task --list --sort alphabetical 2>/dev/null | tail -n +2 | sed 's/^/  /' || \
		(echo "  task           # Run default task"; \
		 echo "  task --list    # Show all available tasks"); \
	else \
		echo "  (Install Task to see available tasks)"; \
		echo ""; \
		echo "Common tasks after installation:"; \
		echo "  task           # Run default task"; \
		echo "  task test      # Run tests"; \
		echo "  task build     # Build project"; \
		echo "  task clean     # Clean up"; \
	fi
	@echo ""
	@echo "Install Task from: https://taskfile.dev/installation/"
	@echo "==============================================="
	@exit 1

#!/usr/bin/env python3
"""
Generate Load Testing Users
==========================

This script generates 100 load testing users with varied attributes for comprehensive
load testing scenarios. The users are added to backend/data/test_users.yaml and include:

- 100 users with usernames: loaduser001-loaduser100
- Varied languages: italian, spanish, french, german, english
- Different proficiency levels: A1, A2, B1, B2
- Multiple AI providers: ollama, openai, google, anthropic
- Appropriate AI models for each provider

Usage:
    cd backend/data
    python ../../scripts/generate_users.py

Requirements:
    pip install pyyaml

The script preserves existing users and appends the load testing users.
Each user has realistic attributes for testing different scenarios including
language learning preferences, AI provider configurations, and skill levels.

The script also provides statistics on the distribution of languages,
levels, and AI providers to ensure balanced test data.
"""

import yaml
import random

# User configuration options
LANGUAGES = ["italian", "spanish", "french", "german", "english"]
LEVELS = ["A1", "A2", "B1", "B2"]
AI_PROVIDERS = ["ollama", "openai", "google", "anthropic"]

def generate_load_users():
    """Generate 100 load testing users with varied attributes."""

    # Read existing users
    with open("test_users.yaml", "r") as f:
        existing_data = yaml.safe_load(f)

    existing_users = existing_data.get("users", [])
    print(f"Found {len(existing_users)} existing users")

    # Generate 100 load testing users
    load_users = []
    for i in range(1, 101):
        user_number = str(i).zfill(3)

        # Randomly select attributes
        language = random.choice(LANGUAGES)
        level = random.choice(LEVELS)
        ai_provider = random.choice(AI_PROVIDERS)

        # Create user with varied attributes
        user = {
            "username": f"loaduser{user_number}",
            "email": f"loaduser{user_number}@example.com",
            "password": "password",
            "preferred_language": language,
            "current_level": level,
            "ai_provider": ai_provider,
            "ai_api_key": "",
            "api_key": f"load-api-key-{user_number}"
        }

        # Add AI model for some providers
        if ai_provider == "openai":
            user["ai_model"] = "gpt-4"
        elif ai_provider == "google":
            user["ai_model"] = "gemini-2.0-flash"
        elif ai_provider == "anthropic":
            user["ai_model"] = "claude-3-sonnet"

        load_users.append(user)

    # Add load users to existing users
    all_users = existing_users + load_users

    # Write back to file
    with open("test_users.yaml", "w") as f:
        yaml.dump({"users": all_users}, f, default_flow_style=False, sort_keys=False)

    print(f"Generated {len(load_users)} load testing users")
    print(f"Total users: {len(all_users)}")

    # Print statistics
    language_counts = {}
    level_counts = {}
    provider_counts = {}

    for user in load_users:
        lang = user["preferred_language"]
        level = user["current_level"]
        provider = user["ai_provider"]

        language_counts[lang] = language_counts.get(lang, 0) + 1
        level_counts[level] = level_counts.get(level, 0) + 1
        provider_counts[provider] = provider_counts.get(provider, 0) + 1

    print("\nUser Distribution:")
    print("Languages:", language_counts)
    print("Levels:", level_counts)
    print("AI Providers:", provider_counts)

if __name__ == "__main__":
    generate_load_users()

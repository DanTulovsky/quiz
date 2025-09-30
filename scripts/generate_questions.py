#!/usr/bin/env python3
"""
Generate Test Questions for Load Testing
========================================

This script generates 500+ test questions using lorem ipsum text for load testing.
The questions are added to backend/data/test_questions.yaml and include:

- Multiple question types: vocabulary, fill_blank, qa, reading_comprehension
- Various languages: italian, spanish, french, german
- Different difficulty levels: A1, A2, B1, B2
- Multiple topics: vocabulary, grammar, conversation, culture, etc.
- Random assignment to load testing users (loaduser001-loaduser100)

Usage:
    cd backend/data
    python ../../scripts/generate_questions.py

Requirements:
    pip install lorem pyyaml

The script preserves existing questions and appends new ones.
Each question includes realistic metadata like difficulty scores, usage counts,
and user assignments for comprehensive load testing scenarios.
"""

import yaml
import random
from lorem import sentence, paragraph

# Question types and their templates
QUESTION_TYPES = [
    "vocabulary",
    "fill_blank",
    "qa",
    "reading_comprehension"
]

LANGUAGES = ["italian", "spanish", "french", "german"]
LEVELS = ["A1", "A2", "B1", "B2"]
TOPICS = ["vocabulary", "grammar", "conversation", "culture", "travel", "food", "sports", "history"]

def generate_question(id_num):
    """Generate a single question with lorem ipsum content."""

    question_type = random.choice(QUESTION_TYPES)
    language = random.choice(LANGUAGES)
    level = random.choice(LEVELS)
    topic = random.choice(TOPICS)

    # Generate base content
    question_text = sentence()
    explanation = sentence()

    # Generate options
    options = [sentence() for _ in range(4)]
    correct_answer = random.randint(0, 3)

    # Create question structure
    question = {
        "type": question_type,
        "language": language,
        "level": level,
        "topic": topic,
        "difficulty_score": round(random.uniform(0.1, 0.9), 1),
        "content": {
            "question": question_text,
            "options": options,
            "correct_answer": correct_answer,
            "explanation": explanation
        },
        "usage_count": random.randint(0, 10),
        "status": "active",
        "users": []
    }

    # Add users (randomly assign to some load users)
    num_users = random.randint(1, 5)
    load_users = [f"loaduser{str(i).zfill(3)}" for i in range(1, 101)]
    selected_users = random.sample(load_users, num_users)
    question["users"] = selected_users

    # Add specific content for different question types
    if question_type == "fill_blank":
        question["content"]["hint"] = sentence()

    elif question_type == "reading_comprehension":
        question["content"]["passage"] = paragraph()
        question["content"]["question"] = sentence()

    return question

def main():
    """Generate 500+ questions and append to existing test_questions.yaml."""

    # Read existing questions
    with open("test_questions.yaml", "r") as f:
        existing_data = yaml.safe_load(f)

    existing_questions = existing_data.get("questions", [])
    print(f"Found {len(existing_questions)} existing questions")

    # Generate new questions
    new_questions = []
    for i in range(500):
        question = generate_question(i + len(existing_questions))
        new_questions.append(question)

        if (i + 1) % 50 == 0:
            print(f"Generated {i + 1} questions...")

    # Combine existing and new questions
    all_questions = existing_questions + new_questions

    # Write back to file
    with open("test_questions.yaml", "w") as f:
        yaml.dump({"questions": all_questions}, f, default_flow_style=False, sort_keys=False)

    print(f"Generated {len(new_questions)} new questions")
    print(f"Total questions: {len(all_questions)}")

if __name__ == "__main__":
    main()

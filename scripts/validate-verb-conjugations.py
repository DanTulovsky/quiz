#!/usr/bin/env python3
"""
validate-verb-conjugations.py
Validates verb conjugation JSON files for:
1. Valid JSON syntax
2. Consistent tenses across all verbs within each language file
"""

import json
import os
import sys
from pathlib import Path
from typing import Dict, List, Set, Any


class Colors:
    """ANSI color codes for terminal output"""
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    NC = '\033[0m'  # No Color


def print_status(status: str, message: str) -> None:
    """Print colored status messages"""
    color_map = {
        "SUCCESS": Colors.GREEN + "✅ ",
        "ERROR": Colors.RED + "❌ ",
        "WARNING": Colors.YELLOW + "⚠️  ",
        "INFO": Colors.BLUE + "ℹ️  ",
    }

    color = color_map.get(status, "")
    print(f"{color}{message}{Colors.NC}")


def validate_json_syntax(file_path: Path) -> tuple[bool, str]:
    """Validate JSON syntax of a file"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            json.load(f)
        return True, "Valid JSON syntax"
    except json.JSONDecodeError as e:
        return False, f"Invalid JSON syntax: {e}"
    except Exception as e:
        return False, f"Error reading file: {e}"


def validate_tense_consistency(file_path: Path) -> tuple[bool, str, List[str]]:
    """Validate that all verbs in a file have the same tenses"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)

        # Check basic structure
        if 'verbs' not in data:
            return False, "No 'verbs' key found in file", []

        verbs = data['verbs']
        if not verbs:
            return False, "No verbs found in file", []

        # Get tense IDs from the first verb as reference
        first_verb = verbs[0]
        if 'tenses' not in first_verb:
            return False, f"First verb ('{first_verb.get('infinitive', 'unknown')}') has no 'tenses' key", []

        reference_tense_ids = set()
        for tense in first_verb['tenses']:
            if 'tenseId' not in tense:
                return False, f"First verb has tense missing 'tenseId'", []
            reference_tense_ids.add(tense['tenseId'])

        # Check each verb has the same tenses
        errors = []
        for i, verb in enumerate(verbs):
            verb_name = verb.get('infinitive', f'verb_{i+1}')

            if 'tenses' not in verb:
                errors.append(f"Verb '{verb_name}' missing 'tenses' key")
                continue

            verb_tense_ids = set()
            for tense in verb['tenses']:
                if 'tenseId' not in tense:
                    errors.append(f"Verb '{verb_name}' has tense missing 'tenseId'")
                    continue
                verb_tense_ids.add(tense['tenseId'])

            if verb_tense_ids != reference_tense_ids:
                missing_in_verb = reference_tense_ids - verb_tense_ids
                extra_in_verb = verb_tense_ids - reference_tense_ids

                error_msg = f"Verb '{verb_name}' has inconsistent tenses"
                if missing_in_verb:
                    error_msg += f" (missing: {sorted(missing_in_verb)})"
                if extra_in_verb:
                    error_msg += f" (extra: {sorted(extra_in_verb)})"

                errors.append(error_msg)

        if errors:
            return False, "Inconsistent tenses between verbs", errors

        tense_list = sorted(reference_tense_ids)
        return True, f"All verbs have consistent tenses ({len(tense_list)} tenses)", tense_list

    except Exception as e:
        return False, f"Error validating file: {e}", []


def validate_required_fields(file_path: Path) -> tuple[bool, str, List[str]]:
    """Validate that all required fields are present and non-empty"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)

        if 'verbs' not in data:
            return False, "No 'verbs' key found in file", []

        verbs = data['verbs']
        if not verbs:
            return False, "No verbs found in file", []

        errors = []

        # Check each verb
        for i, verb in enumerate(verbs):
            verb_name = verb.get('infinitive', f'verb_{i+1}')

            # Check verb-level required fields
            required_verb_fields = ['infinitive', 'infinitiveEn', 'category', 'tenses']
            for field in required_verb_fields:
                if field not in verb:
                    errors.append(f"Verb '{verb_name}' missing required field '{field}'")
                elif not verb[field] or (isinstance(verb[field], str) and verb[field].strip() == ""):
                    errors.append(f"Verb '{verb_name}' has empty '{field}' field")

            # Check tenses
            if 'tenses' in verb and isinstance(verb['tenses'], list):
                for j, tense in enumerate(verb['tenses']):
                    tense_name = tense.get('tenseId', f'tense_{j+1}')

                    # Check tense-level required fields
                    required_tense_fields = ['tenseId', 'tenseName', 'tenseNameEn', 'description', 'conjugations']
                    for field in required_tense_fields:
                        if field not in tense:
                            errors.append(f"Verb '{verb_name}' tense '{tense_name}' missing required field '{field}'")
                        elif not tense[field] or (isinstance(tense[field], str) and tense[field].strip() == ""):
                            errors.append(f"Verb '{verb_name}' tense '{tense_name}' has empty '{field}' field")

                    # Check conjugations
                    if 'conjugations' in tense and isinstance(tense['conjugations'], list):
                        for k, conjugation in enumerate(tense['conjugations']):
                            conj_name = f"conjugation_{k+1}"

                            # Check conjugation-level required fields
                            required_conj_fields = ['pronoun', 'form', 'exampleSentence', 'exampleSentenceEn']
                            for field in required_conj_fields:
                                if field not in conjugation:
                                    errors.append(f"Verb '{verb_name}' tense '{tense_name}' {conj_name} missing required field '{field}'")
                                elif not conjugation[field] or (isinstance(conjugation[field], str) and conjugation[field].strip() == ""):
                                    # Allow "—" for unused forms
                                    if conjugation[field] != "—":
                                        errors.append(f"Verb '{verb_name}' tense '{tense_name}' {conj_name} has empty '{field}' field")

        if errors:
            return False, "Missing or empty required fields", errors

        return True, "All required fields are present and non-empty", []

    except Exception as e:
        return False, f"Error validating required fields: {e}", []


def validate_duplicate_verbs(file_path: Path) -> tuple[bool, str, List[str]]:
    """Validate that there are no duplicate verbs within a language file"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)

        if 'verbs' not in data:
            return False, "No 'verbs' key found in file", []

        verbs = data['verbs']
        if not verbs:
            return False, "No verbs found in file", []

        # Track infinitive forms and their indices
        infinitive_to_indices = {}
        for i, verb in enumerate(verbs):
            if 'infinitive' in verb:
                infinitive = verb['infinitive']
                if infinitive not in infinitive_to_indices:
                    infinitive_to_indices[infinitive] = []
                infinitive_to_indices[infinitive].append(i)

        # Check for duplicates
        errors = []
        for infinitive, indices in infinitive_to_indices.items():
            if len(indices) > 1:
                error_msg = f"Duplicate verb '{infinitive}' found at indices: {indices}"
                errors.append(error_msg)

        if errors:
            return False, "Duplicate verbs found", errors

        return True, "No duplicate verbs found", []

    except Exception as e:
        return False, f"Error validating duplicates: {e}", []


def validate_example_sentences(file_path: Path) -> tuple[bool, str, List[str]]:
    """Validate that example sentences are properly formatted"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)

        if 'verbs' not in data:
            return False, "No 'verbs' key found in file", []

        verbs = data['verbs']
        if not verbs:
            return False, "No verbs found in file", []

        errors = []

        # Check each verb's conjugations
        for i, verb in enumerate(verbs):
            verb_name = verb.get('infinitive', f'verb_{i+1}')

            if 'tenses' in verb and isinstance(verb['tenses'], list):
                for j, tense in enumerate(verb['tenses']):
                    tense_name = tense.get('tenseId', f'tense_{j+1}')

                    if 'conjugations' in tense and isinstance(tense['conjugations'], list):
                        for k, conjugation in enumerate(tense['conjugations']):
                            conj_name = f"conjugation_{k+1}"

                            # Check example sentences
                            example_sentence = conjugation.get('exampleSentence', '')
                            example_sentence_en = conjugation.get('exampleSentenceEn', '')
                            form = conjugation.get('form', '')

                            # If form is "—", then example sentences should also be "—"
                            if form == "—":
                                if example_sentence != "—" or example_sentence_en != "—":
                                    errors.append(f"Verb '{verb_name}' tense '{tense_name}' {conj_name}: when form is '—', example sentences should also be '—'")
                            else:
                                # If form is not "—", example sentences should not be "—" and should be different
                                if example_sentence == "—" or example_sentence_en == "—":
                                    errors.append(f"Verb '{verb_name}' tense '{tense_name}' {conj_name}: example sentences should not be '—' when form is not '—'")
                                elif example_sentence == example_sentence_en:
                                    errors.append(f"Verb '{verb_name}' tense '{tense_name}' {conj_name}: example sentences should be different (got same text for both languages)")

        if errors:
            return False, "Example sentence validation failed", errors

        return True, "All example sentences are properly formatted", []

    except Exception as e:
        return False, f"Error validating example sentences: {e}", []


def main():
    """Main validation function"""
    script_dir = Path(__file__).parent
    project_root = script_dir.parent
    verb_conjugations_dir = project_root / "backend" / "internal" / "handlers" / "data" / "verb-conjugations"

    print("🔍 Validating verb conjugation files...")
    print(f"Directory: {verb_conjugations_dir}")
    print()

    # Counters
    total_files = 0
    valid_files = 0
    invalid_json_files = 0
    inconsistent_files = 0
    required_fields_errors = 0
    duplicate_errors = 0
    example_sentence_errors = 0

    # Check if directory exists
    if not verb_conjugations_dir.exists():
        print_status("ERROR", f"Verb conjugations directory not found: {verb_conjugations_dir}")
        sys.exit(1)

    # Find all JSON files (excluding info.json)
    json_files = list(verb_conjugations_dir.glob("*.json"))
    json_files = [f for f in json_files if f.name != "info.json"]
    json_files.sort(key=lambda x: x.name)

    if not json_files:
        print_status("ERROR", f"No JSON files found in {verb_conjugations_dir}")
        sys.exit(1)

    print("Found verb conjugation files:")
    for file_path in json_files:
        print(f"  - {file_path.name}")
    print()

    # Validate each file
    for file_path in json_files:
        total_files += 1
        print(f"🔍 Validating {file_path.name}...")

        # Check JSON syntax
        json_valid, json_message = validate_json_syntax(file_path)
        if json_valid:
            print_status("SUCCESS", json_message)
        else:
            print_status("ERROR", json_message)
            invalid_json_files += 1
            print()
            continue

        # Check tense consistency
        consistent, consistency_message, tense_list = validate_tense_consistency(file_path)
        if consistent:
            print_status("SUCCESS", consistency_message)
            if tense_list:
                print_status("INFO", f"Tenses: {', '.join(tense_list)}")
        else:
            print_status("ERROR", consistency_message)
            for error in tense_list:
                print(f"    - {error}")
            inconsistent_files += 1

        # Check required fields
        fields_valid, fields_message, fields_errors = validate_required_fields(file_path)
        if fields_valid:
            print_status("SUCCESS", fields_message)
        else:
            print_status("ERROR", fields_message)
            for error in fields_errors:
                print(f"    - {error}")
            required_fields_errors += 1

        # Check for duplicate verbs
        no_duplicates, duplicates_message, duplicate_errors_list = validate_duplicate_verbs(file_path)
        if no_duplicates:
            print_status("SUCCESS", duplicates_message)
        else:
            print_status("ERROR", duplicates_message)
            for error in duplicate_errors_list:
                print(f"    - {error}")
            duplicate_errors += 1

        # Check example sentences
        examples_valid, examples_message, examples_errors = validate_example_sentences(file_path)
        if examples_valid:
            print_status("SUCCESS", examples_message)
        else:
            print_status("ERROR", examples_message)
            for error in examples_errors:
                print(f"    - {error}")
            example_sentence_errors += 1

        # Overall file validity
        if consistent and fields_valid and no_duplicates and examples_valid:
            valid_files += 1

        print()

    # Summary
    print("📊 Validation Summary:")
    print(f"  Total files: {total_files}")
    print(f"  Valid files: {valid_files}")
    print(f"  Invalid JSON: {invalid_json_files}")
    print(f"  Inconsistent tenses: {inconsistent_files}")
    print(f"  Required fields errors: {required_fields_errors}")
    print(f"  Duplicate verbs: {duplicate_errors}")
    print(f"  Example sentence errors: {example_sentence_errors}")

    if (invalid_json_files == 0 and inconsistent_files == 0 and
        required_fields_errors == 0 and duplicate_errors == 0 and
        example_sentence_errors == 0):
        print_status("SUCCESS", "All verb conjugation files are valid and consistent! 🎉")
        sys.exit(0)
    else:
        print_status("ERROR", "Validation failed. Please fix the issues above.")
        sys.exit(1)


if __name__ == "__main__":
    main()

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
import argparse
from pathlib import Path
from typing import Dict, List, Set, Any

# Global verbose flag
VERBOSE = False


class Colors:
    """ANSI color codes for terminal output"""

    RED = "\033[0;31m"
    GREEN = "\033[0;32m"
    YELLOW = "\033[1;33m"
    BLUE = "\033[0;34m"
    NC = "\033[0m"  # No Color


def print_status(status: str, message: str) -> None:
    """Print colored status messages (only in verbose mode)"""
    if not VERBOSE:
        return

    color_map = {
        "SUCCESS": Colors.GREEN + "✅ ",
        "ERROR": Colors.RED + "❌ ",
        "WARNING": Colors.YELLOW + "⚠️  ",
        "INFO": Colors.BLUE + "ℹ️  ",
    }

    color = color_map.get(status, "")
    print(f"{color}{message}{Colors.NC}")


def print_verbose(message: str) -> None:
    """Print a message only in verbose mode"""
    if VERBOSE:
        print(message)


def validate_json_syntax(file_path: Path) -> tuple[bool, str]:
    """Validate JSON syntax of a file"""
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            json.load(f)
        return True, "Valid JSON syntax"
    except json.JSONDecodeError as e:
        return False, f"Invalid JSON syntax: {e}"
    except Exception as e:
        return False, f"Error reading file: {e}"


def validate_tense_consistency(file_path: Path) -> tuple[bool, str, List[str]]:
    """Validate that a single verb file has proper tense structure"""
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            data = json.load(f)

        # Check basic structure for single verb file
        if "tenses" not in data:
            return (
                False,
                f"Verb ('{data.get('infinitive', 'unknown')}') has no 'tenses' key",
                [],
            )

        tenses = data["tenses"]
        if not tenses:
            return (
                False,
                f"Verb ('{data.get('infinitive', 'unknown')}') has no tenses",
                [],
            )

        # Check each tense has required fields
        errors = []
        tense_ids = set()
        for i, tense in enumerate(tenses):
            if "tenseId" not in tense:
                errors.append(f"Tense {i + 1} missing 'tenseId'")
                continue
            tense_ids.add(tense["tenseId"])

        if errors:
            return False, "Invalid tense structure", errors

        tense_list = sorted(tense_ids)
        return (
            True,
            f"Verb has proper tense structure ({len(tense_list)} tenses)",
            tense_list,
        )

    except Exception as e:
        return False, f"Error validating file: {e}", []


def validate_required_fields(file_path: Path) -> tuple[bool, str, List[str]]:
    """Validate that all required fields are present and non-empty"""
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            data = json.load(f)

        errors = []
        verb_name = data.get("infinitive", "unknown")

        # Check verb-level required fields
        required_verb_fields = [
            "language",
            "languageName",
            "infinitive",
            "infinitiveEn",
            "category",
            "tenses",
        ]
        for field in required_verb_fields:
            if field not in data:
                errors.append(f"Verb '{verb_name}' missing required field '{field}'")
            elif not data[field] or (
                isinstance(data[field], str) and data[field].strip() == ""
            ):
                errors.append(f"Verb '{verb_name}' has empty '{field}' field")

        # Check tenses
        if "tenses" in data and isinstance(data["tenses"], list):
            for j, tense in enumerate(data["tenses"]):
                tense_name = tense.get("tenseId", f"tense_{j + 1}")

                # Check tense-level required fields
                required_tense_fields = [
                    "tenseId",
                    "tenseName",
                    "tenseNameEn",
                    "description",
                    "conjugations",
                ]
                for field in required_tense_fields:
                    if field not in tense:
                        errors.append(
                            f"Verb '{verb_name}' tense '{tense_name}' missing required field '{field}'"
                        )
                    elif not tense[field] or (
                        isinstance(tense[field], str) and tense[field].strip() == ""
                    ):
                        errors.append(
                            f"Verb '{verb_name}' tense '{tense_name}' has empty '{field}' field"
                        )

                # Check conjugations
                if "conjugations" in tense and isinstance(tense["conjugations"], list):
                    for k, conjugation in enumerate(tense["conjugations"]):
                        conj_name = f"conjugation_{k + 1}"

                        # Check conjugation-level required fields
                        required_conj_fields = [
                            "pronoun",
                            "form",
                            "exampleSentence",
                            "exampleSentenceEn",
                        ]
                        for field in required_conj_fields:
                            if field not in conjugation:
                                errors.append(
                                    f"Verb '{verb_name}' tense '{tense_name}' {conj_name} missing required field '{field}'"
                                )
                            elif not conjugation[field] or (
                                isinstance(conjugation[field], str)
                                and conjugation[field].strip() == ""
                            ):
                                # Allow "—" for unused forms
                                if conjugation[field] != "—":
                                    errors.append(
                                        f"Verb '{verb_name}' tense '{tense_name}' {conj_name} has empty '{field}' field"
                                    )

        if errors:
            return False, "Missing or empty required fields", errors

        return True, "All required fields are present and non-empty", []

    except Exception as e:
        return False, f"Error validating required fields: {e}", []


def validate_filename_consistency(file_path: Path) -> tuple[bool, str, List[str]]:
    """Validate that the verb infinitive in the file matches the filename"""
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            data = json.load(f)

        if "infinitive" not in data:
            return False, "No 'infinitive' field found in file", []

        verb_infinitive = data["infinitive"]
        expected_filename = f"{verb_infinitive}.json"
        actual_filename = file_path.name

        if actual_filename != expected_filename:
            return (
                False,
                f"Filename mismatch: expected '{expected_filename}', got '{actual_filename}'",
                [],
            )

        return True, f"Filename matches verb infinitive: {verb_infinitive}", []

    except Exception as e:
        return False, f"Error validating filename consistency: {e}", []


def validate_example_sentences(file_path: Path) -> tuple[bool, str, List[str]]:
    """Validate that example sentences are properly formatted"""
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            data = json.load(f)

        errors = []
        verb_name = data.get("infinitive", "unknown")

        # Check verb's conjugations
        if "tenses" in data and isinstance(data["tenses"], list):
            for j, tense in enumerate(data["tenses"]):
                tense_name = tense.get("tenseId", f"tense_{j + 1}")

                if "conjugations" in tense and isinstance(tense["conjugations"], list):
                    for k, conjugation in enumerate(tense["conjugations"]):
                        conj_name = f"conjugation_{k + 1}"

                        # Check example sentences
                        example_sentence = conjugation.get("exampleSentence", "")
                        example_sentence_en = conjugation.get("exampleSentenceEn", "")
                        form = conjugation.get("form", "")

                        # If form is "—", then example sentences should also be "—"
                        if form == "—":
                            if example_sentence != "—" or example_sentence_en != "—":
                                errors.append(
                                    f"Verb '{verb_name}' tense '{tense_name}' {conj_name}: when form is '—', example sentences should also be '—'"
                                )
                        else:
                            # If form is not "—", example sentences should not be "—" and should be different
                            if example_sentence == "—" or example_sentence_en == "—":
                                errors.append(
                                    f"Verb '{verb_name}' tense '{tense_name}' {conj_name}: example sentences should not be '—' when form is not '—'"
                                )
                            elif example_sentence == example_sentence_en:
                                errors.append(
                                    f"Verb '{verb_name}' tense '{tense_name}' {conj_name}: example sentences should be different (got same text for both languages)"
                                )

        if errors:
            return False, "Example sentence validation failed", errors

        return True, "All example sentences are properly formatted", []

    except Exception as e:
        return False, f"Error validating example sentences: {e}", []


def validate_language_consistency(language_dir: Path) -> tuple[bool, str, List[str]]:
    """Validate that all verbs in a language directory have consistent language metadata and tenses"""
    try:
        json_files = list(language_dir.glob("*.json"))
        if not json_files:
            return False, "No verb files found in language directory", []

        errors = []
        reference_language = None
        reference_language_name = None
        reference_tense_ids = None

        for file_path in json_files:
            with open(file_path, "r", encoding="utf-8") as f:
                data = json.load(f)

            # Check language consistency
            if reference_language is None:
                reference_language = data.get("language")
                reference_language_name = data.get("languageName")
            else:
                if data.get("language") != reference_language:
                    errors.append(
                        f"File {file_path.name} has different language code: {data.get('language')} (expected {reference_language})"
                    )
                if data.get("languageName") != reference_language_name:
                    errors.append(
                        f"File {file_path.name} has different language name: {data.get('languageName')} (expected {reference_language_name})"
                    )

            # Check tense consistency
            if "tenses" in data:
                verb_tense_ids = set()
                for tense in data["tenses"]:
                    if "tenseId" in tense:
                        verb_tense_ids.add(tense["tenseId"])

                if reference_tense_ids is None:
                    reference_tense_ids = verb_tense_ids
                else:
                    if verb_tense_ids != reference_tense_ids:
                        missing = reference_tense_ids - verb_tense_ids
                        extra = verb_tense_ids - reference_tense_ids
                        error_msg = f"File {file_path.name} has inconsistent tenses"
                        if missing:
                            error_msg += f" (missing: {sorted(missing)})"
                        if extra:
                            error_msg += f" (extra: {sorted(extra)})"
                        errors.append(error_msg)

        if errors:
            return False, "Language consistency validation failed", errors

        return (
            True,
            f"All verbs have consistent language metadata and tenses ({len(reference_tense_ids)} tenses)",
            [],
        )

    except Exception as e:
        return False, f"Error validating language consistency: {e}", []


def main():
    """Main validation function"""
    global VERBOSE

    parser = argparse.ArgumentParser(description="Validate verb conjugation JSON files")
    parser.add_argument(
        "-v", "--verbose", action="store_true", help="Enable verbose output"
    )
    args = parser.parse_args()

    VERBOSE = args.verbose

    script_dir = Path(__file__).parent
    project_root = script_dir.parent
    verb_conjugations_dir = (
        project_root
        / "backend"
        / "internal"
        / "handlers"
        / "data"
        / "verb-conjugations"
    )

    print_verbose("🔍 Validating verb conjugation files...")
    print_verbose(f"Directory: {verb_conjugations_dir}")
    print_verbose("")

    # Counters
    total_files = 0
    valid_files = 0
    invalid_json_files = 0
    inconsistent_files = 0
    required_fields_errors = 0
    example_sentence_errors = 0
    language_consistency_errors = 0
    filename_consistency_errors = 0

    # Check if directory exists
    if not verb_conjugations_dir.exists():
        print_status(
            "ERROR", f"Verb conjugations directory not found: {verb_conjugations_dir}"
        )
        sys.exit(1)

    # Find all language directories
    language_dirs = [d for d in verb_conjugations_dir.iterdir() if d.is_dir()]
    language_dirs.sort(key=lambda x: x.name)

    if not language_dirs:
        print_status(
            "ERROR", f"No language directories found in {verb_conjugations_dir}"
        )
        sys.exit(1)

    print_verbose("Found language directories:")
    for lang_dir in language_dirs:
        print_verbose(f"  - {lang_dir.name}")
    print_verbose("")

    # Validate each language directory
    for lang_dir in language_dirs:
        print_verbose(f"🔍 Validating language: {lang_dir.name}")

        # Find all verb files in this language directory
        json_files = list(lang_dir.glob("*.json"))
        json_files.sort(key=lambda x: x.name)

        if not json_files:
            print_status("WARNING", f"No verb files found in {lang_dir.name}")
            continue

        print_verbose(f"  Found {len(json_files)} verb files")

        # Validate language consistency
        lang_consistent, lang_message, lang_errors = validate_language_consistency(
            lang_dir
        )
        if lang_consistent:
            print_status("SUCCESS", f"Language consistency: {lang_message}")
        else:
            print_status("ERROR", f"Language consistency: {lang_message}")
            for error in lang_errors:
                print_verbose(f"    - {error}")
            language_consistency_errors += 1

        # Validate each verb file
        for file_path in json_files:
            total_files += 1
            print_verbose(f"  🔍 Validating {file_path.name}...")

            # Check JSON syntax
            json_valid, json_message = validate_json_syntax(file_path)
            if json_valid:
                print_status("SUCCESS", f"  {json_message}")
            else:
                print_status("ERROR", f"  {json_message}")
                invalid_json_files += 1
                continue

            # Check tense consistency
            consistent, consistency_message, tense_list = validate_tense_consistency(
                file_path
            )
            if consistent:
                print_status("SUCCESS", f"  {consistency_message}")
            else:
                print_status("ERROR", f"  {consistency_message}")
                for error in tense_list:
                    print_verbose(f"      - {error}")
                inconsistent_files += 1

            # Check required fields
            fields_valid, fields_message, fields_errors = validate_required_fields(
                file_path
            )
            if fields_valid:
                print_status("SUCCESS", f"  {fields_message}")
            else:
                print_status("ERROR", f"  {fields_message}")
                for error in fields_errors:
                    print_verbose(f"      - {error}")
                required_fields_errors += 1

            # Check example sentences
            examples_valid, examples_message, examples_errors = (
                validate_example_sentences(file_path)
            )
            if examples_valid:
                print_status("SUCCESS", f"  {examples_message}")
            else:
                print_status("ERROR", f"  {examples_message}")
                for error in examples_errors:
                    print_verbose(f"      - {error}")
                example_sentence_errors += 1

            # Check filename consistency
            filename_valid, filename_message, filename_errors = (
                validate_filename_consistency(file_path)
            )
            if filename_valid:
                print_status("SUCCESS", f"  {filename_message}")
            else:
                print_status("ERROR", f"  {filename_message}")
                for error in filename_errors:
                    print_verbose(f"      - {error}")
                filename_consistency_errors += 1

            # Overall file validity
            if consistent and fields_valid and examples_valid and filename_valid:
                valid_files += 1

        print_verbose("")

    # Summary - always print
    print("📊 Validation Summary:")
    print(f"  Total files: {total_files}")
    print(f"  Valid files: {valid_files}")
    print(f"  Invalid JSON: {invalid_json_files}")
    print(f"  Inconsistent tenses: {inconsistent_files}")
    print(f"  Required fields errors: {required_fields_errors}")
    print(f"  Example sentence errors: {example_sentence_errors}")
    print(f"  Language consistency errors: {language_consistency_errors}")
    print(f"  Filename consistency errors: {filename_consistency_errors}")

    # Check if validation passed
    validation_passed = (
        invalid_json_files == 0
        and inconsistent_files == 0
        and required_fields_errors == 0
        and example_sentence_errors == 0
        and language_consistency_errors == 0
        and filename_consistency_errors == 0
    )

    if validation_passed:
        print_status(
            "SUCCESS", "All verb conjugation files are valid and consistent! 🎉"
        )
        sys.exit(0)
    else:
        print_status("ERROR", "Validation failed. Please fix the issues above.")
        sys.exit(1)


if __name__ == "__main__":
    main()

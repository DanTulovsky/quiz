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
            valid_files += 1
        else:
            print_status("ERROR", consistency_message)
            for error in tense_list:
                print(f"    - {error}")
            inconsistent_files += 1

        print()

    # Summary
    print("📊 Validation Summary:")
    print(f"  Total files: {total_files}")
    print(f"  Valid files: {valid_files}")
    print(f"  Invalid JSON: {invalid_json_files}")
    print(f"  Inconsistent tenses: {inconsistent_files}")

    if invalid_json_files == 0 and inconsistent_files == 0:
        print_status("SUCCESS", "All verb conjugation files are valid and consistent! 🎉")
        sys.exit(0)
    else:
        print_status("ERROR", "Validation failed. Please fix the issues above.")
        sys.exit(1)


if __name__ == "__main__":
    main()

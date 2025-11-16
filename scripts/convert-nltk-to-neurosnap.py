#!/usr/bin/env python3
"""
Convert NLTK Punkt pickle model to neurosnap/sentences JSON format.

This script downloads the Russian Punkt model from ru_punkt and converts it
to the JSON format expected by neurosnap/sentences.
"""
import pickle
import json
import sys
import urllib.request
from pathlib import Path

def download_russian_pickle(output_path):
    """Download the Russian Punkt pickle file from ru_punkt repository."""
    url = "https://raw.githubusercontent.com/Mottl/ru_punkt/master/nltk_data/tokenizers/punkt/russian.pickle"
    print(f"Downloading Russian Punkt model from {url}...")
    urllib.request.urlretrieve(url, output_path)
    print(f"✅ Downloaded to {output_path}")

def convert_pickle_to_json(pickle_path, json_path):
    """Convert NLTK pickle format to neurosnap/sentences JSON format."""
    print(f"Loading pickle file from {pickle_path}...")

    # Load the NLTK pickle file
    with open(pickle_path, 'rb') as f:
        # Python 3 pickle format
        tokenizer = pickle.load(f)

    # Extract the parameters from the PunktSentenceTokenizer
    # The tokenizer has a _params attribute with PunktParameters
    params = tokenizer._params

    # Build the JSON structure expected by neurosnap/sentences
    result = {
        "SentStarters": {},
        "OrthoContext": {},
        "AbbrevTypes": [],
        "Collocations": {}
    }

    # Convert SentStarters (set -> dict with value 1)
    if hasattr(params, 'sent_starters') and params.sent_starters:
        result["SentStarters"] = {word: 1 for word in params.sent_starters}
    elif hasattr(params, 'SentStarters'):
        # Might already be a dict
        if isinstance(params.SentStarters, dict):
            result["SentStarters"] = params.SentStarters
        elif isinstance(params.SentStarters, (set, list)):
            result["SentStarters"] = {word: 1 for word in params.SentStarters}

    # Convert OrthoContext (dict mapping word -> context value)
    if hasattr(params, 'ortho_context') and params.ortho_context:
        result["OrthoContext"] = dict(params.ortho_context)
    elif hasattr(params, 'OrthoContext'):
        if isinstance(params.OrthoContext, dict):
            result["OrthoContext"] = params.OrthoContext

    # Convert AbbrevTypes (set -> map with value 1)
    # neurosnap/sentences expects a map with numeric values, not an array
    if hasattr(params, 'abbrev_types') and params.abbrev_types:
        result["AbbrevTypes"] = {abbrev: 1 for abbrev in params.abbrev_types}
    elif hasattr(params, 'AbbrevTypes'):
        if isinstance(params.AbbrevTypes, (set, list)):
            result["AbbrevTypes"] = {abbrev: 1 for abbrev in params.AbbrevTypes}
        elif isinstance(params.AbbrevTypes, dict):
            # If it's already a dict, keep it as is
            result["AbbrevTypes"] = params.AbbrevTypes

    # Convert Collocations (dict mapping collocation -> count)
    if hasattr(params, 'collocations') and params.collocations:
        result["Collocations"] = dict(params.collocations)
    elif hasattr(params, 'Collocations'):
        if isinstance(params.Collocations, dict):
            result["Collocations"] = params.Collocations
        elif isinstance(params.Collocations, (set, list)):
            # Convert set/list to dict with value 1
            result["Collocations"] = {col: 1 for col in params.Collocations}

    # Write JSON file
    print(f"Writing JSON file to {json_path}...")
    with open(json_path, 'w', encoding='utf-8') as f:
        json.dump(result, f, ensure_ascii=False, indent=2)

    print(f"✅ Converted successfully!")
    print(f"   SentStarters: {len(result['SentStarters'])} items")
    print(f"   OrthoContext: {len(result['OrthoContext'])} items")
    print(f"   AbbrevTypes: {len(result['AbbrevTypes'])} items")
    print(f"   Collocations: {len(result['Collocations'])} items")

def main():
    """Main entry point."""
    script_dir = Path(__file__).parent
    root_dir = script_dir.parent
    dest_dir = root_dir / "backend" / "internal" / "resources" / "punkt"

    # Create destination directory
    dest_dir.mkdir(parents=True, exist_ok=True)

    # Temporary pickle file
    pickle_path = dest_dir / "russian.pickle"
    json_path = dest_dir / "russian.json"

    # Download if not exists
    if not pickle_path.exists():
        download_russian_pickle(pickle_path)

    # Convert
    convert_pickle_to_json(pickle_path, json_path)

    print(f"\n✅ Russian Punkt model available at: {json_path}")

if __name__ == "__main__":
    main()


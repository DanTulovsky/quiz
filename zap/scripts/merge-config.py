#!/usr/bin/env python3
"""
ZAP Configuration Merger
Handles inheritance between configuration files by merging base.yaml with specific configs.
"""

import yaml
import sys
import os
from pathlib import Path
from typing import Dict, Any


def deep_merge(base: Dict[str, Any], override: Dict[str, Any]) -> Dict[str, Any]:
    """Deep merge two dictionaries, with override taking precedence."""
    result = base.copy()

    for key, value in override.items():
        if key in result and isinstance(result[key], dict) and isinstance(value, dict):
            result[key] = deep_merge(result[key], value)
        else:
            result[key] = value

    return result


def load_yaml_file(file_path: str) -> Dict[str, Any]:
    """Load a YAML file and return its contents as a dictionary."""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            return yaml.safe_load(f) or {}
    except FileNotFoundError:
        print(f"Warning: Configuration file {file_path} not found")
        return {}
    except yaml.YAMLError as e:
        print(f"Error parsing {file_path}: {e}")
        return {}


def merge_config(config_file: str, base_dir: str = ".") -> Dict[str, Any]:
    """Merge a configuration file with its base configuration."""
    config_path = Path(base_dir) / config_file

    if not config_path.exists():
        print(f"Error: Configuration file {config_file} not found")
        return {}

    # Load the main configuration
    config = load_yaml_file(str(config_path))

    # Check if it extends a base configuration
    extends_file = config.get('extends')
    if extends_file:
        # Load the base configuration
        base_path = Path(base_dir) / extends_file
        base_config = load_yaml_file(str(base_path))

        # Merge configurations
        merged_config = deep_merge(base_config, config)

        # Remove the extends key from the final config
        merged_config.pop('extends', None)

        return merged_config

    return config


def main():
    """Main function to merge and output a configuration."""
    if len(sys.argv) < 2:
        print("Usage: python merge-config.py <config-file> [base-dir]")
        print("Example: python merge-config.py baseline.yaml")
        sys.exit(1)

    config_file = sys.argv[1]
    base_dir = sys.argv[2] if len(sys.argv) > 2 else "."

    # Merge the configuration
    merged_config = merge_config(config_file, base_dir)

    if merged_config:
        # Output the merged configuration as YAML
        print(yaml.dump(merged_config, default_flow_style=False, sort_keys=False))
    else:
        print("Error: Failed to merge configuration")
        sys.exit(1)


if __name__ == "__main__":
    main()

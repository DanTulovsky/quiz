#!/bin/bash
# Build iOS app with configurable destination
# Usage: ios-build.sh [-p] [DESTINATION=...]
#   -p: Interactive picker to select destination

set -e

cd "$(dirname "$0")/../ios" || exit 1

# Check for -p flag
PICK_MODE=false
while [[ $# -gt 0 ]]; do
  case $1 in
    -p|--pick)
      PICK_MODE=true
      shift
      ;;
    *)
      shift
      ;;
  esac
done

if [ "$PICK_MODE" = true ]; then
  echo "📋 Fetching available build destinations..."

  # Parse xcodebuild output - each destination is on a single line
  # Format: { platform:macOS, arch:arm64, variant:Designed for [iPad,iPhone], id:00006040-000E61323C20801C, name:My Mac }
  DEST_LIST=$(xcodebuild -scheme Quiz -showdestinations 2>/dev/null | \
    grep -E "^\s+\{" | \
    awk '{
      line = $0
      platform = ""; id = ""; name = ""; os = ""; variant = ""

      # Extract platform
      if (match(line, /platform:([^,}]+)/)) {
        platform = substr(line, RSTART+9, RLENGTH-9)
        gsub(/^[ \t]+|[ \t]+$/, "", platform)
      }
      # Extract id
      if (match(line, /id:([^,}]+)/)) {
        id = substr(line, RSTART+3, RLENGTH-3)
        gsub(/^[ \t]+|[ \t]+$/, "", id)
      }
      # Extract name
      if (match(line, /name:([^,}]+)/)) {
        name = substr(line, RSTART+5, RLENGTH-5)
        gsub(/^[ \t]+|[ \t]+$/, "", name)
      }
      # Extract OS
      if (match(line, /OS:([^,}]+)/)) {
        os = substr(line, RSTART+3, RLENGTH-3)
        gsub(/^[ \t]+|[ \t]+$/, "", os)
      }
      # Extract variant (may contain brackets and commas, so extract until next field)
      if (match(line, /variant:([^}]+)/)) {
        variant = substr(line, RSTART+8, RLENGTH-8)
        # Remove trailing comma and any following fields
        sub(/,[^}]*$/, "", variant)
        gsub(/^[ \t]+|[ \t]+$/, "", variant)
      }

      # Format display name
      display_name = name
      if (os != "") display_name = display_name " (OS " os ")"
      if (variant != "") display_name = display_name " [" variant "]"

      # Format destination string
      if (platform == "macOS") {
        # For macOS, use id if available (more reliable than variant with special chars)
        if (id != "") {
          dest_str = "platform=macOS,id=" id
        } else if (variant != "") {
          # Quote variant if it contains special characters
          if (variant ~ /[\[\],]/) {
            dest_str = "platform=macOS,variant=\"" variant "\""
          } else {
            dest_str = "platform=macOS,variant=" variant
          }
        } else {
          dest_str = "platform=macOS"
        }
      } else if (id != "" && id !~ /placeholder/) {
        dest_str = "platform=" platform ",id=" id
      } else {
        dest_str = "platform=" platform ",name=" name
        if (os != "") dest_str = dest_str ",OS=" os
      }

      print dest_str "|" display_name
    }' | nl -w2 -s'. ')

  if [ -z "$DEST_LIST" ]; then
    echo "❌ No destinations found"
    exit 1
  fi

  echo ""
  echo "Available destinations:"
  echo "$DEST_LIST" | awk -F'|' '{printf "  %s - %s\n", $1, $2}'
  echo ""
  read -p "Select destination number: " SELECTION

  SELECTED_LINE=$(echo "$DEST_LIST" | sed -n "${SELECTION}p")
  if [ -z "$SELECTED_LINE" ]; then
    echo "❌ Invalid selection"
    exit 1
  fi

  # Extract destination string (after number prefix, before |) and display name (after |)
  # Format: "  1. platform=macOS,variant=Mac Catalyst|My Mac [Designed for [iPad]"
  DEST=$(echo "$SELECTED_LINE" | awk -F'|' '{print $1}' | sed 's/^[[:space:]]*[0-9]*\. //')
  DISPLAY_NAME=$(echo "$SELECTED_LINE" | awk -F'|' '{print $2}')

  if [ -z "$DEST" ]; then
    echo "❌ Invalid selection"
    exit 1
  fi

  echo "✅ Selected: $DISPLAY_NAME"
  echo ""
elif [ -n "$DESTINATION" ]; then
  DEST="$DESTINATION"
else
  DEST="${DESTINATION:-platform=iOS Simulator,name=iPhone 17,OS=26.2}"
fi

echo "📱 Building iOS app with destination: $DEST"
xcodebuild -scheme Quiz -destination "$DEST" clean build
echo "✅ iOS build completed!"


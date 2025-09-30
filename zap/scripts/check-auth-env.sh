#!/bin/bash
if [ -z "$ZAP_ADMIN_USERNAME" ] || [ -z "$ZAP_ADMIN_PASSWORD" ]; then
  echo "❌ ERROR: ZAP_ADMIN_USERNAME and ZAP_ADMIN_PASSWORD environment variables are required for authenticated scans"
  echo "   Set them before running:"
  echo "   export ZAP_ADMIN_USERNAME=admin@example.com"
  echo "   export ZAP_ADMIN_PASSWORD=admin123"
  exit 1
fi

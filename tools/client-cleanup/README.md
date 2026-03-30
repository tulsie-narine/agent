# Client Cleanup Utility

A CLI utility for removing all ScalePad client data for one client after an explicit confirmation.

## What it removes

The utility removes, for a selected client:

- initiatives
- meetings
- action items
- notes
- assessments
- goals
- contracts

## Requirements

- Node.js 18+ (uses native `fetch`)
- ScalePad API base URL and token

## Environment variables

```bash
export SCALEPAD_BASE_URL="https://api.scalepad.example.com"
export SCALEPAD_API_TOKEN="<your-token>"
```

Optional:

```bash
export CLIENT_CLEANUP_CONFIG="./tools/client-cleanup/config.example.json"
```

## Usage

```bash
node tools/client-cleanup/cleanup-client.js
```

The app prompts for:
1. Client ID
2. A confirmation phrase (`DELETE <clientId>`) before deletion starts

## Endpoint customization

If your ScalePad endpoints differ from defaults, copy `config.example.json`, update paths/fields, and point `CLIENT_CLEANUP_CONFIG` to your file.

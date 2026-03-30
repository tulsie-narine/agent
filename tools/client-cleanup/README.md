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
- ScalePad API key

## Environment variables

```bash
export SCALEPAD_BASE_URL="https://api.scalepad.com"
export SCALEPAD_API_KEY="<your-api-key>"
```

`SCALEPAD_API_TOKEN` is also accepted for compatibility with earlier versions of the script.

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

The default config targets the Lifecycle Manager API and filters records with `filter[client.id]`. If your tenant or workflow needs different paths/fields, copy `config.example.json`, update paths/fields, and point `CLIENT_CLEANUP_CONFIG` to your file.

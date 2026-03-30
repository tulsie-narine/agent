#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const readline = require('readline/promises');
const { stdin: input, stdout: output } = require('process');

const DEFAULT_CONFIG = {
  resources: [
    { name: 'initiatives', listPath: '/lifecycle-manager/v1/initiatives', deletePath: '/lifecycle-manager/v1/initiatives/{id}', queryParam: 'filter[client.id]', idField: 'id' },
    { name: 'meetings', listPath: '/lifecycle-manager/v1/meetings', deletePath: '/lifecycle-manager/v1/meetings/{id}', queryParam: 'filter[client.id]', idField: 'id' },
    { name: 'action items', listPath: '/lifecycle-manager/v1/action-items', deletePath: '/lifecycle-manager/v1/action-items/{id}', queryParam: 'filter[client.id]', idField: 'engagement_action_id' },
    { name: 'notes', listPath: '/lifecycle-manager/v1/notes', deletePath: '/lifecycle-manager/v1/notes/{id}', queryParam: 'filter[client.id]', idField: 'note_id' },
    { name: 'assessments', listPath: '/lifecycle-manager/v1/assessments', deletePath: '/lifecycle-manager/v1/assessments/{id}', queryParam: 'filter[client.id]', idField: 'id' },
    { name: 'goals', listPath: '/lifecycle-manager/v1/goals', deletePath: '/lifecycle-manager/v1/goals/{id}', queryParam: 'filter[client.id]', idField: 'id' },
    { name: 'contracts', listPath: '/lifecycle-manager/v1/contracts', deletePath: '/lifecycle-manager/v1/contracts/{id}', queryParam: 'filter[client.id]', idField: 'id' }
  ]
};

function normalizeBaseUrl(url) {
  return url.endsWith('/') ? url.slice(0, -1) : url;
}

function loadConfig() {
  const configPath = process.env.CLIENT_CLEANUP_CONFIG;
  if (!configPath) {
    return DEFAULT_CONFIG;
  }

  const fullPath = path.resolve(configPath);
  const raw = fs.readFileSync(fullPath, 'utf8');
  const parsed = JSON.parse(raw);

  if (!Array.isArray(parsed.resources) || parsed.resources.length === 0) {
    throw new Error('Invalid config: "resources" must be a non-empty array');
  }

  return parsed;
}

async function apiRequest(baseUrl, apiKey, method, requestPath, query) {
  const url = new URL(`${baseUrl}${requestPath}`);
  if (query && typeof query === 'object') {
    for (const [key, value] of Object.entries(query)) {
      url.searchParams.set(key, String(value));
    }
  }

  const response = await fetch(url, {
    method,
    headers: {
      Accept: 'application/json',
      'x-api-key': apiKey
    }
  });

  if (response.status === 204) {
    return null;
  }

  const contentType = response.headers.get('content-type') || '';
  const responseBody = contentType.includes('application/json')
    ? await response.json()
    : await response.text();

  if (!response.ok) {
    const message = typeof responseBody === 'string' ? responseBody : JSON.stringify(responseBody);
    throw new Error(`HTTP ${response.status} for ${method} ${url.pathname}: ${message}`);
  }

  return responseBody;
}

function extractItems(payload) {
  if (!payload) {
    return [];
  }

  if (Array.isArray(payload)) {
    return payload;
  }

  if (Array.isArray(payload.items)) {
    return payload.items;
  }

  if (Array.isArray(payload.results)) {
    return payload.results;
  }

  if (Array.isArray(payload.data)) {
    return payload.data;
  }

  return [];
}

async function listAllResourceItems(baseUrl, apiKey, clientId, resource) {
  const items = [];
  let cursor;

  do {
    const query = {
      [resource.queryParam || 'filter[client.id]']: clientId,
      page_size: resource.pageSize || 200
    };

    if (cursor) {
      query.cursor = cursor;
    }

    const listData = await apiRequest(baseUrl, apiKey, 'GET', resource.listPath, query);
    items.push(...extractItems(listData));
    cursor = listData && typeof listData === 'object' ? listData.next_cursor : undefined;
  } while (cursor);

  return items;
}

async function cleanupResource(baseUrl, apiKey, clientId, resource) {
  const items = await listAllResourceItems(baseUrl, apiKey, clientId, resource);
  const idField = resource.idField || 'id';
  const ids = items
    .map((item) => (item ? item[idField] : undefined))
    .filter((id) => id !== undefined && id !== null);

  let deleted = 0;
  for (const id of ids) {
    const deletePath = (resource.deletePath || `${resource.listPath}/{id}`).replace('{id}', String(id));
    await apiRequest(baseUrl, apiKey, 'DELETE', deletePath);
    deleted += 1;
  }

  return { found: items.length, deleted };
}

async function main() {
  const baseUrl = process.env.SCALEPAD_BASE_URL || 'https://api.scalepad.com';
  const apiKey = process.env.SCALEPAD_API_TOKEN || process.env.SCALEPAD_API_KEY;

  if (!apiKey) {
    throw new Error('Missing SCALEPAD_API_TOKEN or SCALEPAD_API_KEY environment variable');
  }

  const config = loadConfig();
  const rl = readline.createInterface({ input, output });

  try {
    console.log('=== ScalePad Client Cleanup ===');
    const clientId = (await rl.question('Enter the client ID to clean up: ')).trim();

    if (!clientId) {
      throw new Error('A client ID is required');
    }

    console.log('\nThis will remove all records for this client in:');
    for (const resource of config.resources) {
      console.log(` - ${resource.name}`);
    }

    const confirmationText = `DELETE ${clientId}`;
    const confirmation = (await rl.question(`\nType "${confirmationText}" to confirm: `)).trim();

    if (confirmation !== confirmationText) {
      console.log('Cleanup cancelled.');
      return;
    }

    const normalizedBaseUrl = normalizeBaseUrl(baseUrl);
    const summary = [];

    for (const resource of config.resources) {
      process.stdout.write(`Cleaning ${resource.name}... `);
      const result = await cleanupResource(normalizedBaseUrl, apiKey, clientId, resource);
      summary.push({ resource: resource.name, ...result });
      console.log(`done (found: ${result.found}, deleted: ${result.deleted})`);
    }

    console.log('\nCleanup completed.');
    for (const row of summary) {
      console.log(` - ${row.resource}: found ${row.found}, deleted ${row.deleted}`);
    }
  } finally {
    rl.close();
  }
}

main().catch((error) => {
  console.error(`\nError: ${error.message}`);
  process.exit(1);
});

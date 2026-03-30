#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const readline = require('readline/promises');
const { stdin: input, stdout: output } = require('process');

const DEFAULT_CONFIG = {
  resources: [
    { name: 'initiatives', listPath: '/v1/initiatives', deletePath: '/v1/initiatives/{id}', queryParam: 'clientId', idField: 'id' },
    { name: 'meetings', listPath: '/v1/meetings', deletePath: '/v1/meetings/{id}', queryParam: 'clientId', idField: 'id' },
    { name: 'action items', listPath: '/v1/action-items', deletePath: '/v1/action-items/{id}', queryParam: 'clientId', idField: 'id' },
    { name: 'notes', listPath: '/v1/notes', deletePath: '/v1/notes/{id}', queryParam: 'clientId', idField: 'id' },
    { name: 'assessments', listPath: '/v1/assessments', deletePath: '/v1/assessments/{id}', queryParam: 'clientId', idField: 'id' },
    { name: 'goals', listPath: '/v1/goals', deletePath: '/v1/goals/{id}', queryParam: 'clientId', idField: 'id' },
    { name: 'contracts', listPath: '/v1/contracts', deletePath: '/v1/contracts/{id}', queryParam: 'clientId', idField: 'id' }
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

async function apiRequest(baseUrl, token, method, requestPath, query) {
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
      Authorization: `Bearer ${token}`
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

async function cleanupResource(baseUrl, token, clientId, resource) {
  const listData = await apiRequest(baseUrl, token, 'GET', resource.listPath, {
    [resource.queryParam || 'clientId']: clientId
  });

  const items = extractItems(listData);
  const idField = resource.idField || 'id';
  const ids = items
    .map((item) => (item ? item[idField] : undefined))
    .filter((id) => id !== undefined && id !== null);

  let deleted = 0;
  for (const id of ids) {
    const deletePath = (resource.deletePath || `${resource.listPath}/{id}`).replace('{id}', String(id));
    await apiRequest(baseUrl, token, 'DELETE', deletePath);
    deleted += 1;
  }

  return { found: items.length, deleted };
}

async function main() {
  const baseUrl = process.env.SCALEPAD_BASE_URL;
  const token = process.env.SCALEPAD_API_TOKEN;

  if (!baseUrl || !token) {
    throw new Error('Missing SCALEPAD_BASE_URL or SCALEPAD_API_TOKEN environment variables');
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
      const result = await cleanupResource(normalizedBaseUrl, token, clientId, resource);
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

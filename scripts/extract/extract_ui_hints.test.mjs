import test from 'node:test';
import assert from 'node:assert/strict';

import { mergeExistingMetadata } from './extract_ui_hints.mjs';

test('mergeExistingMetadata preserves known version for same commit when extraction falls back to unknown', () => {
  const existing = {
    version: '1.0.0',
    extracted_at: '2026-04-12T03:44:39.826Z',
    comfyui_commit_sha: 'abc123',
    comfyui_version: 'v0.18.5',
    total_nodes: 2,
    failed_nodes: [],
    nodes: {
      A: { min_width: 100 },
    },
  };
  const output = {
    version: '1.0.0',
    extracted_at: '2026-04-12T04:00:00.000Z',
    comfyui_commit_sha: 'abc123',
    comfyui_version: 'unknown',
    total_nodes: 2,
    failed_nodes: [],
    nodes: {
      A: { min_width: 100 },
    },
  };

  mergeExistingMetadata(existing, output);

  assert.equal(output.comfyui_version, 'v0.18.5');
  assert.equal(output.extracted_at, '2026-04-12T03:44:39.826Z');
});

test('mergeExistingMetadata does not preserve stale version across commit changes', () => {
  const existing = {
    version: '1.0.0',
    extracted_at: '2026-04-12T03:44:39.826Z',
    comfyui_commit_sha: 'abc123',
    comfyui_version: 'v0.18.5',
    total_nodes: 2,
    failed_nodes: [],
    nodes: {
      A: { min_width: 100 },
    },
  };
  const output = {
    version: '1.0.0',
    extracted_at: '2026-04-12T04:00:00.000Z',
    comfyui_commit_sha: 'def456',
    comfyui_version: 'unknown',
    total_nodes: 2,
    failed_nodes: [],
    nodes: {
      A: { min_width: 100 },
    },
  };

  mergeExistingMetadata(existing, output);

  assert.equal(output.comfyui_version, 'unknown');
  assert.equal(output.extracted_at, '2026-04-12T04:00:00.000Z');
});

test('mergeExistingMetadata preserves committed version when same commit derives a different version string', () => {
  const existing = {
    version: '1.0.0',
    extracted_at: '2026-04-12T03:44:39.826Z',
    comfyui_commit_sha: 'abc123',
    comfyui_version: 'v0.18.5',
    total_nodes: 2,
    failed_nodes: [],
    nodes: {
      A: { min_width: 100 },
    },
  };
  const output = {
    version: '1.0.0',
    extracted_at: '2026-04-12T04:00:00.000Z',
    comfyui_commit_sha: 'abc123',
    comfyui_version: '7782171',
    total_nodes: 2,
    failed_nodes: [],
    nodes: {
      A: { min_width: 100 },
    },
  };

  mergeExistingMetadata(existing, output);

  assert.equal(output.comfyui_version, 'v0.18.5');
  assert.equal(output.extracted_at, '2026-04-12T03:44:39.826Z');
});

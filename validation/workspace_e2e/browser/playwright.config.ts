import { defineConfig } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Read base URL from environment or runtime.env file
let baseURL = process.env.WORKSPACE_E2E_BASE_URL;

if (!baseURL) {
  const runtimeEnvPath = path.resolve(__dirname, '../.runtime/runtime.env');
  if (fs.existsSync(runtimeEnvPath)) {
    const runtimeEnv = fs.readFileSync(runtimeEnvPath, 'utf-8');
    const match = runtimeEnv.match(/^WORKSPACE_E2E_BASE_URL=(.+)$/m);
    if (match) {
      baseURL = match[1];
    }
  }
}

// Fallback to default only if no runtime file exists
baseURL = baseURL ?? 'http://127.0.0.1:8188';

export default defineConfig({
  testDir: './tests',
  timeout: 120000,
  fullyParallel: false,
  workers: 1,
  reporter: 'list',
  projects: [
    {
      name: 'chromium',
      use: {
        browserName: 'chromium',
      },
    },
  ],
  use: {
    baseURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
});

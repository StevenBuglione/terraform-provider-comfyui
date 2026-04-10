import { defineConfig } from '@playwright/test';

const baseURL = process.env.WORKSPACE_E2E_BASE_URL ?? 'http://127.0.0.1:8188';

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

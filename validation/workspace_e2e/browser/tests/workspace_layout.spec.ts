import fs from 'node:fs';
import path from 'node:path';

import { expect, test } from '@playwright/test';

import { dragFirstNode, getFirstNodePosition, listWorkspaceNames, loadWorkspaceByName } from './helpers/comfyui';
import { readLayoutMetrics } from './helpers/layout_metrics';

const expectedWorkspaces = ['dense_grid', 'mixed_overrides', 'vertical_stack', 'wide_gallery'];
const browserArtifactsDir = path.resolve(process.cwd(), '../artifacts/browser');

test('verifies staged workspaces in the real ComfyUI canvas', async ({ page }) => {
  fs.mkdirSync(browserArtifactsDir, { recursive: true });

  await page.goto('/');
  await page.waitForLoadState('networkidle');

  const discoveredNames = await listWorkspaceNames(page);
  for (const workspaceName of expectedWorkspaces) {
    expect(discoveredNames).toContain(workspaceName);
  }

  for (const workspaceName of expectedWorkspaces) {
    await test.step(`verify ${workspaceName}`, async () => {
      const loaded = await loadWorkspaceByName(page, workspaceName);
      expect(loaded.nodeCount).toBeGreaterThan(0);
      expect(loaded.groupCount).toBeGreaterThan(0);

      await page.waitForTimeout(200);

      const metrics = await readLayoutMetrics(page);
      expect(metrics.nodeCount).toBe(loaded.nodeCount);
      expect(metrics.groupCount).toBe(loaded.groupCount);
      expect(metrics.ungroupedNodes).toEqual([]);
      expect(metrics.groupOverlaps).toEqual([]);
      expect(metrics.nodeOverlaps).toEqual([]);
      expect(metrics.intraGroupNodeOverlaps).toEqual([]);
      expect(metrics.headerOverlaps).toEqual([]);
      expect(metrics.bodyContainmentViolations).toEqual([]);
      expect(metrics.backwardLinks).toEqual([]);
      expect(metrics.groups.every((group) => group.fullyVisible)).toBe(true);

      // Verify styled group in mixed_overrides workspace
      if (workspaceName === 'mixed_overrides') {
        const compactRefGroup = metrics.groups.find((g) => g.title === 'Compact Reference');
        expect(compactRefGroup).toBeDefined();
        expect(compactRefGroup?.color).toBe('#ff00ff');
        expect(compactRefGroup?.fontSize).toBe(28);
      }

      const initialPosition = await getFirstNodePosition(page);
      const movedPosition = await dragFirstNode(page, 48, 36);
      expect(movedPosition).not.toEqual(initialPosition);

      await page.screenshot({
        path: path.join(browserArtifactsDir, `${workspaceName}.png`),
        fullPage: true,
      });
      fs.writeFileSync(
        path.join(browserArtifactsDir, `${workspaceName}.metrics.json`),
        JSON.stringify(metrics, null, 2),
        'utf-8',
      );
    });
  }
});

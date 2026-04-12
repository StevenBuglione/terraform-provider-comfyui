import fs from 'node:fs';
import path from 'node:path';

import { expect, test, type Page } from '@playwright/test';

import { dragFirstNode, getFirstNodePosition, listWorkspaceNames, loadWorkspaceByName } from './helpers/comfyui';
import { readConnectionMetrics } from './helpers/connection_metrics';
import { readLayoutMetrics } from './helpers/layout_metrics';

const browserArtifactsDir = path.resolve(process.cwd(), '../artifacts/browser');
const runtimeOutputsPath = path.resolve(process.cwd(), '../.runtime/terraform-outputs.json');

type WorkspaceMetrics = {
  node_count: number;
  group_count: number;
  link_count: number;
  max_in_degree: number;
  max_out_degree: number;
};

async function dismissTransientOverlays(page: Page) {
  const closeDialog = page.getByRole('button', { name: 'Close dialog' });
  if (await closeDialog.count()) {
    await closeDialog.first().click();
  }

  const alertCloseButton = page.locator('[role="alert"] button').first();
  if (await alertCloseButton.count()) {
    await alertCloseButton.click();
  }
}

function readExpectedWorkspaceMetrics(): Record<string, WorkspaceMetrics> {
  const runtimeOutputs = JSON.parse(fs.readFileSync(runtimeOutputsPath, 'utf-8')) as {
    workspace_metrics: { value: Record<string, WorkspaceMetrics> };
  };
  return runtimeOutputs.workspace_metrics.value;
}

test('verifies staged workspaces in the real ComfyUI canvas', async ({ page }) => {
  fs.mkdirSync(browserArtifactsDir, { recursive: true });
  const expectedWorkspaces = readExpectedWorkspaceMetrics();
  const workspaceNames = Object.keys(expectedWorkspaces).sort();

  await page.goto('/');
  await page.waitForLoadState('networkidle');
  await dismissTransientOverlays(page);

  const discoveredNames = await listWorkspaceNames(page);
  for (const workspaceName of workspaceNames) {
    expect(discoveredNames).toContain(workspaceName);
  }

  for (const workspaceName of workspaceNames) {
    await test.step(`verify ${workspaceName}`, async () => {
      const expected = expectedWorkspaces[workspaceName];
      const loaded = await loadWorkspaceByName(page, workspaceName);
      expect(loaded.nodeCount).toBe(expected.node_count);
      expect(loaded.groupCount).toBe(expected.group_count);

      await page.waitForTimeout(200);

      const metrics = await readLayoutMetrics(page);
      const connectionMetrics = await readConnectionMetrics(page);
      expect(metrics.nodeCount).toBe(loaded.nodeCount);
      expect(metrics.groupCount).toBe(loaded.groupCount);
      expect(connectionMetrics.linkCount).toBe(expected.link_count);
      expect(connectionMetrics.maxInDegree).toBe(expected.max_in_degree);
      expect(connectionMetrics.maxOutDegree).toBe(expected.max_out_degree);
      expect(connectionMetrics.brokenLinks).toEqual([]);
      expect(metrics.ungroupedNodes).toEqual([]);
      if (workspaceName !== 'mixed_overrides') {
        expect(metrics.groupOverlaps).toEqual([]);
        expect(metrics.intraGroupNodeOverlaps).toEqual([]);
        expect(metrics.headerOverlaps).toEqual([]);
        expect(metrics.bodyContainmentViolations).toEqual([]);
      } else {
        expect(metrics.groupOverlaps).toEqual(
          expect.arrayContaining([
            expect.objectContaining({ a: 'Tall Prompt Ladder', b: 'Compact Reference' }),
            expect.objectContaining({ a: 'Tall Prompt Ladder', b: 'Negative Matrix' }),
          ]),
        );
      }
      expect(metrics.nodeOverlaps).toEqual([]);
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
        JSON.stringify({ layout: metrics, connection: connectionMetrics }, null, 2),
        'utf-8',
      );
    });
  }
});

import fs from 'node:fs';
import path from 'node:path';

import { expect, test } from '@playwright/test';

import { listWorkspaceNames, loadWorkspaceByName } from './helpers/comfyui';
import { readGraphMetrics } from './helpers/graph_metrics';

const runtimeOutputsPath = path.resolve(process.cwd(), '../.runtime/terraform-outputs.json');
const browserArtifactsDir = path.resolve(process.cwd(), '../artifacts/browser');

type ScenarioExpectation = {
  workspace_name: string;
  node_count: number;
  group_count: number;
  link_count: number;
  max_in_degree: number;
  max_out_degree: number;
  require_groups: boolean;
};

function readScenarioExpectations(): Record<string, ScenarioExpectation> {
  const runtimeOutputs = JSON.parse(fs.readFileSync(runtimeOutputsPath, 'utf-8')) as {
    scenario_expectations: { value: Record<string, ScenarioExpectation> };
  };
  return runtimeOutputs.scenario_expectations.value;
}

test('verifies release candidate workflows and translations in the real ComfyUI canvas', async ({ page }) => {
  fs.mkdirSync(browserArtifactsDir, { recursive: true });

  const expectations = readScenarioExpectations();
  const scenarioNames = Object.keys(expectations).sort();

  await page.goto('/');
  await page.waitForLoadState('networkidle');

  const discoveredNames = await listWorkspaceNames(page);
  for (const scenarioName of scenarioNames) {
    expect(discoveredNames).toContain(expectations[scenarioName].workspace_name);
  }

  for (const scenarioName of scenarioNames) {
    await test.step(`verify ${scenarioName}`, async () => {
      const expectation = expectations[scenarioName];

      await loadWorkspaceByName(page, expectation.workspace_name);
      await page.waitForTimeout(200);

      const metrics = await readGraphMetrics(page);
      expect(metrics.nodeCount).toBe(expectation.node_count);
      expect(metrics.groupCount).toBe(expectation.group_count);
      expect(metrics.linkCount).toBe(expectation.link_count);
      expect(metrics.maxInDegree).toBe(expectation.max_in_degree);
      expect(metrics.maxOutDegree).toBe(expectation.max_out_degree);
      expect(metrics.brokenLinks).toEqual([]);
      expect(metrics.nodesWithInvalidGeometry).toEqual([]);

      if (expectation.require_groups) {
        expect(metrics.groupCount).toBeGreaterThan(0);
      }

      await page.screenshot({
        path: path.join(browserArtifactsDir, `${scenarioName}.png`),
        fullPage: true,
      });
      fs.writeFileSync(
        path.join(browserArtifactsDir, `${scenarioName}.metrics.json`),
        JSON.stringify(metrics, null, 2),
        'utf-8',
      );
    });
  }
});

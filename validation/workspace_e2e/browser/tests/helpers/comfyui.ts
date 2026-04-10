import { expect, type Page } from '@playwright/test';

export type LoadedWorkspace = {
  name: string;
  nodeCount: number;
  groupCount: number;
  nodeIds: number[];
};

export async function listWorkspaceNames(page: Page): Promise<string[]> {
  return page.evaluate(async () => {
    const subgraphs = (await fetch('/global_subgraphs').then((response) => response.json())) as Record<
      string,
      { name: string }
    >;

    return Object.values(subgraphs)
      .map((entry) => entry.name)
      .sort();
  });
}

export async function loadWorkspaceByName(page: Page, workspaceName: string): Promise<LoadedWorkspace> {
  const loaded = await page.evaluate(async (name) => {
    const subgraphs = (await fetch('/global_subgraphs').then((response) => response.json())) as Record<
      string,
      { name: string }
    >;
    const match = Object.entries(subgraphs).find((entry) => entry[1].name === name);
    if (!match) {
      throw new Error(`workspace ${name} not found in /global_subgraphs`);
    }

    const payload = (await fetch('/global_subgraphs/' + match[0]).then((response) => response.json())) as {
      data: string;
    };
    const parsed = JSON.parse(payload.data);

    await window.app.loadGraphData(parsed);

    const graph = window.app.graph;
    return {
      name,
      nodeCount: graph?._nodes?.length ?? 0,
      groupCount: graph?._groups?.length ?? 0,
      nodeIds: (graph?._nodes ?? []).map((node: { id: number }) => node.id),
    };
  }, workspaceName);

  await expect
    .poll(
      async () =>
        page.evaluate(() => ({
          nodeCount: window.app.graph?._nodes?.length ?? 0,
          groupCount: window.app.graph?._groups?.length ?? 0,
        })),
      { message: `wait for ${workspaceName} to load into the canvas` },
    )
    .toEqual({
      nodeCount: loaded.nodeCount,
      groupCount: loaded.groupCount,
    });

  return loaded;
}

export async function getFirstNodePosition(page: Page): Promise<[number, number]> {
  return page.evaluate(() => {
    const firstNode = window.app.graph?._nodes?.[0];
    if (!firstNode) {
      throw new Error('graph has no nodes');
    }
    return [firstNode.pos[0], firstNode.pos[1]] as [number, number];
  });
}

export async function dragFirstNode(page: Page, deltaX: number, deltaY: number): Promise<[number, number]> {
  const dragPoint = await page.evaluate(() => {
    const firstNode = window.app.graph?._nodes?.[0];
    if (!firstNode) {
      throw new Error('graph has no nodes');
    }

    const clientPoint = window.app.canvasPosToClientPos([firstNode.pos[0] + 40, firstNode.pos[1] + 40]);
    return [clientPoint[0], clientPoint[1]] as [number, number];
  });

  await page.mouse.move(dragPoint[0], dragPoint[1]);
  await page.mouse.down();
  await page.mouse.move(dragPoint[0] + deltaX, dragPoint[1] + deltaY, { steps: 10 });
  await page.mouse.up();

  return getFirstNodePosition(page);
}

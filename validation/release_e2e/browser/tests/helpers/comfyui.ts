import { expect, type Page } from '@playwright/test';

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

export async function loadWorkspaceByName(page: Page, workspaceName: string): Promise<void> {
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
      nodeCount: graph?._nodes?.length ?? 0,
      groupCount: graph?._groups?.length ?? 0,
      linkCount: Object.keys(graph?.links ?? {}).length,
    };
  }, workspaceName);

  await expect
    .poll(
      async () =>
        page.evaluate(() => ({
          nodeCount: window.app.graph?._nodes?.length ?? 0,
          groupCount: window.app.graph?._groups?.length ?? 0,
          linkCount: Object.keys(window.app.graph?.links ?? {}).length,
        })),
      { message: `wait for ${workspaceName} to load into the canvas` },
    )
    .toEqual(loaded);
}

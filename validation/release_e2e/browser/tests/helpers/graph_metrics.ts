import type { Page } from '@playwright/test';

export type GraphMetrics = {
  nodeCount: number;
  groupCount: number;
  linkCount: number;
  brokenLinks: number[];
  nodesWithInvalidGeometry: number[];
  maxInDegree: number;
  maxOutDegree: number;
};

export async function readGraphMetrics(page: Page): Promise<GraphMetrics> {
  return page.evaluate(() => {
    const graph = window.app.graph;
    const nodes = graph?._nodes ?? [];
    const links = graph?.links ?? {};
    const nodesById = new Map(nodes.map((node: any) => [node.id, node]));

    const brokenLinks: number[] = [];
    for (const [rawLinkId, link] of Object.entries(links as Record<string, any>)) {
      if (!nodesById.has(link.origin_id) || !nodesById.has(link.target_id)) {
        brokenLinks.push(Number(rawLinkId));
      }
    }

    const nodesWithInvalidGeometry = nodes
      .filter((node: any) => {
        const [x, y] = node.pos ?? [NaN, NaN];
        const [width, height] = node.size ?? [NaN, NaN];
        return ![x, y, width, height].every((value) => Number.isFinite(value));
      })
      .map((node: any) => node.id);

    const maxInDegree = nodes.reduce((max: number, node: any) => {
      const degree = (node.inputs ?? []).reduce(
        (count: number, input: any) => count + (input?.link !== null && input?.link !== undefined ? 1 : 0),
        0,
      );
      return Math.max(max, degree);
    }, 0);

    const maxOutDegree = nodes.reduce((max: number, node: any) => {
      const degree = (node.outputs ?? []).reduce(
        (count: number, output: any) => count + ((output?.links ?? []).length || 0),
        0,
      );
      return Math.max(max, degree);
    }, 0);

    return {
      nodeCount: nodes.length,
      groupCount: (graph?._groups ?? []).length,
      linkCount: Object.keys(links).length,
      brokenLinks,
      nodesWithInvalidGeometry,
      maxInDegree,
      maxOutDegree,
    };
  });
}

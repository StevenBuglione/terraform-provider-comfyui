import type { Page } from '@playwright/test';

export type ConnectionMetrics = {
  linkCount: number;
  brokenLinks: number[];
  maxInDegree: number;
  maxOutDegree: number;
};

export async function readConnectionMetrics(page: Page): Promise<ConnectionMetrics> {
  return page.evaluate(() => {
    const graph = window.app.graph;
    const nodes = graph?._nodes ?? [];
    const links = graph?.links ?? {};
    const nodeIds = new Set(nodes.map((node: any) => node.id));

    const brokenLinks = Object.entries(links as Record<string, any>)
      .filter(([, link]) => !nodeIds.has(link.origin_id) || !nodeIds.has(link.target_id))
      .map(([linkId]) => Number(linkId));

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
      linkCount: Object.keys(links).length,
      brokenLinks,
      maxInDegree,
      maxOutDegree,
    };
  });
}

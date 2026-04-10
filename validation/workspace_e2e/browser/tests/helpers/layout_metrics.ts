import type { Page } from '@playwright/test';

export type LayoutMetrics = {
  nodeCount: number;
  groupCount: number;
  ungroupedNodes: number[];
  groupOverlaps: Array<{ a: string; b: string; area: number }>;
  nodeOverlaps: Array<{ a: number; b: number; area: number }>;
  groups: Array<{
    title: string;
    x: number;
    y: number;
    width: number;
    height: number;
    fullyVisible: boolean;
  }>;
};

export async function readLayoutMetrics(page: Page): Promise<LayoutMetrics> {
  return page.evaluate(() => {
    const graph = window.app.graph;
    const canvasRect = window.app.canvas.canvas.getBoundingClientRect();

    const normalizeRect = (rect: [number, number, number, number]) => ({
      x: rect[0],
      y: rect[1],
      width: rect[2],
      height: rect[3],
      right: rect[0] + rect[2],
      bottom: rect[1] + rect[3],
    });

    const overlapArea = (
      leftRect: { x: number; y: number; right: number; bottom: number },
      rightRect: { x: number; y: number; right: number; bottom: number },
    ) => {
      const x = Math.max(0, Math.min(leftRect.right, rightRect.right) - Math.max(leftRect.x, rightRect.x));
      const y = Math.max(0, Math.min(leftRect.bottom, rightRect.bottom) - Math.max(leftRect.y, rightRect.y));
      return x * y;
    };

    const groups = (graph?._groups ?? []).map((group: { title?: string; _bounding: [number, number, number, number] }, index: number) => {
      const rect = normalizeRect(group._bounding);
      const topLeft = window.app.canvasPosToClientPos([rect.x, rect.y]);
      const bottomRight = window.app.canvasPosToClientPos([rect.right, rect.bottom]);

      return {
        index,
        title: group.title || `group-${index + 1}`,
        rect,
        fullyVisible:
          topLeft[0] >= canvasRect.left &&
          topLeft[1] >= canvasRect.top &&
          bottomRight[0] <= canvasRect.right &&
          bottomRight[1] <= canvasRect.bottom,
      };
    });

    const nodes = (graph?._nodes ?? []).map((node: { id: number; pos: [number, number]; size: [number, number] }) => {
      const rect = normalizeRect([node.pos[0], node.pos[1], node.size[0], node.size[1]]);
      const centerX = rect.x + rect.width / 2;
      const centerY = rect.y + rect.height / 2;
      const group = groups.find(
        (candidate) =>
          centerX >= candidate.rect.x &&
          centerX <= candidate.rect.right &&
          centerY >= candidate.rect.y &&
          centerY <= candidate.rect.bottom,
      );

      return {
        id: node.id,
        rect,
        groupIndex: group?.index ?? null,
      };
    });

    const groupOverlaps: Array<{ a: string; b: string; area: number }> = [];
    for (let left = 0; left < groups.length; left++) {
      for (let right = left + 1; right < groups.length; right++) {
        const area = overlapArea(groups[left].rect, groups[right].rect);
        if (area > 0) {
          groupOverlaps.push({
            a: groups[left].title,
            b: groups[right].title,
            area,
          });
        }
      }
    }

    const nodeOverlaps: Array<{ a: number; b: number; area: number }> = [];
    for (let left = 0; left < nodes.length; left++) {
      for (let right = left + 1; right < nodes.length; right++) {
        if (nodes[left].groupIndex === null || nodes[right].groupIndex === null) {
          continue;
        }
        if (nodes[left].groupIndex === nodes[right].groupIndex) {
          continue;
        }

        const area = overlapArea(nodes[left].rect, nodes[right].rect);
        if (area > 0) {
          nodeOverlaps.push({
            a: nodes[left].id,
            b: nodes[right].id,
            area,
          });
        }
      }
    }

    return {
      nodeCount: nodes.length,
      groupCount: groups.length,
      ungroupedNodes: nodes.filter((node) => node.groupIndex === null).map((node) => node.id),
      groupOverlaps,
      nodeOverlaps,
      groups: groups.map((group) => ({
        title: group.title,
        x: group.rect.x,
        y: group.rect.y,
        width: group.rect.width,
        height: group.rect.height,
        fullyVisible: group.fullyVisible,
      })),
    };
  });
}

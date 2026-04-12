import fs from 'node:fs';
import path from 'node:path';

import { chromium } from 'playwright';

const baseURL = process.env.UI_HINTS_BASE_URL ?? 'http://127.0.0.1:8188';
const outputPath = process.env.UI_HINTS_OUTPUT_PATH ?? path.resolve(process.cwd(), 'node_ui_hints.json');
const comfyuiCommitSHA = process.env.UI_HINTS_COMFYUI_COMMIT_SHA ?? 'unknown';
const comfyuiVersion = process.env.UI_HINTS_COMFYUI_VERSION ?? 'unknown';

function toFiniteNumber(value) {
  return Number.isFinite(value) ? Number(value) : null;
}

async function extractNodeUIHints() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  try {
    await page.goto(baseURL);
    await page.waitForLoadState('networkidle');

    const result = await page.evaluate(async () => {
      const objectInfo = await fetch('/object_info').then((response) => response.json());
      const nodeTypes = Object.keys(objectInfo).sort();
      const nodes = {};
      const failures = [];

      for (const nodeType of nodeTypes) {
        try {
          const node = window.LiteGraph?.createNode(nodeType);
          if (!node) {
            throw new Error('LiteGraph.createNode returned null');
          }

          const computedSize =
            typeof node.computeSize === 'function'
              ? Array.from(node.computeSize())
              : [];
          const nodeSize = Array.from(node.size ?? []);

          const widgets = {};
          for (const [index, widget] of (node.widgets ?? []).entries()) {
            const widgetKey = widget?.name || `widget_${index}`;
            const widgetComputedSize =
              typeof widget?.computeSize === 'function' ? Array.from(widget.computeSize()) : [];
            const minNodeSize = Array.isArray(widget?.options?.minNodeSize)
              ? Array.from(widget.options.minNodeSize)
              : [];

            widgets[widgetKey] = {
              widget_type: widget?.type ?? 'unknown',
              has_compute_size: typeof widget?.computeSize === 'function',
              computed_width: Number.isFinite(widgetComputedSize[0]) ? widgetComputedSize[0] : null,
              computed_height: Number.isFinite(widgetComputedSize[1]) ? widgetComputedSize[1] : null,
              min_node_width: Number.isFinite(minNodeSize[0]) ? minNodeSize[0] : null,
              min_node_height: Number.isFinite(minNodeSize[1]) ? minNodeSize[1] : null,
            };
          }

          nodes[nodeType] = {
            node_type: nodeType,
            display_name: node.title ?? nodeType,
            min_width: Number.isFinite(nodeSize[0]) ? nodeSize[0] : null,
            min_height: Number.isFinite(nodeSize[1]) ? nodeSize[1] : null,
            computed_width: Number.isFinite(computedSize[0]) ? computedSize[0] : null,
            computed_height: Number.isFinite(computedSize[1]) ? computedSize[1] : null,
            widgets,
          };
        } catch (error) {
          failures.push({
            node_type: nodeType,
            message: String(error),
          });
        }
      }

      return {
        total_nodes: nodeTypes.length,
        nodes,
        failures,
      };
    });

    if (result.failures.length > 0) {
      throw new Error(
        `failed to extract UI hints for ${result.failures.length} node types: ${result.failures
          .slice(0, 10)
          .map((failure) => `${failure.node_type}: ${failure.message}`)
          .join('; ')}`,
      );
    }

    const normalizedNodes = {};
    for (const [nodeType, hint] of Object.entries(result.nodes)) {
      const normalizedWidgets = {};
      for (const [widgetName, widgetHint] of Object.entries(hint.widgets)) {
        normalizedWidgets[widgetName] = {
          widget_type: widgetHint.widget_type,
          has_compute_size: widgetHint.has_compute_size,
          computed_width: toFiniteNumber(widgetHint.computed_width),
          computed_height: toFiniteNumber(widgetHint.computed_height),
          min_node_width: toFiniteNumber(widgetHint.min_node_width),
          min_node_height: toFiniteNumber(widgetHint.min_node_height),
        };
      }

      normalizedNodes[nodeType] = {
        node_type: hint.node_type,
        display_name: hint.display_name,
        min_width: toFiniteNumber(hint.min_width),
        min_height: toFiniteNumber(hint.min_height),
        computed_width: toFiniteNumber(hint.computed_width),
        computed_height: toFiniteNumber(hint.computed_height),
        widgets: normalizedWidgets,
      };
    }

    const output = {
      version: '1.0.0',
      extracted_at: new Date().toISOString(),
      comfyui_commit_sha: comfyuiCommitSHA,
      comfyui_version: comfyuiVersion,
      total_nodes: result.total_nodes,
      failed_nodes: [],
      nodes: normalizedNodes,
    };

    fs.writeFileSync(outputPath, `${JSON.stringify(output, null, 2)}\n`, 'utf-8');
  } finally {
    await browser.close();
  }
}

await extractNodeUIHints();

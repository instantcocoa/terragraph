import type { GraphNode, GraphEdge, NodeKind } from './types';
import { detectGroups } from './groups';

const NODE_WIDTH = 2.8; // inches for graphviz
const NODE_HEIGHT = 1.0;
const DPI = 72;

const KIND_TIER: Record<string, number> = {
	resource: 0, module: 0, data: 1, local: 2, variable: 3, output: 4
};
const LEFT_COLUMN = new Set(['provider', 'terraform']);

// Viz.js instance (lazy loaded)
let vizPromise: Promise<any> | null = null;

async function getViz() {
	if (!vizPromise) {
		vizPromise = import('@viz-js/viz').then((m) => m.instance());
	}
	return vizPromise;
}

/**
 * Synchronous fallback: simple tier grid.
 */
export function layoutGraph(
	nodes: Array<{ id: string; kind?: string; name?: string }>,
	_edges: Array<{ source: string; target: string }>
): Map<string, { x: number; y: number }> {
	if (nodes.length === 0) return new Map();
	const positions = new Map<string, { x: number; y: number }>();
	const mainNodes = nodes.filter((n) => !LEFT_COLUMN.has(n.kind ?? ''));
	const leftNodes = nodes.filter((n) => LEFT_COLUMN.has(n.kind ?? ''));

	const tiers = new Map<number, typeof mainNodes>();
	for (const node of mainNodes) {
		const t = KIND_TIER[node.kind ?? ''] ?? 2;
		if (!tiers.has(t)) tiers.set(t, []);
		tiers.get(t)!.push(node);
	}

	let y = 0;
	for (const tierKey of [...tiers.keys()].sort()) {
		const tierNodes = tiers.get(tierKey)!;
		tierNodes.sort((a, b) => (a.name ?? a.id).localeCompare(b.name ?? b.id));
		const totalWidth = tierNodes.length * 250;
		const startX = -totalWidth / 2;
		for (let i = 0; i < tierNodes.length; i++) {
			positions.set(tierNodes[i].id, { x: startX + i * 250, y });
		}
		y += 160;
	}

	let minX = Infinity;
	for (const p of positions.values()) minX = Math.min(minX, p.x);
	if (!isFinite(minX)) minX = 0;
	for (let i = 0; i < leftNodes.length; i++) {
		positions.set(leftNodes[i].id, { x: minX - 300, y: i * 100 });
	}
	return positions;
}

/**
 * Async Graphviz layout with subgraph clusters for grouped resources.
 * Uses the `dot` engine for hierarchical layout.
 */
export async function layoutGraphAsync(
	nodes: Array<{ id: string; kind?: string; name?: string }>,
	edges: Array<{ source: string; target: string }>
): Promise<Map<string, { x: number; y: number }>> {
	if (nodes.length === 0) return new Map();

	const positions = new Map<string, { x: number; y: number }>();
	const mainNodes = nodes.filter((n) => !LEFT_COLUMN.has(n.kind ?? ''));
	const leftNodes = nodes.filter((n) => LEFT_COLUMN.has(n.kind ?? ''));

	if (mainNodes.length === 0) return positions;

	const nodeTier = new Map<string, number>();
	for (const node of mainNodes) {
		nodeTier.set(node.id, KIND_TIER[node.kind ?? ''] ?? 2);
	}

	// Detect groups for clustering
	const fullNodes = nodes as GraphNode[];
	const fullEdges = edges as GraphEdge[];
	const { groups, groupedIds } = detectGroups(fullNodes, fullEdges);

	const mainIds = new Set(mainNodes.map((n) => n.id));

	// Sanitize ID for Graphviz (replace dots with underscores)
	const gvId = (id: string) => id.replace(/[.\-]/g, '_');

	// Build DOT string
	let dot = 'digraph {\n';
	dot += '  rankdir=TB;\n';
	dot += '  splines=ortho;\n';
	dot += '  nodesep=0.5;\n';
	dot += '  ranksep=1.0;\n';
	dot += `  node [shape=box, width=${NODE_WIDTH}, height=${NODE_HEIGHT}, fixedsize=true];\n`;
	dot += '  newrank=true;\n';

	// Rank constraints by tier
	const tiers = new Map<number, string[]>();
	for (const node of mainNodes) {
		if (groupedIds.has(node.id)) continue;
		const t = nodeTier.get(node.id) ?? 2;
		if (!tiers.has(t)) tiers.set(t, []);
		tiers.get(t)!.push(node.id);
	}

	for (const [, tierNodeIds] of tiers) {
		dot += `  { rank=same; ${tierNodeIds.map(gvId).join('; ')}; }\n`;
	}

	// Add cluster subgraphs for grouped resources
	for (const group of groups) {
		dot += `  subgraph cluster_${gvId(group.primary.id)} {\n`;
		dot += `    style=rounded;\n`;
		dot += `    color="#2f3146";\n`;
		dot += `    bgcolor="#1a1b2610";\n`;
		for (const member of group.members) {
			dot += `    ${gvId(member.id)} [label="${member.name}"];\n`;
		}
		dot += `  }\n`;
	}

	// Add standalone nodes (not in any group)
	for (const node of mainNodes) {
		if (groupedIds.has(node.id)) continue;
		if (groups.some((g) => g.primary.id === node.id)) continue;
		dot += `  ${gvId(node.id)} [label="${node.name}"];\n`;
	}

	// Add edges - always downward in tier order
	for (const edge of edges) {
		if (!mainIds.has(edge.source) || !mainIds.has(edge.target)) continue;
		const srcTier = nodeTier.get(edge.source) ?? 2;
		const tgtTier = nodeTier.get(edge.target) ?? 2;

		if (srcTier <= tgtTier) {
			dot += `  ${gvId(edge.source)} -> ${gvId(edge.target)};\n`;
		} else {
			dot += `  ${gvId(edge.target)} -> ${gvId(edge.source)};\n`;
		}
	}

	// Invisible tier chain edges
	const tierKeys = [...tiers.keys()].sort();
	for (let i = 0; i < tierKeys.length - 1; i++) {
		const cur = tiers.get(tierKeys[i])!;
		const next = tiers.get(tierKeys[i + 1])!;
		dot += `  ${gvId(cur[0])} -> ${gvId(next[0])} [style=invis, weight=100];\n`;
	}

	dot += '}\n';

	try {
		const viz = await getViz();
		const result = viz.renderJSON(dot);

		// Parse JSON output - Graphviz JSON format has objects array
		if (result && result.objects) {
			for (const obj of result.objects) {
				if (obj.name && obj.pos) {
					// pos is "x,y" in points
					const [xStr, yStr] = obj.pos.split(',');
					const x = parseFloat(xStr);
					const y = parseFloat(yStr);

					// Find the original node ID from the sanitized name
					const originalId = mainNodes.find((n) => gvId(n.id) === obj.name)?.id;
					if (originalId) {
						positions.set(originalId, { x, y: -y }); // Graphviz Y is inverted
					}
				}
			}
		}

		// If JSON parsing didn't work well, try plain format
		if (positions.size === 0) {
			const plainResult = viz.render(dot, { format: 'plain' });
			if (plainResult.status === 'success') {
				const lines = plainResult.output.split('\n');
				for (const line of lines) {
					if (line.startsWith('node ')) {
						const parts = line.split(/\s+/);
						const name = parts[1];
						const x = parseFloat(parts[2]) * DPI;
						const y = -parseFloat(parts[3]) * DPI; // Invert Y
						const originalId = mainNodes.find((n) => gvId(n.id) === name)?.id;
						if (originalId) {
							positions.set(originalId, { x, y });
						}
					}
				}
			}
		}
	} catch (e) {
		console.error('Graphviz layout failed:', e);
		return layoutGraph(nodes, edges);
	}

	// Left column
	if (leftNodes.length > 0 && positions.size > 0) {
		let minX = Infinity;
		let firstY = 0;
		for (const pos of positions.values()) {
			if (pos.x < minX) { minX = pos.x; firstY = pos.y; }
		}

		const leftX = minX - NODE_WIDTH * DPI - 60;
		const sortedLeft = [...leftNodes].sort((a, b) =>
			(a.name ?? a.id).localeCompare(b.name ?? b.id)
		);
		for (let i = 0; i < sortedLeft.length; i++) {
			positions.set(sortedLeft[i].id, {
				x: leftX,
				y: firstY + i * 90
			});
		}
	}

	return positions;
}

import { Graph, layout } from '@dagrejs/dagre';
import type { GraphLabel } from '@dagrejs/dagre';

const NODE_WIDTH = 220;
const NODE_HEIGHT = 80;
const RANK_SEP = 100;
const NODE_SEP = 50;

// Rank order: top to bottom logical flow
// Lower number = higher on screen
const KIND_RANK: Record<string, number> = {
	terraform: 0,
	provider: 0,
	variable: 1,
	local: 2,
	data: 3,
	resource: 4,
	module: 4,
	output: 5
};

/**
 * Computes a deterministic top-to-bottom DAG layout.
 *
 * - Nodes are grouped by kind into vertical tiers (variables top, resources middle, outputs bottom)
 * - Connected nodes are placed near each other
 * - Alphabetical sorting within tiers ensures consistent layout
 * - Same input always produces the same output
 */
export function layoutGraph(
	nodes: Array<{ id: string; kind?: string; name?: string }>,
	edges: Array<{ source: string; target: string }>
): Map<string, { x: number; y: number }> {
	if (nodes.length === 0) return new Map();

	const g = new Graph<GraphLabel>();

	g.setGraph({
		rankdir: 'TB',
		ranksep: RANK_SEP,
		nodesep: NODE_SEP,
		align: 'UL'
	});

	g.setDefaultEdgeLabel(() => ({}));

	// Sort nodes alphabetically by kind-rank then name for deterministic ordering
	const sortedNodes = [...nodes].sort((a, b) => {
		const rankA = KIND_RANK[a.kind as string] ?? 4;
		const rankB = KIND_RANK[b.kind as string] ?? 4;
		if (rankA !== rankB) return rankA - rankB;
		return (a.name as string ?? a.id).localeCompare(b.name as string ?? b.id);
	});

	for (const node of sortedNodes) {
		g.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
	}

	// Add invisible edges between tiers to enforce vertical ordering.
	// Connect one representative node from each tier to the next tier.
	const tiers = new Map<number, string[]>();
	for (const node of sortedNodes) {
		const rank = KIND_RANK[node.kind as string] ?? 4;
		if (!tiers.has(rank)) tiers.set(rank, []);
		tiers.get(rank)!.push(node.id);
	}

	const tierKeys = [...tiers.keys()].sort();
	for (let i = 0; i < tierKeys.length - 1; i++) {
		const currentTier = tiers.get(tierKeys[i])!;
		const nextTier = tiers.get(tierKeys[i + 1])!;
		// Add a hidden edge from the first node of each tier to enforce ordering
		g.setEdge(currentTier[0], nextTier[0], { weight: 0, minlen: 1, style: 'invis' });
	}

	// Add real edges - these pull connected nodes together
	for (const edge of edges) {
		// Check both nodes exist
		if (g.hasNode(edge.source) && g.hasNode(edge.target)) {
			g.setEdge(edge.source, edge.target, { weight: 2 });
		}
	}

	layout(g);

	const positions = new Map<string, { x: number; y: number }>();

	for (const node of sortedNodes) {
		const laid = g.node(node.id);
		if (laid) {
			positions.set(node.id, { x: laid.x, y: laid.y });
		}
	}

	return positions;
}

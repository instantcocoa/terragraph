import { Graph, layout } from '@dagrejs/dagre';
import type { GraphLabel } from '@dagrejs/dagre';

const NODE_WIDTH = 200;
const NODE_HEIGHT = 72;

const KIND_TIER: Record<string, number> = {
	resource: 0,
	module: 0,
	data: 1,
	local: 2,
	variable: 3,
	output: 4
};

const LEFT_COLUMN = new Set(['provider', 'terraform']);

export function layoutGraph(
	nodes: Array<{ id: string; kind?: string; name?: string }>,
	edges: Array<{ source: string; target: string }>
): Map<string, { x: number; y: number }> {
	if (nodes.length === 0) return new Map();

	const positions = new Map<string, { x: number; y: number }>();
	const mainNodes = nodes.filter((n) => !LEFT_COLUMN.has(n.kind ?? ''));
	const leftNodes = nodes.filter((n) => LEFT_COLUMN.has(n.kind ?? ''));

	if (mainNodes.length > 0) {
		const g = new Graph<GraphLabel>();
		g.setGraph({ rankdir: 'TB', ranksep: 90, nodesep: 35 });
		g.setDefaultEdgeLabel(() => ({}));

		const nodeTier = new Map<string, number>();
		for (const node of mainNodes) {
			nodeTier.set(node.id, KIND_TIER[node.kind ?? ''] ?? 2);
		}

		// Sort deterministically
		const sorted = [...mainNodes].sort((a, b) => {
			const tA = KIND_TIER[a.kind ?? ''] ?? 2;
			const tB = KIND_TIER[b.kind ?? ''] ?? 2;
			if (tA !== tB) return tA - tB;
			return (a.name ?? a.id).localeCompare(b.name ?? b.id);
		});

		for (const node of sorted) {
			g.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
		}

		// Add all edges, reversing any that go upward in tier order
		const mainIds = new Set(mainNodes.map((n) => n.id));
		for (const edge of edges) {
			if (!mainIds.has(edge.source) || !mainIds.has(edge.target)) continue;
			const srcTier = nodeTier.get(edge.source) ?? 2;
			const tgtTier = nodeTier.get(edge.target) ?? 2;

			if (srcTier <= tgtTier) {
				g.setEdge(edge.source, edge.target);
			} else {
				g.setEdge(edge.target, edge.source);
			}
		}

		// Chain one node per tier to enforce ordering when no edges exist between tiers
		const tiers = new Map<number, string>();
		for (const node of sorted) {
			const t = nodeTier.get(node.id) ?? 2;
			if (!tiers.has(t)) tiers.set(t, node.id);
		}
		const tierKeys = [...tiers.keys()].sort();
		for (let i = 0; i < tierKeys.length - 1; i++) {
			const a = tiers.get(tierKeys[i])!;
			const b = tiers.get(tierKeys[i + 1])!;
			if (!g.hasEdge(a, b)) {
				g.setEdge(a, b, { weight: 0, minlen: 1 });
			}
		}

		try {
			layout(g);
			for (const node of sorted) {
				const laid = g.node(node.id);
				if (laid) positions.set(node.id, { x: laid.x, y: laid.y });
			}
		} catch {
			// Fallback: simple grid
			for (let i = 0; i < sorted.length; i++) {
				positions.set(sorted[i].id, {
					x: (i % 4) * (NODE_WIDTH + 40),
					y: Math.floor(i / 4) * (NODE_HEIGHT + 40)
				});
			}
		}
	}

	// Left column
	if (leftNodes.length > 0) {
		let minX = Infinity;
		let firstY = 0;
		for (const pos of positions.values()) {
			if (pos.x < minX) { minX = pos.x; firstY = pos.y; }
		}
		if (!isFinite(minX)) { minX = NODE_WIDTH; firstY = 0; }

		const leftX = minX - NODE_WIDTH - 80;
		const sortedLeft = [...leftNodes].sort((a, b) =>
			(a.name ?? a.id).localeCompare(b.name ?? b.id)
		);
		for (let i = 0; i < sortedLeft.length; i++) {
			positions.set(sortedLeft[i].id, {
				x: leftX,
				y: firstY + i * (NODE_HEIGHT + 20)
			});
		}
	}

	return positions;
}

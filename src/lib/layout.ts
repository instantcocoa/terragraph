import ELK from 'elkjs/lib/elk.bundled.js';

const NODE_WIDTH = 200;
const NODE_HEIGHT = 72;

const KIND_LAYER: Record<string, number> = {
	resource: 0,
	module: 0,
	data: 1,
	local: 2,
	variable: 3,
	output: 4
};

const LEFT_COLUMN = new Set(['provider', 'terraform']);

let elk: InstanceType<typeof ELK> | null = null;
function getELK(): InstanceType<typeof ELK> {
	if (!elk) elk = new ELK();
	return elk;
}

/**
 * Synchronous fallback layout: simple tier-based grid.
 */
export function layoutGraph(
	nodes: Array<{ id: string; kind?: string; name?: string }>,
	edges: Array<{ source: string; target: string }>
): Map<string, { x: number; y: number }> {
	void edges; // used by async version
	if (nodes.length === 0) return new Map();

	const positions = new Map<string, { x: number; y: number }>();
	const mainNodes = nodes.filter((n) => !LEFT_COLUMN.has(n.kind ?? ''));
	const leftNodes = nodes.filter((n) => LEFT_COLUMN.has(n.kind ?? ''));

	const tiers = new Map<number, typeof mainNodes>();
	for (const node of mainNodes) {
		const t = KIND_LAYER[node.kind ?? ''] ?? 2;
		if (!tiers.has(t)) tiers.set(t, []);
		tiers.get(t)!.push(node);
	}

	let y = 0;
	for (const tierKey of [...tiers.keys()].sort()) {
		const tierNodes = tiers.get(tierKey)!;
		tierNodes.sort((a, b) => (a.name ?? a.id).localeCompare(b.name ?? b.id));
		const totalWidth = tierNodes.length * (NODE_WIDTH + 35);
		const startX = -totalWidth / 2;
		for (let i = 0; i < tierNodes.length; i++) {
			positions.set(tierNodes[i].id, { x: startX + i * (NODE_WIDTH + 35), y });
		}
		y += NODE_HEIGHT + 90;
	}

	let minX = Infinity;
	for (const p of positions.values()) minX = Math.min(minX, p.x);
	if (!isFinite(minX)) minX = 0;
	for (let i = 0; i < leftNodes.length; i++) {
		positions.set(leftNodes[i].id, {
			x: minX - NODE_WIDTH - 80,
			y: i * (NODE_HEIGHT + 20)
		});
	}

	return positions;
}

/**
 * Async ELK layout - returns a promise with optimized positions.
 * Uses ELK layered algorithm with orthogonal edge routing and
 * crossing minimization.
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
		nodeTier.set(node.id, KIND_LAYER[node.kind ?? ''] ?? 2);
	}

	const mainIds = new Set(mainNodes.map((n) => n.id));

	const sorted = [...mainNodes].sort((a, b) => {
		const tA = KIND_LAYER[a.kind ?? ''] ?? 2;
		const tB = KIND_LAYER[b.kind ?? ''] ?? 2;
		if (tA !== tB) return tA - tB;
		return (a.name ?? a.id).localeCompare(b.name ?? b.id);
	});

	const elkEdges = edges
		.filter((e) => mainIds.has(e.source) && mainIds.has(e.target))
		.map((e, i) => {
			const srcTier = nodeTier.get(e.source) ?? 2;
			const tgtTier = nodeTier.get(e.target) ?? 2;
			if (srcTier <= tgtTier) {
				return { id: `e${i}`, sources: [e.source], targets: [e.target] };
			} else {
				return { id: `e${i}`, sources: [e.target], targets: [e.source] };
			}
		});

	const elkGraph = {
		id: 'root',
		layoutOptions: {
			'elk.algorithm': 'layered',
			'elk.direction': 'DOWN',
			'elk.layered.spacing.nodeNodeBetweenLayers': '90',
			'elk.spacing.nodeNode': '35',
			'elk.layered.crossingMinimization.strategy': 'LAYER_SWEEP',
			'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
			'elk.layered.considerModelOrder.strategy': 'NODES_AND_EDGES',
			'elk.edgeRouting': 'ORTHOGONAL',
			'elk.layered.spacing.edgeNodeBetweenLayers': '20',
			'elk.layered.spacing.edgeEdgeBetweenLayers': '15'
		},
		children: sorted.map((node) => ({
			id: node.id,
			width: NODE_WIDTH,
			height: NODE_HEIGHT
		})),
		edges: elkEdges
	};

	try {
		const result = await getELK().layout(elkGraph);
		if (result.children) {
			for (const child of result.children) {
				positions.set(child.id, { x: child.x ?? 0, y: child.y ?? 0 });
			}
		}
	} catch {
		return layoutGraph(nodes, edges);
	}

	// Left column
	if (leftNodes.length > 0) {
		let minX = Infinity;
		let firstY = 0;
		for (const pos of positions.values()) {
			if (pos.x < minX) { minX = pos.x; firstY = pos.y; }
		}
		if (!isFinite(minX)) { minX = 0; firstY = 0; }

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

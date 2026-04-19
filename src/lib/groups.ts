import type { GraphNode, GraphEdge } from './types';

export interface NodeGroup {
	id: string;
	label: string;
	primary: GraphNode;
	members: GraphNode[];
}

/**
 * Detect tightly-coupled resource groups for compound node display.
 *
 * A resource gets nested inside another if:
 * - Only ONE resource/data/module references it (ignoring outputs, variables, locals)
 * - It's itself a resource, data source, or module
 * - It doesn't have too many outgoing connections (not a hub)
 */
export function detectGroups(
	nodes: GraphNode[],
	edges: GraphEdge[]
): { groups: NodeGroup[]; groupedIds: Set<string> } {
	const groupableKinds = new Set(['resource', 'data', 'module']);
	const nodeMap = new Map(nodes.map((n) => [n.id, n]));

	// Only count incoming references FROM resources/data/modules (not outputs/vars)
	const resourceIncoming = new Map<string, Set<string>>();
	for (const edge of edges) {
		const sourceNode = nodeMap.get(edge.source);
		if (!sourceNode || !groupableKinds.has(sourceNode.kind)) continue;
		if (!resourceIncoming.has(edge.target)) resourceIncoming.set(edge.target, new Set());
		resourceIncoming.get(edge.target)!.add(edge.source);
	}

	const groups: NodeGroup[] = [];
	const groupedIds = new Set<string>();

	for (const node of nodes) {
		if (!groupableKinds.has(node.kind)) continue;
		if (groupedIds.has(node.id)) continue;

		// Check: is this node referenced by exactly ONE resource/data/module?
		const resourceRefs = resourceIncoming.get(node.id);
		if (!resourceRefs || resourceRefs.size !== 1) continue;

		const parentId = [...resourceRefs][0];
		const parent = nodeMap.get(parentId);
		if (!parent || !groupableKinds.has(parent.kind)) continue;
		if (groupedIds.has(parentId)) continue;

		// Don't nest the parent inside itself
		if (parentId === node.id) continue;

		// Find or create group
		let group = groups.find((g) => g.primary.id === parentId);
		if (!group) {
			group = {
				id: `group-${parentId}`,
				label: parent.name,
				primary: parent,
				members: [parent]
			};
			groups.push(group);
		}

		// Limit group size to keep nodes readable
		if (group.members.length >= 4) continue;

		group.members.push(node);
		groupedIds.add(node.id);
	}

	// Only keep groups with actual children
	const validGroups = groups.filter((g) => g.members.length >= 2);

	groupedIds.clear();
	for (const g of validGroups) {
		for (const m of g.members) {
			if (m.id !== g.primary.id) {
				groupedIds.add(m.id);
			}
		}
	}

	return { groups: validGroups, groupedIds };
}

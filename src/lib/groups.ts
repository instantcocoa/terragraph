import type { GraphNode, GraphEdge } from './types';

export interface NodeGroup {
	id: string;
	label: string;
	primary: GraphNode; // The main resource in the group
	members: GraphNode[]; // All nodes in the group (including primary)
}

/**
 * Detect tightly-coupled resource groups.
 *
 * A resource is grouped with its "parent" if:
 * - It's only referenced by ONE other resource (exclusive dependency)
 * - It's in a closely related tier (resource↔resource, resource↔data)
 * - It's not a variable, output, local, or provider (those stay standalone)
 *
 * Returns groups and the set of node IDs that are grouped (non-primary).
 */
export function detectGroups(
	nodes: GraphNode[],
	edges: GraphEdge[]
): { groups: NodeGroup[]; groupedIds: Set<string> } {
	const groupableKinds = new Set(['resource', 'data', 'module']);

	// Count how many resources reference each node (incoming reference count)
	const incomingFrom = new Map<string, Set<string>>();
	for (const edge of edges) {
		if (!incomingFrom.has(edge.target)) incomingFrom.set(edge.target, new Set());
		incomingFrom.get(edge.target)!.add(edge.source);
	}

	// Count outgoing references
	const outgoingTo = new Map<string, Set<string>>();
	for (const edge of edges) {
		if (!outgoingTo.has(edge.source)) outgoingTo.set(edge.source, new Set());
		outgoingTo.get(edge.source)!.add(edge.target);
	}

	const nodeMap = new Map(nodes.map((n) => [n.id, n]));
	const groups: NodeGroup[] = [];
	const groupedIds = new Set<string>();

	// Find resources that are exclusively used by one other resource
	for (const node of nodes) {
		if (!groupableKinds.has(node.kind)) continue;
		if (groupedIds.has(node.id)) continue;

		const refs = incomingFrom.get(node.id);
		if (!refs || refs.size !== 1) continue;

		const parentId = [...refs][0];
		const parent = nodeMap.get(parentId);
		if (!parent || !groupableKinds.has(parent.kind)) continue;
		if (groupedIds.has(parentId)) continue;

		// Don't group if the child also has many outgoing refs (it's a hub)
		const childOutgoing = outgoingTo.get(node.id);
		if (childOutgoing && childOutgoing.size > 2) continue;

		// This node is exclusively owned by parent - group it
		// Find or create the parent's group
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
		group.members.push(node);
		groupedIds.add(node.id);
	}

	// Only keep groups with 2+ members (primary + at least one child)
	const validGroups = groups.filter((g) => g.members.length >= 2);

	// Remove ungrouped primary IDs from groupedIds
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

<script lang="ts">
	import {
		SvelteFlow,
		Controls,
		Background,
		BackgroundVariant,
		MiniMap,
		type Node,
		type Edge
	} from '@xyflow/svelte';
	import '@xyflow/svelte/dist/style.css';
	import TerraNode from './TerraNode.svelte';
	import FitToNode from './FitToNode.svelte';
	import { workspace } from '$lib/stores/workspace.svelte';
	import { theme } from '$lib/stores/theme.svelte';
	import { layoutGraph, layoutGraphAsync } from '$lib/layout';
	import { detectGroups } from '$lib/groups';
	import type { PlanAction } from '$lib/types';

	const nodeTypes = { terra: TerraNode };

	// Detect compound groups
	const groupInfo = $derived(detectGroups(workspace.nodes, workspace.edges));

	// Start with sync layout, then upgrade to ELK async
	let positions = $state(new Map<string, { x: number; y: number }>());

	$effect(() => {
		const nodes = workspace.nodes;
		const edges = workspace.edges;
		if (nodes.length === 0) { positions = new Map(); return; }
		positions = layoutGraph(nodes, edges);

		layoutGraphAsync(nodes, edges).then((p) => {
			if (p.size > 0) positions = p;
		});
	});

	const connectableIds = $derived(
		new Set(workspace.connectableNodes.map((n) => n.id))
	);

	// Build flow nodes: group nodes become parent containers, members become children
	let flowNodes = $derived.by(() => {
		const result: Node[] = [];
		const childOf = new Map<string, string>(); // nodeId -> groupId

		// Map grouped children to their group
		for (const group of groupInfo.groups) {
			for (const member of group.members) {
				if (member.id !== group.primary.id) {
					childOf.set(member.id, group.id);
				}
			}

			// Add group container node
			const pos = positions.get(group.primary.id) ?? { x: 0, y: 0 };
			const memberCount = group.members.length;
			result.push({
				id: group.id,
				type: 'group',
				position: { x: pos.x - 15, y: pos.y - 15 },
				style: `width: ${220 + 10}px; height: ${72 * memberCount + 20 * (memberCount - 1) + 30}px; background: rgba(255,255,255,0.02); border: 1px solid var(--border, #2f3146); border-radius: 10px;`,
				data: {}
			});
		}

		for (const node of workspace.nodes) {
			const planChange = workspace.planChangeMap.get(node.address);
			const isGroupChild = childOf.has(node.id);
			const isGroupPrimary = groupInfo.groups.some((g) => g.primary.id === node.id);

			let position: { x: number; y: number };

			if (isGroupChild) {
				// Position relative to parent group
				const groupId = childOf.get(node.id)!;
				const group = groupInfo.groups.find((g) => g.id === groupId)!;
				const childIndex = group.members.findIndex((m) => m.id === node.id);
				// Stack children below the primary
				position = { x: 15, y: 15 + childIndex * (72 + 20) };
			} else if (isGroupPrimary) {
				// Primary is first child of the group container
				position = { x: 15, y: 15 };
			} else {
				const pos = positions.get(node.id) ?? { x: 0, y: 0 };
				position = pos;
			}

			const flowNode: Node = {
				id: node.id,
				type: 'terra',
				position,
				data: {
					label: node.name,
					kind: node.kind,
					resourceType: node.resourceType,
					provider: node.provider,
					planAction: (workspace.showPlanOverlay ? planChange?.action : undefined) as PlanAction | undefined,
					selected: workspace.selectedNodeId === node.id,
					connectable: connectableIds.has(node.id),
					dimmed: workspace.connectingMode && !connectableIds.has(node.id) && workspace.selectedNodeId !== node.id,
					diagnosticCount: workspace.diagnostics.filter(
						(d) => d.range?.file === node.source.file &&
							d.range.startLine >= node.source.startLine &&
							d.range.endLine <= node.source.endLine
					).length
				}
			};

			// Set parent for grouped nodes
			if (isGroupChild || isGroupPrimary) {
				const groupId = isGroupChild
					? childOf.get(node.id)!
					: groupInfo.groups.find((g) => g.primary.id === node.id)!.id;
				flowNode.parentId = groupId;
				flowNode.expandParent = true;
			}

			result.push(flowNode);
		}

		return result;
	});

	// Tier map for edge direction correction
	const KIND_TIER: Record<string, number> = {
		resource: 0, module: 0, data: 1, local: 2, variable: 3, output: 4
	};
	const nodeKindMap = $derived(new Map(workspace.nodes.map((n) => [n.id, n.kind])));

	const EDGE_COLORS: Record<string, string> = {
		resource: '#3b82f680',
		data: '#8b5cf680',
		module: '#f59e0b80',
		variable: '#10b98180',
		local: '#06b6d480',
		output: '#f43f5e80',
		provider: '#6366f180'
	};

	let flowEdges = $derived<Edge[]>(
		workspace.edges.map((edge) => {
			const sourceKind = nodeKindMap.get(edge.source) ?? '';
			const targetKind = nodeKindMap.get(edge.target) ?? '';
			const sourceTier = KIND_TIER[sourceKind] ?? 2;
			const targetTier = KIND_TIER[targetKind] ?? 2;

			// Always make edges flow downward visually (from higher tier to lower)
			const goesUp = sourceTier > targetTier;
			const visualSource = goesUp ? edge.target : edge.source;
			const visualTarget = goesUp ? edge.source : edge.target;

			// Color by the upstream (top) node
			const upstreamKind = goesUp ? targetKind : sourceKind;
			const color = EDGE_COLORS[upstreamKind] ?? '#52525b60';

			if (edge.kind === 'depends_on') {
				return {
					id: edge.id,
					source: visualSource,
					target: visualTarget,
					type: 'smoothstep',
					animated: false,
					style: `stroke: #f59e0b; stroke-width: 1.5; stroke-dasharray: 6 4;`
				};
			}

			return {
				id: edge.id,
				source: visualSource,
				target: visualTarget,
				type: 'smoothstep',
				animated: false,
				style: `stroke: ${color}; stroke-width: 1.5;`
			};
		})
	);
</script>

<div class="graph-container">
	{#if workspace.nodes.length > 0}
		<SvelteFlow
			nodes={flowNodes}
			edges={flowEdges}
			{nodeTypes}
			fitView
			colorMode={theme.current === 'light' ? 'light' : 'dark'}
			onnodeclick={({ node }) => workspace.selectNode(node.id)}
			onpaneclick={() => workspace.selectNode(null)}
			proOptions={{ hideAttribution: true }}
		>
			<Controls />
			<Background variant={BackgroundVariant.Dots} gap={20} size={1} />
			<MiniMap />
			<FitToNode />
		</SvelteFlow>
	{:else if workspace.loading}
		<div class="empty-state">
			<div class="spinner"></div>
			<p>Loading workspace...</p>
		</div>
	{:else if workspace.error}
		<div class="empty-state error">
			<p class="error-icon">!</p>
			<p>{workspace.error}</p>
		</div>
	{:else}
		<div class="empty-state">
			<p class="empty-icon">&#9633;</p>
			<p>No workspace loaded</p>
			<p class="hint">Enter a Terraform workspace path above to get started</p>
		</div>
	{/if}
</div>

<style>
	.graph-container {
		width: 100%;
		height: 100%;
		position: relative;
	}

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		color: var(--text-muted, #7982a9);
		gap: 8px;
	}

	.empty-state.error {
		color: #ef4444;
	}

	.empty-icon {
		font-size: 48px;
		opacity: 0.3;
	}

	.error-icon {
		font-size: 36px;
		width: 48px;
		height: 48px;
		border-radius: 50%;
		border: 2px solid #ef4444;
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: bold;
	}

	.hint {
		font-size: 13px;
		opacity: 0.5;
	}

	.spinner {
		width: 24px;
		height: 24px;
		border: 2px solid var(--border, #2f3146);
		border-top-color: var(--accent, #7aa2f7);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>

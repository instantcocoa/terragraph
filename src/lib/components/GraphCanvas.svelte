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

	// Map: primary node ID -> its grouped children
	const childrenByPrimary = $derived.by(() => {
		const m = new Map<string, Array<{ id: string; label: string; kind: import('$lib/types').NodeKind; resourceType?: string }>>();
		for (const group of groupInfo.groups) {
			const children = group.members
				.filter((member) => member.id !== group.primary.id)
				.map((member) => ({
					id: member.id,
					label: member.name,
					kind: member.kind,
					resourceType: member.resourceType
				}));
			m.set(group.primary.id, children);
		}
		return m;
	});

	// Layout only non-grouped nodes (grouped children are rendered inside their parent)
	const layoutNodes = $derived(
		workspace.nodes.filter((n) => !groupInfo.groupedIds.has(n.id))
	);

	let positions = $state(new Map<string, { x: number; y: number }>());

	$effect(() => {
		const nodes = layoutNodes;
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

	// Only render non-grouped nodes (grouped children are embedded in their parent's TerraNode)
	let flowNodes = $derived.by(() => {
		return layoutNodes.map((node) => {
			const pos = positions.get(node.id) ?? { x: 0, y: 0 };
			const planChange = workspace.planChangeMap.get(node.address);
			const children = childrenByPrimary.get(node.id);

			const flowNode: Node = {
				id: node.id,
				type: 'terra',
				position: { x: pos.x, y: pos.y },
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
					).length,
					children: children,
					onChildClick: (id: string) => workspace.selectNode(id)
				}
			};

			return flowNode;
		});
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
		workspace.edges
		.filter((edge) => !groupInfo.groupedIds.has(edge.source) && !groupInfo.groupedIds.has(edge.target))
		.map((edge) => {
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

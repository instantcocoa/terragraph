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
	import { layoutGraph } from '$lib/layout';
	import type { PlanAction } from '$lib/types';

	const nodeTypes = { terra: TerraNode };

	const positions = $derived(layoutGraph(workspace.nodes, workspace.edges));

	const connectableIds = $derived(
		new Set(workspace.connectableNodes.map((n) => n.id))
	);

	let flowNodes = $derived<Node[]>(
		workspace.nodes.map((node) => {
			const pos = positions.get(node.id) ?? { x: 0, y: 0 };
			const planChange = workspace.planChangeMap.get(node.address);

			return {
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
					).length
				}
			};
		})
	);

	// Color edges by the source node's kind
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
			const color = EDGE_COLORS[sourceKind] ?? '#52525b60';

			if (edge.kind === 'depends_on') {
				return {
					id: edge.id,
					source: edge.source,
					target: edge.target,
					type: 'smoothstep',
					animated: false,
					style: `stroke: #f59e0b; stroke-width: 1.5; stroke-dasharray: 6 4;`
				};
			}

			return {
				id: edge.id,
				source: edge.source,
				target: edge.target,
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

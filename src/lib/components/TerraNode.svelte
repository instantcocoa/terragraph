<script lang="ts">
	import { Handle, Position } from '@xyflow/svelte';
	import { NODE_KIND_CONFIG, PLAN_ACTION_CONFIG } from '$lib/types';
	import type { NodeKind, PlanAction } from '$lib/types';

	let { data }: { data: { label: string; kind: NodeKind; resourceType?: string; provider?: string; planAction?: PlanAction; selected?: boolean; connectable?: boolean; dimmed?: boolean; diagnosticCount?: number } } = $props();

	const config = $derived(NODE_KIND_CONFIG[data.kind]);
	const planConfig = $derived(data.planAction ? PLAN_ACTION_CONFIG[data.planAction] : null);

	// Convert hex color to rgba for theme-safe background
	function hexToRgba(hex: string, alpha: number): string {
		const r = parseInt(hex.slice(1, 3), 16);
		const g = parseInt(hex.slice(3, 5), 16);
		const b = parseInt(hex.slice(5, 7), 16);
		return `rgba(${r}, ${g}, ${b}, ${alpha})`;
	}

	const bgColor = $derived(hexToRgba(config.color, 0.15));
</script>

<div
	class="terra-node"
	class:selected={data.selected}
	class:connectable={data.connectable}
	class:dimmed={data.dimmed}
	style:border-color={planConfig ? planConfig.color : config.color}
	style:background-color={bgColor}
>
	<Handle type="target" position={Position.Top} />

	<div class="node-header">
		<span class="node-icon" style:color={config.color}>{config.icon}</span>
		<span class="node-kind">{config.label}</span>
		{#if planConfig}
			<span class="plan-badge" style:background-color={planConfig.color}>
				{planConfig.icon} {planConfig.label}
			</span>
		{/if}
		{#if data.diagnosticCount && data.diagnosticCount > 0}
			<span class="diag-badge">{data.diagnosticCount}</span>
		{/if}
	</div>

	<div class="node-label">{data.label}</div>

	{#if data.resourceType}
		<div class="node-type">{data.resourceType}</div>
	{/if}

	<Handle type="source" position={Position.Bottom} />
</div>

<style>
	.terra-node {
		padding: 8px 12px;
		border: 2px solid #3b82f6;
		border-radius: 8px;
		min-width: 180px;
		max-width: 280px;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		color: var(--text, #c0caf5);
		transition: all 0.15s ease;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
	}

	.terra-node.selected {
		box-shadow:
			0 0 0 2px rgba(59, 130, 246, 0.5),
			0 4px 16px rgba(0, 0, 0, 0.4);
	}

	.terra-node.connectable {
		box-shadow:
			0 0 0 2px rgba(34, 197, 94, 0.6),
			0 0 12px rgba(34, 197, 94, 0.3);
		border-color: #22c55e !important;
	}

	.terra-node.dimmed {
		opacity: 0.3;
	}

	.node-header {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 14px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		opacity: 0.7;
		margin-bottom: 4px;
	}

	.node-icon {
		font-size: 14px;
	}

	.node-kind {
		flex: 1;
	}

	.plan-badge {
		font-size: 11px;
		padding: 1px 5px;
		border-radius: 4px;
		color: #000;
		font-weight: 600;
	}

	.diag-badge {
		font-size: 11px;
		padding: 1px 5px;
		border-radius: 4px;
		background: #ef4444;
		color: #fff;
		font-weight: 600;
	}

	.node-label {
		font-size: 15px;
		font-weight: 600;
		word-break: break-word;
	}

	.node-type {
		font-size: 14px;
		opacity: 0.5;
		margin-top: 2px;
	}
</style>

<script lang="ts">
	import { workspace } from '$lib/stores/workspace.svelte';
	import { NODE_KIND_CONFIG } from '$lib/types';
	import type { NodeKind, GraphNode } from '$lib/types';

	let search = $state('');
	let collapsedGroups = $state<Set<string>>(new Set());

	// Hierarchy order: top-to-bottom dependency flow
	const KIND_ORDER: NodeKind[] = [
		'terraform',
		'provider',
		'variable',
		'local',
		'data',
		'resource',
		'module',
		'output'
	];

	const filteredNodes = $derived(
		workspace.nodes.filter((node) => {
			if (!search) return true;
			const s = search.toLowerCase();
			return (
				node.name.toLowerCase().includes(s) ||
				node.address.toLowerCase().includes(s) ||
				(node.resourceType?.toLowerCase().includes(s) ?? false)
			);
		})
	);

	// Group filtered nodes by kind, maintaining hierarchy order
	const groupedNodes = $derived.by(() => {
		const groups: Array<{ kind: NodeKind; label: string; color: string; icon: string; nodes: GraphNode[] }> = [];
		for (const kind of KIND_ORDER) {
			const nodesOfKind = filteredNodes.filter((n) => n.kind === kind);
			if (nodesOfKind.length === 0) continue;
			const config = NODE_KIND_CONFIG[kind];
			groups.push({
				kind,
				label: config.label,
				color: config.color,
				icon: config.icon,
				nodes: nodesOfKind
			});
		}
		return groups;
	});

	function toggleGroup(kind: string) {
		const next = new Set(collapsedGroups);
		if (next.has(kind)) {
			next.delete(kind);
		} else {
			next.add(kind);
		}
		collapsedGroups = next;
	}

	// Get edges FROM this node (what it references)
	function getOutgoingCount(node: GraphNode): number {
		return workspace.edges.filter((e) => e.source === node.id).length;
	}

	// Get edges TO this node (what references it)
	function getIncomingCount(node: GraphNode): number {
		return workspace.edges.filter((e) => e.target === node.id).length;
	}
</script>

<div class="sidebar">
	<div class="sidebar-header">
		<h3>Explorer</h3>
		{#if workspace.nodes.length > 0}
			<span class="node-count">{workspace.nodes.length}</span>
		{/if}
	</div>

	{#if workspace.path}
		<div class="search-box">
			<input
				type="text"
				bind:value={search}
				placeholder="Search nodes..."
				class="search-input"
			/>
		</div>

		<!-- Hierarchical tree -->
		<div class="tree">
			{#each groupedNodes as group}
				{@const collapsed = collapsedGroups.has(group.kind)}
				<div class="tree-group">
					<button
						class="group-header"
						onclick={() => toggleGroup(group.kind)}
						style:--group-color={group.color}
					>
						<span class="chevron">{collapsed ? '▸' : '▾'}</span>
						<span class="group-icon" style:color={group.color}>{group.icon}</span>
						<span class="group-label">{group.label}s</span>
						<span class="group-count">{group.nodes.length}</span>
					</button>

					{#if !collapsed}
						<div class="group-children">
							{#each group.nodes as node}
								{@const isSelected = workspace.selectedNodeId === node.id}
								{@const outgoing = getOutgoingCount(node)}
								{@const incoming = getIncomingCount(node)}
								<button
									class="tree-node"
									class:selected={isSelected}
									onclick={() => workspace.selectNode(node.id)}
								>
									<span class="tree-indent"></span>
									<div class="tree-node-info">
										<div class="tree-node-name">
											{node.name}
											{#if node.resourceType}
												<span class="tree-node-type">{node.resourceType}</span>
											{/if}
										</div>
										<div class="tree-node-meta">
											{#if incoming > 0}
												<span class="edge-badge in" title="{incoming} incoming references">{incoming} in</span>
											{/if}
											{#if outgoing > 0}
												<span class="edge-badge out" title="{outgoing} outgoing references">{outgoing} out</span>
											{/if}
											{#if node.source}
												<span class="tree-node-file">{node.source.file}:{node.source.startLine}</span>
											{/if}
										</div>
									</div>
								</button>
							{/each}
						</div>
					{/if}
				</div>
			{/each}
		</div>

		<!-- Files -->
		{#if workspace.files.length > 0}
			<div class="files-section">
				<h4 class="section-label">Files ({workspace.files.length})</h4>
				{#each workspace.files as file}
					<div class="file-item">{file}</div>
				{/each}
			</div>
		{/if}
	{:else}
		<div class="empty">No workspace loaded</div>
	{/if}
</div>

<style>
	.sidebar {
		height: 100%;
		overflow-y: auto;
		font-size: 14px;
	}

	.sidebar-header {
		padding: 10px 12px;
		border-bottom: 1px solid var(--border);
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.sidebar-header h3 {
		font-size: 14px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
	}

	.node-count {
		font-size: 12px;
		padding: 1px 6px;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.06);
		color: var(--text-subtle);
	}

	.search-box {
		padding: 8px;
	}

	.search-input {
		width: 100%;
		padding: 6px 8px;
		background: var(--bg-base);
		border: 1px solid var(--border);
		border-radius: 4px;
		color: var(--text);
		font-size: 14px;
		outline: none;
	}

	.search-input:focus {
		border-color: var(--accent);
	}

	.search-input::placeholder {
		color: #3f3f46;
	}

	.tree {
		padding-bottom: 8px;
	}

	.tree-group {
		border-bottom: 1px solid rgba(39, 39, 42, 0.5);
	}

	.group-header {
		display: flex;
		align-items: center;
		gap: 4px;
		width: 100%;
		padding: 6px 8px;
		border: none;
		background: rgba(255, 255, 255, 0.02);
		color: #a1a1aa;
		cursor: pointer;
		text-align: left;
		font-size: 14px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.03em;
		transition: background 0.1s;
	}

	.group-header:hover {
		background: rgba(255, 255, 255, 0.04);
	}

	.chevron {
		font-size: 12px;
		width: 12px;
		color: var(--text-subtle);
	}

	.group-icon {
		font-size: 14px;
	}

	.group-label {
		flex: 1;
	}

	.group-count {
		font-size: 12px;
		padding: 0 5px;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.05);
		color: var(--text-subtle);
		font-weight: 400;
	}

	.group-children {
		padding: 2px 0;
	}

	.tree-node {
		display: flex;
		align-items: flex-start;
		gap: 0;
		width: 100%;
		padding: 4px 8px 4px 12px;
		border: none;
		background: none;
		color: var(--text);
		cursor: pointer;
		text-align: left;
		border-radius: 0;
		transition: background 0.1s;
	}

	.tree-node:hover {
		background: rgba(255, 255, 255, 0.03);
	}

	.tree-node.selected {
		background: rgba(59, 130, 246, 0.1);
		border-left: 2px solid #3b82f6;
		padding-left: 10px;
	}

	.tree-indent {
		width: 16px;
		flex-shrink: 0;
		border-left: 1px solid var(--border);
		height: 100%;
		min-height: 16px;
		margin-right: 6px;
	}

	.tree-node-info {
		min-width: 0;
		flex: 1;
	}

	.tree-node-name {
		font-size: 14px;
		font-weight: 500;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.tree-node-type {
		font-size: 12px;
		color: var(--text-subtle);
		font-weight: 400;
	}

	.tree-node-meta {
		display: flex;
		align-items: center;
		gap: 4px;
		margin-top: 1px;
	}

	.edge-badge {
		font-size: 12px;
		padding: 0 4px;
		border-radius: 3px;
		font-weight: 500;
	}

	.edge-badge.in {
		background: rgba(59, 130, 246, 0.1);
		color: #60a5fa;
	}

	.edge-badge.out {
		background: rgba(245, 158, 11, 0.1);
		color: #fbbf24;
	}

	.tree-node-file {
		font-size: 12px;
		color: #3f3f46;
		font-family: monospace;
	}

	.files-section {
		border-top: 1px solid var(--border);
		padding: 4px 0 8px;
	}

	.section-label {
		font-size: 12px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-subtle);
		padding: 8px 12px 4px;
	}

	.file-item {
		padding: 3px 12px;
		font-family: monospace;
		font-size: 13px;
		color: var(--text-subtle);
	}

	.empty {
		padding: 24px;
		text-align: center;
		color: var(--text-subtle);
	}
</style>

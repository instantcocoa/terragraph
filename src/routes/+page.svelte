<script lang="ts">
	import Toolbar from '$lib/components/Toolbar.svelte';
	import GraphCanvas from '$lib/components/GraphCanvas.svelte';
	import Inspector from '$lib/components/Inspector.svelte';
	import BottomPanel from '$lib/components/BottomPanel.svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import AddBlockDialog from '$lib/components/AddBlockDialog.svelte';
	import HCLEditor from '$lib/components/HCLEditor.svelte';
	import { workspace } from '$lib/stores/workspace.svelte';
	import { theme } from '$lib/stores/theme.svelte';

	let bottomCollapsed = $state(false);
	let sidebarVisible = $state(true);

	// Resizable panel sizes
	let sidebarWidth = $state(260);
	let inspectorWidth = $state(380);
	let bottomHeight = $state(220);

	// Drag state
	let dragging = $state<'sidebar' | 'inspector' | 'bottom' | null>(null);

	function startDrag(panel: 'sidebar' | 'inspector' | 'bottom') {
		dragging = panel;
	}

	function handleMouseMove(e: MouseEvent) {
		if (!dragging) return;
		e.preventDefault();

		if (dragging === 'sidebar') {
			sidebarWidth = Math.max(180, Math.min(400, e.clientX));
		} else if (dragging === 'inspector') {
			inspectorWidth = Math.max(280, Math.min(600, window.innerWidth - e.clientX));
		} else if (dragging === 'bottom') {
			bottomHeight = Math.max(80, Math.min(500, window.innerHeight - e.clientY));
		}
	}

	function stopDrag() {
		dragging = null;
	}
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape') workspace.selectNode(null);
		if (e.key === 'b' && (e.metaKey || e.ctrlKey)) {
			e.preventDefault();
			sidebarVisible = !sidebarVisible;
		}
		if (e.key === 'z' && (e.metaKey || e.ctrlKey) && !e.shiftKey) {
			e.preventDefault();
			workspace.undo();
		}
		if (e.key === 'z' && (e.metaKey || e.ctrlKey) && e.shiftKey) {
			e.preventDefault();
			workspace.redo();
		}
	}}
	onmousemove={handleMouseMove}
	onmouseup={stopDrag}
/>

<div class="ide-layout theme-{theme.current}" class:dragging={!!dragging}>
	<Toolbar />

	<div class="main-area">
		{#if sidebarVisible}
			<div class="sidebar-panel" style="width:{sidebarWidth}px; max-width:30vw;">
				<Sidebar />
			</div>
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="resize-handle vertical" onmousedown={() => startDrag('sidebar')}></div>
		{/if}

		<div class="graph-area">
			<GraphCanvas />
		</div>

		{#if workspace.selectedNode}
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="resize-handle vertical" onmousedown={() => startDrag('inspector')}></div>
			<div class="inspector-panel" style="width:{inspectorWidth}px; max-width:40vw;">
				<div class="panel-header">
					<span>Inspector</span>
					<button class="close-btn" onclick={() => workspace.selectNode(null)}>x</button>
				</div>
				<Inspector />
			</div>
		{/if}
	</div>

	{#if !bottomCollapsed || workspace.validating || workspace.planning}
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="resize-handle horizontal" onmousedown={() => startDrag('bottom')}></div>
	{/if}

	<div
		class="bottom-area"
		class:collapsed={bottomCollapsed && !workspace.validating && !workspace.planning}
		style="height:{bottomCollapsed && !workspace.validating && !workspace.planning ? 'auto' : `${bottomHeight}px`};"
	>
		<div class="bottom-toolbar">
			<button class="bottom-toggle" onclick={() => (bottomCollapsed = !bottomCollapsed)}>
				{bottomCollapsed ? '▴ Show Panel' : '▾ Hide Panel'}
			</button>
			{#if workspace.diagnostics.length > 0}
				<span class="bottom-badge error">{workspace.diagnostics.length} issues</span>
			{/if}
			{#if workspace.planSummary}
				<span class="bottom-badge info">Plan: {workspace.planSummary.create + workspace.planSummary.update + workspace.planSummary.delete + workspace.planSummary.replace} changes</span>
			{/if}
		</div>
		{#if !bottomCollapsed || workspace.validating || workspace.planning}
			<BottomPanel collapsed={false} />
		{/if}
	</div>

	{#if workspace.showAddDialog}
		<AddBlockDialog onclose={() => (workspace.showAddDialog = false)} />
	{/if}

	{#if workspace.showHCLEditor && workspace.selectedNode}
		<HCLEditor onclose={() => (workspace.showHCLEditor = false)} />
	{/if}
</div>

<style>
	.ide-layout {
		display: flex;
		flex-direction: column;
		height: 100vh;
		width: 100vw;
		max-height: 100vh;
		max-width: 100vw;
		overflow: hidden;
		background: var(--bg-base);
		color: var(--text);
	}

	.ide-layout.dragging {
		cursor: col-resize;
		user-select: none;
	}

	.main-area {
		flex: 1;
		display: flex;
		overflow: hidden;
		min-height: 0;
		max-width: 100%;
	}

	.sidebar-panel {
		background: var(--bg-panel);
		flex-shrink: 0;
		overflow-x: hidden;
		overflow-y: auto;
	}

	.graph-area {
		flex: 1;
		min-width: 200px;
		overflow: hidden;
	}

	.inspector-panel {
		background: var(--bg-panel);
		display: flex;
		flex-direction: column;
		flex-shrink: 0;
		overflow-x: hidden;
		overflow-y: auto;
	}

	.panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
		font-size: 13px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		border-bottom: 1px solid var(--border);
		flex-shrink: 0;
	}

	.close-btn {
		width: 20px;
		height: 20px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		border-radius: 3px;
		font-size: 14px;
	}

	.close-btn:hover {
		background: var(--border);
		color: var(--text);
	}

	.resize-handle {
		flex-shrink: 0;
		background: var(--border);
		transition: background 0.15s;
		z-index: 5;
	}

	.resize-handle:hover,
	.ide-layout.dragging .resize-handle {
		background: var(--accent, #7aa2f7);
	}

	.resize-handle.vertical {
		width: 3px;
		cursor: col-resize;
	}

	.resize-handle.horizontal {
		height: 3px;
		cursor: row-resize;
	}

	.bottom-area {
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		min-height: 36px;
		max-height: 50vh;
	}

	.bottom-area.collapsed {
		height: auto !important;
	}

	.bottom-toolbar {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 0 8px;
		height: 28px;
		flex-shrink: 0;
		border-top: 1px solid var(--border);
		background: var(--bg-panel);
	}

	.bottom-toggle {
		font-size: 12px;
		color: var(--text-muted);
		background: none;
		border: none;
		cursor: pointer;
		padding: 2px 6px;
		border-radius: 3px;
	}

	.bottom-toggle:hover {
		color: var(--text);
		background: var(--bg-hover);
	}

	.bottom-badge {
		font-size: 12px;
		padding: 1px 8px;
		border-radius: 8px;
	}

	.bottom-badge.error {
		background: rgba(239, 68, 68, 0.15);
		color: #ef4444;
	}

	.bottom-badge.info {
		background: rgba(122, 162, 247, 0.15);
		color: var(--accent);
	}
</style>

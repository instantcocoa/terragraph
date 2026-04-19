<script lang="ts">
	import { workspace } from '$lib/stores/workspace.svelte';
	import { theme } from '$lib/stores/theme.svelte';

	let pathInput = $state('');
	let picking = $state(false);

	async function handleLoad() {
		if (pathInput.trim()) {
			workspace.load(pathInput.trim());
		} else {
			// Open native folder picker
			await pickFolder();
		}
	}

	async function pickFolder() {
		picking = true;
		try {
			const res = await fetch('/api/pick-folder', { method: 'POST' });
			const data = await res.json();
			if (data.path) {
				pathInput = data.path;
				workspace.load(data.path);
			}
		} catch {
			// User cancelled or picker unavailable
		} finally {
			picking = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			handleLoad();
		}
	}
</script>

<div class="toolbar">
	<div class="toolbar-left">
		<span class="logo">TerraGraph</span>
		<button class="btn theme-btn" onclick={() => theme.toggle()} title="Switch theme">
			{theme.current === 'dark' ? '🌙' : theme.current === 'light' ? '☀️' : '🌅'}
		</button>
	</div>

	<div class="toolbar-center">
		<div class="path-input-group">
			<input
				type="text"
				bind:value={pathInput}
				onkeydown={handleKeydown}
				placeholder="Enter path or click Open..."
				class="path-input"
			/>
			<button class="btn primary" onclick={handleLoad} disabled={workspace.loading || picking}>
				{workspace.loading ? 'Loading...' : picking ? 'Opening...' : pathInput.trim() ? 'Load' : 'Open'}
			</button>
		</div>
	</div>

	<div class="toolbar-right">
		{#if workspace.path}
			<button
				class="btn add"
				onclick={() => (workspace.showAddDialog = true)}
			>
				+ Add
			</button>
			<div class="separator"></div>
			<button
				class="btn undo-redo"
				onclick={() => workspace.undo()}
				disabled={workspace.undoCount === 0}
				title="Undo (Cmd+Z)"
			>
				&#x21A9; {#if workspace.undoCount > 0}<span class="history-count">{workspace.undoCount}</span>{/if}
			</button>
			<button
				class="btn undo-redo"
				onclick={() => workspace.redo()}
				disabled={workspace.redoCount === 0}
				title="Redo (Cmd+Shift+Z)"
			>
				&#x21AA; {#if workspace.redoCount > 0}<span class="history-count">{workspace.redoCount}</span>{/if}
			</button>
			<div class="separator"></div>
			<button
				class="btn"
				onclick={() => workspace.load(workspace.path)}
				disabled={workspace.loading}
			>
				Reload
			</button>
			<button
				class="btn validate-btn"
				class:pass={workspace.validateResult === 'pass'}
				class:fail={workspace.validateResult === 'fail'}
				onclick={() => workspace.validate()}
				disabled={workspace.validating}
			>
				{#if workspace.validating}
					Validating...
				{:else if workspace.validateResult === 'pass'}
					Valid &#10003;
				{:else if workspace.validateResult === 'fail'}
					{workspace.diagnostics.length} Issue{workspace.diagnostics.length !== 1 ? 's' : ''} &#10007;
				{:else}
					Validate
				{/if}
			</button>
			<button
				class="btn accent"
				onclick={() => workspace.plan()}
				disabled={workspace.planning}
			>
				{workspace.planning ? 'Planning...' : 'Plan'}
			</button>
			{#if workspace.showPlanOverlay}
				<button
					class="btn"
					onclick={() => (workspace.showPlanOverlay = !workspace.showPlanOverlay)}
				>
					{workspace.showPlanOverlay ? 'Hide Plan' : 'Show Plan'}
				</button>
			{/if}
		{/if}
	</div>
</div>

<style>
	.toolbar {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 16px;
		background: var(--bg-panel);
		border-bottom: 1px solid var(--border);
		flex-shrink: 0;
		overflow: hidden;
		max-width: 100%;
	}

	.toolbar-left {
		flex-shrink: 0;
	}

	.logo {
		font-size: 18px;
		font-weight: 700;
		color: var(--text);
		letter-spacing: -0.025em;
	}

	.toolbar-center {
		flex: 1;
		max-width: 600px;
	}

	.path-input-group {
		display: flex;
		gap: 4px;
	}

	.path-input {
		flex: 1;
		padding: 6px 12px;
		background: var(--bg-base);
		border: 1px solid var(--border);
		border-radius: 6px;
		color: var(--text);
		font-size: 14px;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		outline: none;
		transition: border-color 0.15s ease;
	}

	.path-input:focus {
		border-color: var(--accent);
	}

	.path-input::placeholder {
		color: #3f3f46;
	}

	.toolbar-right {
		display: flex;
		gap: 6px;
		flex-shrink: 1;
		flex-wrap: wrap;
		overflow: hidden;
	}

	.btn {
		padding: 6px 14px;
		font-size: 13px;
		font-weight: 500;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--border);
		color: var(--text);
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.btn:hover:not(:disabled) {
		background: #3f3f46;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn.primary {
		background: #3b82f6;
		border-color: #3b82f6;
		color: #fff;
	}

	.btn.primary:hover:not(:disabled) {
		background: #2563eb;
	}

	.btn.undo-redo {
		padding: 6px 10px;
		font-size: 16px;
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.history-count {
		font-size: 11px;
		padding: 0 4px;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.08);
		color: var(--text-muted);
	}

	.btn.add {
		background: #22c55e;
		border-color: #22c55e;
		color: #000;
		font-weight: 600;
	}

	.btn.add:hover:not(:disabled) {
		background: #16a34a;
	}

	.separator {
		width: 1px;
		height: 24px;
		background: var(--border);
		margin: 0 2px;
	}

	.theme-btn {
		padding: 6px 10px;
		font-size: 16px;
		margin-left: 8px;
	}

	.btn.accent {
		background: #7c3aed;
		border-color: #7c3aed;
		color: #fff;
	}

	.btn.accent:hover:not(:disabled) {
		background: #6d28d9;
	}

	.btn.validate-btn.pass {
		background: rgba(34, 197, 94, 0.15);
		border-color: #22c55e;
		color: #22c55e;
	}

	.btn.validate-btn.fail {
		background: rgba(239, 68, 68, 0.15);
		border-color: #ef4444;
		color: #ef4444;
	}
</style>

<script lang="ts">
	import { workspace } from '$lib/stores/workspace.svelte';

	let search = $state('');
	let selectedCategory = $state<'resource' | 'data'>('resource');

	const providers = $derived(workspace.availableProviders);

	const filteredTypes = $derived.by(() => {
		if (!workspace.schemas) return [];
		const types =
			selectedCategory === 'resource'
				? Object.entries(workspace.schemas.resources)
				: Object.entries(workspace.schemas.dataSources);

		if (!search) return types.slice(0, 80);
		const s = search.toLowerCase();
		return types.filter(([name]) => name.includes(s)).slice(0, 80);
	});

	// Group by provider prefix
	const groupedTypes = $derived.by(() => {
		const groups = new Map<string, Array<[string, { attributes: Array<{ name: string; required: boolean }> }]>>();
		for (const [name, schema] of filteredTypes) {
			const prefix = name.split('_')[0];
			if (!groups.has(prefix)) groups.set(prefix, []);
			groups.get(prefix)!.push([name, schema as any]);
		}
		return groups;
	});

	async function addFromPalette(resourceType: string) {
		const parts = resourceType.split('_');
		const name = parts.length > 1 ? parts.slice(1).join('_') : resourceType;
		const file = workspace.files[0] || 'main.tf';

		await workspace.addBlock({
			file,
			blockType: selectedCategory,
			resourceType,
			name
		}, true);
	}
</script>

<div class="palette">
	<div class="palette-header">
		<h3>Resources</h3>
		<div class="palette-tabs">
			<button class:active={selectedCategory === 'resource'} onclick={() => (selectedCategory = 'resource')}>
				Resources
			</button>
			<button class:active={selectedCategory === 'data'} onclick={() => (selectedCategory = 'data')}>
				Data
			</button>
		</div>
	</div>

	<div class="palette-search">
		<input
			type="text"
			bind:value={search}
			placeholder="Search {selectedCategory === 'resource' ? 'resources' : 'data sources'}..."
			class="search-input"
		/>
	</div>

	{#if !workspace.schemas}
		<div class="palette-empty">
			{#if workspace.schemaLoading}
				Loading provider schemas...
			{:else}
				Load a workspace to see available resources
			{/if}
		</div>
	{:else}
		<div class="palette-list">
			{#each [...groupedTypes.entries()] as [provider, types]}
				<div class="palette-group">
					<div class="palette-group-header">{provider} ({types.length})</div>
					{#each types as [typeName, schema]}
						{@const requiredCount = schema.attributes?.filter((a: any) => a.required).length ?? 0}
						<button class="palette-item" onclick={() => addFromPalette(typeName)}>
							<span class="palette-item-name">{typeName}</span>
							{#if requiredCount > 0}
								<span class="palette-item-req">{requiredCount} req</span>
							{/if}
						</button>
					{/each}
				</div>
			{/each}
			{#if filteredTypes.length === 0}
				<div class="palette-empty">No matching {selectedCategory === 'resource' ? 'resources' : 'data sources'}</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.palette {
		height: 100%;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.palette-header {
		padding: 10px 12px 0;
	}

	.palette-header h3 {
		font-size: 12px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		margin-bottom: 8px;
	}

	.palette-tabs {
		display: flex;
		gap: 2px;
		margin-bottom: 8px;
	}

	.palette-tabs button {
		flex: 1;
		padding: 5px 8px;
		font-size: 13px;
		border: 1px solid var(--border);
		background: none;
		color: var(--text-muted);
		cursor: pointer;
		border-radius: 4px;
	}

	.palette-tabs button.active {
		background: var(--bg-hover);
		color: var(--accent);
		border-color: var(--accent);
	}

	.palette-search {
		padding: 0 12px 8px;
	}

	.search-input {
		width: 100%;
		padding: 6px 8px;
		background: var(--bg-input, var(--bg-base));
		border: 1px solid var(--border);
		border-radius: 4px;
		color: var(--text);
		font-size: 13px;
		outline: none;
	}

	.search-input:focus {
		border-color: var(--accent);
	}

	.palette-list {
		flex: 1;
		overflow-y: auto;
		padding: 0 4px 8px;
	}

	.palette-group {
		margin-bottom: 4px;
	}

	.palette-group-header {
		font-size: 12px;
		font-weight: 600;
		color: var(--text-subtle);
		padding: 6px 8px 2px;
		text-transform: uppercase;
		letter-spacing: 0.03em;
	}

	.palette-item {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 5px 8px;
		border: none;
		background: none;
		color: var(--text);
		cursor: pointer;
		text-align: left;
		border-radius: 4px;
		font-size: 13px;
		gap: 6px;
		transition: background 0.1s;
	}

	.palette-item:hover {
		background: var(--bg-hover);
	}

	.palette-item-name {
		flex: 1;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 12px;
	}

	.palette-item-req {
		font-size: 11px;
		color: var(--text-subtle);
		padding: 0 4px;
		border-radius: 3px;
		background: rgba(239, 68, 68, 0.1);
		color: #f87171;
	}

	.palette-empty {
		padding: 24px 12px;
		text-align: center;
		color: var(--text-subtle);
		font-size: 13px;
	}
</style>

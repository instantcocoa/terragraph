<script lang="ts">
	import { workspace } from '$lib/stores/workspace.svelte';
	import * as api from '$lib/api';
	import type { ScaffoldDependency } from '$lib/api';

	let { onclose }: { onclose: () => void } = $props();

	let blockType = $state<'resource' | 'data' | 'variable' | 'output' | 'provider'>('resource');
	let selectedProvider = $state('');
	let resourceType = $state('');
	let name = $state('');
	let targetFile = $state('');
	let typeSearch = $state('');
	let showDropdown = $state(false);
	let newProviderName = $state('');
	let newProviderRegion = $state('');

	const KNOWN_PROVIDERS = [
		{ name: 'aws', label: 'AWS', source: 'hashicorp/aws', defaultRegion: 'us-east-1' },
		{ name: 'google', label: 'Google Cloud', source: 'hashicorp/google', defaultRegion: 'us-central1' },
		{ name: 'azurerm', label: 'Azure', source: 'hashicorp/azurerm', defaultRegion: 'eastus' },
		{ name: 'hcloud', label: 'Hetzner', source: 'hetznercloud/hcloud', defaultRegion: 'nbg1' },
		{ name: 'digitalocean', label: 'DigitalOcean', source: 'digitalocean/digitalocean', defaultRegion: 'nyc1' },
		{ name: 'cloudflare', label: 'Cloudflare', source: 'cloudflare/cloudflare', defaultRegion: '' },
		{ name: 'kubernetes', label: 'Kubernetes', source: 'hashicorp/kubernetes', defaultRegion: '' },
		{ name: 'docker', label: 'Docker', source: 'kreuzwerker/docker', defaultRegion: '' },
		{ name: 'github', label: 'GitHub', source: 'integrations/github', defaultRegion: '' },
		{ name: 'random', label: 'Random', source: 'hashicorp/random', defaultRegion: '' },
		{ name: 'null', label: 'Null', source: 'hashicorp/null', defaultRegion: '' },
		{ name: 'local', label: 'Local', source: 'hashicorp/local', defaultRegion: '' },
		{ name: 'tls', label: 'TLS', source: 'hashicorp/tls', defaultRegion: '' },
	];

	// Scaffold state
	let scaffoldDeps = $state<ScaffoldDependency[]>([]);
	let scaffoldAttrs = $state<Record<string, string>>({});
	let selectedDeps = $state<Set<string>>(new Set());
	let loadingScaffold = $state(false);

	const needsResourceType = $derived(blockType === 'resource' || blockType === 'data');
	const isProvider = $derived(blockType === 'provider');
	const files = $derived(workspace.files.length > 0 ? workspace.files : ['main.tf']);

	// Providers available from schema
	const providers = $derived(workspace.availableProviders);

	// Auto-select first provider if available
	$effect(() => {
		if (providers.length > 0 && !selectedProvider) {
			// Pick aws if available, otherwise first
			selectedProvider = providers.includes('aws') ? 'aws' : providers[0];
		}
	});

	// Resource/data types filtered by provider and search
	const filteredTypes = $derived.by(() => {
		if (!needsResourceType || !selectedProvider) return [];
		const allTypes =
			blockType === 'resource'
				? workspace.resourceTypesByProvider(selectedProvider)
				: workspace.dataSourceTypesByProvider(selectedProvider);

		if (!typeSearch) return allTypes.slice(0, 100);
		const search = typeSearch.toLowerCase();
		return allTypes.filter((t) => t.includes(search)).slice(0, 100);
	});

	// Total count for current provider
	const totalTypesForProvider = $derived.by(() => {
		if (!needsResourceType || !selectedProvider) return 0;
		return blockType === 'resource'
			? workspace.resourceTypesByProvider(selectedProvider).length
			: workspace.dataSourceTypesByProvider(selectedProvider).length;
	});

	$effect(() => {
		if (files.length > 0 && !targetFile) {
			targetFile = files[0];
		}
	});

	// Reset resource type when switching providers or block types
	$effect(() => {
		// Track these deps
		void blockType;
		void selectedProvider;
		resourceType = '';
		typeSearch = '';
	});

	async function selectType(t: string) {
		resourceType = t;
		typeSearch = t;
		showDropdown = false;
		// Auto-generate a name from the type
		if (!name) {
			const parts = t.split('_');
			name = parts.length > 1 ? parts.slice(1).join('_') : t;
		}
		// Fetch scaffold info
		loadingScaffold = true;
		try {
			const scaffold = await api.getScaffold(t);
			scaffoldAttrs = scaffold.attributes || {};
			scaffoldDeps = scaffold.dependencies || [];
			// Auto-select required dependencies
			selectedDeps = new Set(
				scaffoldDeps.filter((d) => d.required).map((d) => `${d.blockType}.${d.resourceType}.${d.name}`)
			);
		} catch {
			scaffoldDeps = [];
			scaffoldAttrs = {};
		} finally {
			loadingScaffold = false;
		}
	}

	function toggleDep(dep: ScaffoldDependency) {
		const key = `${dep.blockType}.${dep.resourceType}.${dep.name}`;
		const next = new Set(selectedDeps);
		if (next.has(key)) {
			next.delete(key);
		} else {
			next.add(key);
		}
		selectedDeps = next;
	}

	function isDepSelected(dep: ScaffoldDependency): boolean {
		return selectedDeps.has(`${dep.blockType}.${dep.resourceType}.${dep.name}`);
	}

	// Check if a dependency resource already exists in the workspace
	function depExists(dep: ScaffoldDependency): boolean {
		const addr = dep.blockType === 'data'
			? `data.${dep.resourceType}.${dep.name}`
			: `${dep.resourceType}.${dep.name}`;
		return workspace.nodes.some((n) => n.address === addr);
	}

	function defaultForSchemaType(type: unknown, name: string): string {
		// Generate a sensible default based on type and attribute name
		if (type === 'string') {
			// Try to generate a meaningful default from the attribute name
			if (name === 'name' || name.endsWith('_name')) return `"my-${name.replace(/_name$/, '')}"`;
			if (name === 'region') return `"us-east-1"`;
			if (name === 'type') return `"default"`;
			return `"TODO"`;
		}
		if (type === 'number') return '1';
		if (type === 'bool') return 'false';
		if (Array.isArray(type)) {
			if (type[0] === 'list' || type[0] === 'set') return '[]';
			if (type[0] === 'map') return '{}';
		}
		return `"TODO"`;
	}

	async function handleAdd() {
		// Handle provider addition
		if (isProvider) {
			if (!newProviderName) return;
			const providerInfo = KNOWN_PROVIDERS.find((p) => p.name === newProviderName);
			const attrs: Record<string, string> = {};
			if (newProviderRegion) {
				const regionAttr = newProviderName === 'hcloud' ? 'location' : 'region';
				attrs[regionAttr] = `"${newProviderRegion}"`;
			} else if (providerInfo?.defaultRegion) {
				const regionAttr = newProviderName === 'hcloud' ? 'location' : 'region';
				attrs[regionAttr] = `"${providerInfo.defaultRegion}"`;
			}
			await workspace.addProvider(newProviderName, attrs);
			onclose();
			return;
		}

		if (!name.trim()) return;
		if (needsResourceType && !resourceType.trim()) return;

		const file = targetFile || 'main.tf';

		// First create selected dependencies
		for (const dep of scaffoldDeps) {
			if (!isDepSelected(dep) || depExists(dep)) continue;
			await workspace.addBlock({
				file,
				blockType: dep.blockType as 'resource' | 'data',
				resourceType: dep.resourceType,
				name: dep.name,
				attributes: dep.attributes
			});
		}

		// Build attributes: scaffold defaults + required schema fields
		const attrs: Record<string, string> = { ...scaffoldAttrs };

		// If schema is loaded, populate required fields that aren't already set
		if (needsResourceType && workspace.schemas) {
			const schemaMap = blockType === 'resource'
				? workspace.schemas.resources
				: workspace.schemas.dataSources;
			const schema = schemaMap[resourceType];
			if (schema) {
				for (const attr of schema.attributes) {
					if (attr.required && !(attr.name in attrs)) {
						attrs[attr.name] = defaultForSchemaType(attr.type, attr.name);
					}
				}
			}
		}

		// Create the main resource and select it
		const bt = blockType as 'resource' | 'data' | 'variable' | 'output';
		await workspace.addBlock({
			file,
			blockType: bt,
			resourceType: needsResourceType ? resourceType : undefined,
			name: name.trim(),
			attributes: Object.keys(attrs).length > 0 ? attrs : undefined
		}, true);

		onclose();
	}
</script>

<div class="overlay" role="none" onmousedown={onclose}>
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<!-- svelte-ignore a11y_interactive_supports_focus -->
	<div class="dialog" onmousedown={(e) => e.stopPropagation()} role="dialog" aria-label="Add block">
		<div class="dialog-header">
			<h3>Add Block</h3>
			<button class="close-btn" onclick={onclose}>x</button>
		</div>

		<div class="dialog-body">
			<div class="form-field">
				<label for="block-type">Block Type</label>
				<div class="type-selector" id="block-type">
					<button class:active={blockType === 'resource'} onclick={() => (blockType = 'resource')}>
						Resource
					</button>
					<button class:active={blockType === 'data'} onclick={() => (blockType = 'data')}>
						Data Source
					</button>
					<button class:active={blockType === 'variable'} onclick={() => (blockType = 'variable')}>
						Variable
					</button>
					<button class:active={blockType === 'output'} onclick={() => (blockType = 'output')}>
						Output
					</button>
					<button class:active={blockType === 'provider'} onclick={() => (blockType = 'provider')}>
						Provider
					</button>
				</div>
			</div>

			{#if isProvider}
				<!-- Provider selection -->
				<div class="form-field">
					<label for="provider-pick">Select Provider</label>
					<div class="provider-grid" id="provider-pick">
						{#each KNOWN_PROVIDERS as p}
							<button
								class="provider-card"
								class:active={newProviderName === p.name}
								onclick={() => { newProviderName = p.name; newProviderRegion = p.defaultRegion; }}
							>
								<div class="provider-card-name">{p.label}</div>
								<div class="provider-card-source">{p.source}</div>
							</button>
						{/each}
					</div>
				</div>

				{#if newProviderName}
					{@const providerInfo = KNOWN_PROVIDERS.find((p) => p.name === newProviderName)}
					{#if providerInfo?.defaultRegion}
						<div class="form-field">
							<label for="provider-region">Region</label>
							<input
								id="provider-region"
								type="text"
								bind:value={newProviderRegion}
								placeholder={providerInfo.defaultRegion}
								class="input"
							/>
						</div>
					{/if}

					<div class="preview">
						<div class="preview-label">Will create</div>
						<pre class="preview-code">terraform {'{'}
  required_providers {'{'}
    {newProviderName} = {'{'}
      source  = "{providerInfo?.source}"
      version = "~> {newProviderName === 'aws' || newProviderName === 'google' ? '5.0' : '1.0'}"
    {'}'}
  {'}'}
{'}'}

provider "{newProviderName}" {'{'}
{#if newProviderRegion}  {newProviderName === 'hcloud' ? 'location' : 'region'} = "{newProviderRegion}"
{/if}{'}'}</pre>
					</div>
				{/if}

			{:else if needsResourceType}
				<!-- Provider selector -->
				<div class="form-field">
					<label for="provider-select">Provider</label>
					{#if providers.length > 0}
						<div class="provider-selector" id="provider-select">
							{#each providers as p}
								<button
									class="provider-chip"
									class:active={selectedProvider === p}
									onclick={() => (selectedProvider = p)}
								>
									{p}
									<span class="provider-count">
										{blockType === 'resource'
											? workspace.resourceTypesByProvider(p).length
											: workspace.dataSourceTypesByProvider(p).length}
									</span>
								</button>
							{/each}
						</div>
					{:else if !workspace.schemas}
						<div class="schema-notice">
							<p>Load schema first to see available providers and resource types.</p>
							<button
								class="load-schema-btn"
								onclick={() => workspace.loadSchema()}
								disabled={workspace.schemaLoading}
							>
								{workspace.schemaLoading ? 'Loading Schema...' : 'Load Schema'}
							</button>
						</div>
					{/if}
				</div>

				<!-- Resource type search + autocomplete -->
				<div class="form-field">
					<label for="resource-type-input">
						{blockType === 'resource' ? 'Resource' : 'Data Source'} Type
						{#if totalTypesForProvider > 0}
							<span class="count-hint">({totalTypesForProvider} available)</span>
						{/if}
					</label>
					{#if workspace.schemas && selectedProvider}
						<div class="autocomplete-wrapper">
							<input
								id="resource-type-input"
								type="text"
								bind:value={typeSearch}
								onfocus={() => (showDropdown = true)}
								oninput={() => {
									showDropdown = true;
									resourceType = typeSearch;
								}}
								placeholder={`Search ${selectedProvider}_ types...`}
								class="input"
								autocomplete="off"
							/>
							{#if resourceType && resourceType === typeSearch}
								<span class="check-mark">&#10003;</span>
							{/if}
						</div>
						{#if showDropdown && filteredTypes.length > 0}
							<div class="type-list">
								{#each filteredTypes as t}
									<button
										class="type-option"
										class:selected={resourceType === t}
										onclick={() => selectType(t)}
									>
										<span class="type-name">{t}</span>
									</button>
								{/each}
								{#if filteredTypes.length === 100}
									<div class="type-overflow">Showing first 100 results. Type to narrow down.</div>
								{/if}
							</div>
						{:else if showDropdown && typeSearch && filteredTypes.length === 0}
							<div class="type-empty">
								No {blockType === 'resource' ? 'resource' : 'data source'} types matching "{typeSearch}" for {selectedProvider}
							</div>
						{/if}
					{:else}
						<input
							id="resource-type-input"
							type="text"
							bind:value={resourceType}
							oninput={() => (typeSearch = resourceType)}
							placeholder={`e.g. ${selectedProvider || 'aws'}_instance`}
							class="input"
						/>
					{/if}
				</div>
			{/if}

			<div class="form-field">
				<label for="block-name">Name</label>
				<input
					id="block-name"
					type="text"
					bind:value={name}
					placeholder={blockType === 'variable'
						? 'e.g. region'
						: blockType === 'output'
							? 'e.g. instance_id'
							: 'e.g. web'}
					class="input"
					onkeydown={(e) => e.key === 'Enter' && handleAdd()}
				/>
			</div>

			<div class="form-field">
				<label for="target-file">Target File</label>
				<select id="target-file" bind:value={targetFile} class="input">
					{#each files as f}
						<option value={f}>{f}</option>
					{/each}
					<option value="main.tf">main.tf (new)</option>
				</select>
			</div>

			<!-- Connected Dependencies -->
			{#if scaffoldDeps.length > 0 && needsResourceType}
				<div class="form-field">
					<label for="deps-section">Connected Resources</label>
					<div class="deps-list" id="deps-section">
						{#each scaffoldDeps as dep}
							{@const exists = depExists(dep)}
							{@const key = `${dep.blockType}.${dep.resourceType}.${dep.name}`}
							<label class="dep-item" class:exists>
								<input
									type="checkbox"
									checked={isDepSelected(dep) || exists}
									disabled={exists}
									onchange={() => toggleDep(dep)}
								/>
								<div class="dep-info">
									<div class="dep-type">
										<span class="dep-block-type">{dep.blockType}</span>
										{dep.resourceType}.{dep.name}
										{#if dep.required}
											<span class="dep-required">required</span>
										{/if}
										{#if exists}
											<span class="dep-exists">exists</span>
										{/if}
									</div>
									<div class="dep-reason">{dep.reason}</div>
									{#if dep.linkAttr}
										<div class="dep-link">Sets {dep.linkAttr} = {dep.linkExpr}</div>
									{/if}
								</div>
							</label>
						{/each}
					</div>
				</div>
			{:else if loadingScaffold}
				<div class="form-field">
					<label>Connected Resources</label>
					<div class="deps-loading">Loading scaffold info...</div>
				</div>
			{/if}

			<!-- Preview -->
			{#if (needsResourceType && resourceType && name) || (!needsResourceType && name)}
				<div class="preview">
					<div class="preview-label">
						Will create {1 + scaffoldDeps.filter((d) => isDepSelected(d) && !depExists(d)).length} block{scaffoldDeps.filter((d) => isDepSelected(d) && !depExists(d)).length > 0 ? 's' : ''}
					</div>
					<pre class="preview-code">{#each scaffoldDeps as dep}{#if isDepSelected(dep) && !depExists(dep)}{dep.blockType} "{dep.resourceType}" "{dep.name}" {'{'} ... {'}'}
{/if}{/each}{#if blockType === 'resource'}resource "{resourceType}" "{name}" {'{'}
{#each Object.entries(scaffoldAttrs) as [k, v]}  {k} = {v}
{/each}{'}'}
{:else if blockType === 'data'}data "{resourceType}" "{name}" {'{'}
{#each Object.entries(scaffoldAttrs) as [k, v]}  {k} = {v}
{/each}{'}'}
{:else if blockType === 'variable'}variable "{name}" {'{'}
  type = string
{'}'}
{:else if blockType === 'output'}output "{name}" {'{'}
  value = ""
{'}'}{/if}</pre>
				</div>
			{/if}
		</div>

		<div class="dialog-footer">
			<button class="btn" onclick={onclose}>Cancel</button>
			<button
				class="btn primary"
				onclick={handleAdd}
				disabled={isProvider ? !newProviderName : (!name.trim() || (needsResourceType && !resourceType.trim()))}
			>
				{isProvider ? `Add ${newProviderName || 'provider'}` : `Add ${blockType}`}
			</button>
		</div>
	</div>
</div>

<style>
	.overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.6);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
	}

	.dialog {
		background: #1c1c1f;
		border: 1px solid #27272a;
		border-radius: 12px;
		width: 520px;
		max-height: 85vh;
		display: flex;
		flex-direction: column;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
	}

	.dialog-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid #27272a;
	}

	.dialog-header h3 {
		font-size: 16px;
		font-weight: 600;
	}

	.close-btn {
		width: 24px;
		height: 24px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: none;
		border: none;
		color: #71717a;
		cursor: pointer;
		border-radius: 4px;
		font-size: 16px;
	}

	.close-btn:hover {
		background: #27272a;
		color: #e4e4e7;
	}

	.dialog-body {
		padding: 20px;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.form-field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.form-field label {
		font-size: 12px;
		font-weight: 500;
		color: #a1a1aa;
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.count-hint {
		font-weight: 400;
		color: #52525b;
	}

	.type-selector {
		display: flex;
		gap: 4px;
	}

	.type-selector button {
		flex: 1;
		padding: 8px;
		font-size: 12px;
		border: 1px solid #27272a;
		border-radius: 6px;
		background: #09090b;
		color: #71717a;
		cursor: pointer;
		transition: all 0.15s;
	}

	.type-selector button:hover {
		border-color: #3f3f46;
		color: #a1a1aa;
	}

	.type-selector button.active {
		border-color: #3b82f6;
		color: #3b82f6;
		background: rgba(59, 130, 246, 0.1);
	}

	.provider-selector {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
	}

	.provider-chip {
		padding: 6px 10px;
		font-size: 12px;
		border: 1px solid #27272a;
		border-radius: 6px;
		background: #09090b;
		color: #71717a;
		cursor: pointer;
		transition: all 0.15s;
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.provider-chip:hover {
		border-color: #3f3f46;
		color: #a1a1aa;
	}

	.provider-chip.active {
		border-color: #22c55e;
		color: #22c55e;
		background: rgba(34, 197, 94, 0.1);
	}

	.provider-count {
		font-size: 10px;
		padding: 0 4px;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.06);
		color: #52525b;
	}

	.provider-chip.active .provider-count {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
	}

	.schema-notice {
		padding: 12px;
		background: rgba(245, 158, 11, 0.05);
		border: 1px solid rgba(245, 158, 11, 0.2);
		border-radius: 6px;
		font-size: 12px;
		color: #a1a1aa;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.load-schema-btn {
		padding: 6px 12px;
		font-size: 12px;
		background: rgba(245, 158, 11, 0.15);
		border: 1px solid rgba(245, 158, 11, 0.3);
		border-radius: 6px;
		color: #f59e0b;
		cursor: pointer;
		align-self: flex-start;
	}

	.load-schema-btn:hover:not(:disabled) {
		background: rgba(245, 158, 11, 0.25);
	}

	.load-schema-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.autocomplete-wrapper {
		position: relative;
	}

	.autocomplete-wrapper .input {
		padding-right: 28px;
	}

	.check-mark {
		position: absolute;
		right: 8px;
		top: 50%;
		transform: translateY(-50%);
		color: #22c55e;
		font-size: 14px;
	}

	.input {
		padding: 8px 12px;
		background: #09090b;
		border: 1px solid #27272a;
		border-radius: 6px;
		color: #e4e4e7;
		font-size: 13px;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		outline: none;
		width: 100%;
	}

	.input:focus {
		border-color: #3b82f6;
	}

	.input::placeholder {
		color: #3f3f46;
		font-family: -apple-system, BlinkMacSystemFont, sans-serif;
	}

	select.input {
		cursor: pointer;
		font-family: -apple-system, BlinkMacSystemFont, sans-serif;
	}

	.type-list {
		max-height: 200px;
		overflow-y: auto;
		border: 1px solid #27272a;
		border-radius: 6px;
		background: #0a0a0c;
	}

	.type-option {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 6px 12px;
		text-align: left;
		font-size: 12px;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		background: none;
		border: none;
		color: #a1a1aa;
		cursor: pointer;
		gap: 8px;
	}

	.type-option:hover {
		background: rgba(59, 130, 246, 0.1);
		color: #e4e4e7;
	}

	.type-option.selected {
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
	}

	.type-name {
		flex: 1;
	}

	.type-overflow {
		padding: 8px 12px;
		font-size: 11px;
		color: #52525b;
		text-align: center;
		border-top: 1px solid #27272a;
	}

	.type-empty {
		padding: 12px;
		font-size: 12px;
		color: #52525b;
		text-align: center;
		border: 1px solid #27272a;
		border-radius: 6px;
		background: #0a0a0c;
	}

	.provider-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 6px;
	}

	.provider-card {
		padding: 10px;
		border: 1px solid var(--border, #2f3146);
		border-radius: 6px;
		background: var(--bg-input, #15161e);
		cursor: pointer;
		text-align: left;
		transition: all 0.15s;
	}

	.provider-card:hover {
		border-color: var(--text-muted, #7982a9);
	}

	.provider-card.active {
		border-color: var(--accent, #7aa2f7);
		background: rgba(122, 162, 247, 0.1);
	}

	.provider-card-name {
		font-size: 14px;
		font-weight: 600;
		color: var(--text, #c0caf5);
	}

	.provider-card-source {
		font-size: 11px;
		color: var(--text-subtle, #565f89);
		font-family: monospace;
		margin-top: 2px;
	}

	.deps-list {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.dep-item {
		display: flex;
		align-items: flex-start;
		gap: 8px;
		padding: 8px 10px;
		border: 1px solid #27272a;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.1s;
		background: #09090b;
	}

	.dep-item:hover {
		border-color: #3f3f46;
	}

	.dep-item.exists {
		opacity: 0.5;
	}

	.dep-item input[type='checkbox'] {
		margin-top: 2px;
		flex-shrink: 0;
		accent-color: #22c55e;
	}

	.dep-info {
		flex: 1;
		min-width: 0;
	}

	.dep-type {
		font-size: 13px;
		font-weight: 500;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		display: flex;
		align-items: center;
		gap: 6px;
		flex-wrap: wrap;
	}

	.dep-block-type {
		font-size: 11px;
		padding: 0 4px;
		border-radius: 3px;
		background: rgba(139, 92, 246, 0.15);
		color: #a78bfa;
		font-weight: 400;
	}

	.dep-required {
		font-size: 11px;
		padding: 0 4px;
		border-radius: 3px;
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
		font-weight: 400;
		font-family: -apple-system, sans-serif;
	}

	.dep-exists {
		font-size: 11px;
		padding: 0 4px;
		border-radius: 3px;
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
		font-weight: 400;
		font-family: -apple-system, sans-serif;
	}

	.dep-reason {
		font-size: 12px;
		color: #71717a;
		margin-top: 2px;
	}

	.dep-link {
		font-size: 11px;
		color: #52525b;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		margin-top: 2px;
	}

	.deps-loading {
		color: #52525b;
		font-size: 13px;
		padding: 8px;
	}

	.preview {
		border: 1px solid #27272a;
		border-radius: 6px;
		overflow: hidden;
	}

	.preview-label {
		padding: 6px 12px;
		font-size: 10px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: #52525b;
		background: rgba(255, 255, 255, 0.02);
		border-bottom: 1px solid #27272a;
	}

	.preview-code {
		padding: 10px 12px;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 12px;
		color: #a1a1aa;
		background: #09090b;
		margin: 0;
		line-height: 1.5;
	}

	.dialog-footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		padding: 16px 20px;
		border-top: 1px solid #27272a;
	}

	.btn {
		padding: 8px 16px;
		font-size: 13px;
		border: 1px solid #27272a;
		border-radius: 6px;
		background: #27272a;
		color: #e4e4e7;
		cursor: pointer;
		transition: all 0.15s;
	}

	.btn:hover:not(:disabled) {
		background: #3f3f46;
	}

	.btn:disabled {
		opacity: 0.4;
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
</style>

<script lang="ts">
	import { workspace } from '$lib/stores/workspace.svelte';
	import { NODE_KIND_CONFIG, PLAN_ACTION_CONFIG } from '$lib/types';
	import type { SchemaAttribute, SchemaBlockType } from '$lib/types';

	let activeTab = $state<'attributes' | 'source' | 'plan' | 'eval'>('attributes');
	let editingField = $state<string | null>(null);
	let editValue = $state('');
	let renamingNode = $state(false);
	let renameValue = $state('');

	const node = $derived(workspace.selectedNode);
	const config = $derived(node ? NODE_KIND_CONFIG[node.kind] : null);
	const planChange = $derived(workspace.selectedNodePlan);
	const schema = $derived(workspace.selectedNodeSchema);

	// Build a map of currently-set attributes by name
	const setAttributes = $derived.by(() => {
		const m = new Map<string, { expression?: string; value?: unknown; references?: string[] }>();
		if (!node) return m;
		for (const attr of node.attributes ?? []) {
			m.set(attr.name, attr);
		}
		return m;
	});

	// Schema attributes categorized for display
	// Required attributes (always shown prominently)
	const schemaAttrsRequired = $derived.by(() => {
		if (!schema) return [];
		return schema.attributes.filter((a) => a.required);
	});

	// Optional, user-settable attributes (not purely computed)
	const schemaAttrsOptional = $derived.by(() => {
		if (!schema) return [];
		return schema.attributes.filter((a) => a.optional && !a.required);
	});

	// Computed-only attributes (read-only, no user input possible)
	const schemaAttrsComputed = $derived.by(() => {
		if (!schema) return [];
		return schema.attributes.filter(
			(a) => a.computed && !a.optional && !a.required
		);
	});

	let showComputed = $state(false);

	function startEdit(attrName: string, currentExpression: string) {
		editingField = attrName;
		editValue = currentExpression;
	}

	async function saveEdit(attrName: string) {
		if (editingField !== attrName) return;
		await workspace.patchAttribute(attrName, editValue);
		editingField = null;
	}

	function cancelEdit() {
		editingField = null;
	}

	function formatType(type: unknown): string {
		if (typeof type === 'string') return type;
		if (Array.isArray(type)) {
			if (type[0] === 'list' || type[0] === 'set') return `${type[0]}(${formatType(type[1])})`;
			if (type[0] === 'map') return `map(${formatType(type[1])})`;
			if (type[0] === 'object') return 'object';
			return type.join(', ');
		}
		return String(type);
	}

	function defaultForType(type: unknown): string {
		if (type === 'string') return '""';
		if (type === 'number') return '0';
		if (type === 'bool') return 'false';
		if (Array.isArray(type)) {
			if (type[0] === 'list' || type[0] === 'set') return '[]';
			if (type[0] === 'map') return '{}';
		}
		return '""';
	}

	async function handleDelete() {
		if (!node) return;
		if (!confirm(`Remove ${node.address}?`)) return;
		await workspace.removeBlock(node.address, node.source.file);
	}
</script>

<div class="inspector">
	{#if node && config}
		<div class="inspector-header" style:border-left-color={config.color}>
			<div class="header-kind">
				<span style:color={config.color}>{config.icon}</span>
				{config.label}
				<button class="delete-btn" onclick={handleDelete} title="Remove block">x</button>
			</div>
			{#if renamingNode}
				<div class="rename-row">
					<input
						class="rename-input"
						bind:value={renameValue}
						onkeydown={(e) => {
							if (e.key === 'Enter' && renameValue.trim()) {
								workspace.renameBlock(renameValue.trim());
								renamingNode = false;
							}
							if (e.key === 'Escape') renamingNode = false;
						}}
					/>
					<button class="save-btn" onclick={() => { workspace.renameBlock(renameValue.trim()); renamingNode = false; }}>Save</button>
					<button class="cancel-btn" onclick={() => renamingNode = false}>x</button>
				</div>
			{:else}
				<h3 class="header-name">
					{node.name}
					<button class="edit-btn inline" onclick={() => { renamingNode = true; renameValue = node.name; }} title="Rename">rename</button>
				</h3>
			{/if}
			<div class="header-address">{node.address}</div>
			{#if node.source}
				<div class="header-file">{node.source.file}:{node.source.startLine}</div>
			{/if}
			{#if schema && !workspace.schemaLoading}
				<div class="schema-badge">Schema loaded</div>
			{:else if workspace.schemaLoading}
				<div class="schema-badge loading">Loading schema...</div>
			{/if}
		</div>

		<div class="edit-hcl-bar">
			<button class="edit-hcl-btn" onclick={() => (workspace.showHCLEditor = true)}>
				Edit HCL
			</button>
		</div>

		{#if planChange}
			<div
				class="plan-banner"
				style:background-color={PLAN_ACTION_CONFIG[planChange.action].color + '20'}
				style:border-color={PLAN_ACTION_CONFIG[planChange.action].color}
			>
				<span class="plan-action-icon">{PLAN_ACTION_CONFIG[planChange.action].icon}</span>
				This resource will be
				<strong>{PLAN_ACTION_CONFIG[planChange.action].label.toLowerCase()}d</strong>
			</div>
		{/if}

		{#if (node.kind === 'resource' || node.kind === 'data' || node.kind === 'output') && !workspace.connectingMode}
			<div class="connect-bar">
				<button class="connect-btn" onclick={() => (workspace.connectingMode = true)}>
					Connect to another resource...
				</button>
			</div>
		{:else if workspace.connectingMode}
			<div class="connect-bar active">
				<span>Click a highlighted node to create a reference</span>
				<button class="cancel-connect" onclick={() => (workspace.connectingMode = false)}>Cancel</button>
			</div>
		{/if}

		<div class="tabs">
			<button class:active={activeTab === 'attributes'} onclick={() => (activeTab = 'attributes')}>
				Attributes
			</button>
			<button class:active={activeTab === 'source'} onclick={() => (activeTab = 'source')}>
				Source
			</button>
			{#if planChange}
				<button class:active={activeTab === 'plan'} onclick={() => (activeTab = 'plan')}>
					Plan
				</button>
			{/if}
			{#if node.kind === 'data'}
				<button
					class:active={activeTab === 'eval'}
					onclick={() => {
						activeTab = 'eval';
						if (!workspace.evalResult && !workspace.evaluating) {
							workspace.evalData(node.address);
						}
					}}
				>
					Evaluate
					{#if workspace.evaluating}
						...
					{:else if workspace.evalResult}
						&#10003;
					{/if}
				</button>
			{/if}
		</div>

		<div class="tab-content">
			{#if activeTab === 'attributes'}
				<!-- Identity section for variables/outputs -->
				<div class="section">
					{#if node.description}
						<div class="field">
							<div class="field-label">Description</div>
							<div class="field-value text">{node.description}</div>
						</div>
					{/if}
					{#if node.provider}
						<div class="field">
							<div class="field-label">Provider</div>
							<div class="field-value code">{node.provider}</div>
						</div>
					{/if}
					{#if node.varType}
						<div class="field">
							<div class="field-label">Type</div>
							<div class="field-value code">{node.varType}</div>
						</div>
					{/if}
					{#if node.default !== undefined && node.default !== null}
						<div class="field">
							<div class="field-label">Default</div>
							<div class="field-value code">{JSON.stringify(node.default)}</div>
						</div>
					{/if}
					{#if node.moduleSource}
						<div class="field">
							<div class="field-label">Source</div>
							<div class="field-value code">{node.moduleSource}</div>
						</div>
					{/if}
				</div>

				<!-- Schema-driven attributes -->
				{#if schema}
					<!-- Required attributes (always visible) -->
					{#if schemaAttrsRequired.length > 0}
						<div class="section">
							<h4 class="section-title">Required ({schemaAttrsRequired.length})</h4>
							{#each schemaAttrsRequired as sAttr}
								{@const current = setAttributes.get(sAttr.name)}
								{#if current}
									{@render schemaField(sAttr, current.expression || JSON.stringify(current.value) || '', current.references)}
								{:else}
									{@render unsetSchemaField(sAttr)}
								{/if}
							{/each}
						</div>
					{/if}

					<!-- Optional attributes (all visible) -->
					{#if schemaAttrsOptional.length > 0}
						<div class="section">
							<h4 class="section-title">Optional ({schemaAttrsOptional.length})</h4>
							{#each schemaAttrsOptional as sAttr}
								{@const current = setAttributes.get(sAttr.name)}
								{#if current}
									{@render schemaField(sAttr, current.expression || JSON.stringify(current.value) || '', current.references)}
								{:else}
									{@render unsetSchemaField(sAttr)}
								{/if}
							{/each}
						</div>
					{/if}

					<!-- Computed-only attributes (collapsible since they're read-only) -->
					{#if schemaAttrsComputed.length > 0}
						<div class="section">
							<h4 class="section-title collapsible">
								<button class="collapse-btn" onclick={() => (showComputed = !showComputed)}>
									{showComputed ? '▾' : '▸'} Computed ({schemaAttrsComputed.length})
								</button>
							</h4>
							{#if showComputed}
								{#each schemaAttrsComputed as sAttr}
									<div class="field computed">
										<div class="field-label">
											{sAttr.name}
											<span class="type-badge">{formatType(sAttr.type)}</span>
											<span class="computed-badge">computed</span>
										</div>
										{#if sAttr.description}
											<div class="field-desc">{sAttr.description}</div>
										{/if}
									</div>
								{/each}
							{/if}
						</div>
					{/if}

					<!-- Nested block types from schema -->
					{#if schema.blockTypes && schema.blockTypes.length > 0}
						<div class="section">
							<h4 class="section-title">Block Types ({schema.blockTypes.length})</h4>
							{#each schema.blockTypes as bt}
								{@render schemaBlockType(bt)}
							{/each}
						</div>
					{/if}
				{:else}
					<!-- Fallback: show raw attributes when no schema -->
					{#if node.attributes && node.attributes.length > 0}
						<div class="section">
							<h4 class="section-title">Attributes</h4>
							{#each node.attributes as attr}
								<div class="field">
									<div class="field-label">
										{attr.name}
										{#if editingField !== attr.name && !attr.references?.length}
											<button
												class="edit-btn"
												onclick={() =>
													startEdit(
														attr.name,
														attr.expression || JSON.stringify(attr.value) || ''
													)}>edit</button
											>
										{/if}
									</div>
									{#if editingField === attr.name}
										{@render editField(attr.name)}
									{:else}
										<div class="field-value code">
											{attr.expression || JSON.stringify(attr.value) || '(empty)'}
										</div>
									{/if}
									{#if attr.references && attr.references.length > 0}
										<div class="field-refs">
											{#each attr.references as ref}
												{@const targetId = workspace.resolveRef(ref)}{#if targetId}<button class="ref-tag clickable" onclick={() => workspace.selectNode(targetId)}>{ref}</button>{:else}<span class="ref-tag">{ref}</span>{/if}
											{/each}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				{/if}

				<!-- Nested blocks from parsed HCL -->
				{#if node.nestedBlocks && node.nestedBlocks.length > 0}
					<div class="section">
						<h4 class="section-title">Nested Blocks</h4>
						{#each node.nestedBlocks as block}
							<div class="nested-block">
								<div class="block-header">
									{block.type}
									{#if block.labels}
										{#each block.labels as label}
											<span class="block-label">"{label}"</span>
										{/each}
									{/if}
								</div>
								{#if block.attributes}
									{#each block.attributes as attr}
										<div class="field nested">
											<div class="field-label">{attr.name}</div>
											<div class="field-value code">
												{attr.expression || JSON.stringify(attr.value) || '(empty)'}
											</div>
										</div>
									{/each}
								{/if}
							</div>
						{/each}
					</div>
				{/if}

				{#if node.dependsOn && node.dependsOn.length > 0}
					<div class="section">
						<h4 class="section-title">Dependencies</h4>
						{#each node.dependsOn as dep}
							{@const depId = workspace.resolveRef(dep)}
							{#if depId}
								<button class="dep-item clickable" onclick={() => workspace.selectNode(depId)}>{dep}</button>
							{:else}
								<div class="dep-item">{dep}</div>
							{/if}
						{/each}
					</div>
				{/if}

				<!-- Used By: incoming references -->
				{#if workspace.incomingRefs.length > 0}
					<div class="section">
						<h4 class="section-title">Used By ({workspace.incomingRefs.length})</h4>
						{#each workspace.incomingRefs as incoming}
							{@const inConfig = NODE_KIND_CONFIG[incoming.nodeKind as import('$lib/types').NodeKind]}
							<button class="used-by-item" onclick={() => workspace.selectNode(incoming.nodeId)}>
								<span class="used-by-icon" style:color={inConfig?.color}>{inConfig?.icon}</span>
								<div class="used-by-info">
									<div class="used-by-name">{incoming.nodeAddress}</div>
									<div class="used-by-attr">via .{incoming.attribute}</div>
								</div>
								<span class="used-by-arrow">&#8594;</span>
							</button>
						{/each}
					</div>
				{/if}
			{:else if activeTab === 'source'}
				<div class="source-view">
					{#if node.rawHCL}
						<pre><code>{node.rawHCL}</code></pre>
					{:else}
						<p class="no-source">No source available</p>
					{/if}
				</div>
			{:else if activeTab === 'plan' && planChange}
				<div class="plan-diff">
					{#if planChange.before || planChange.after}
						<h4 class="section-title">Changes</h4>
						{@const allKeys = new Set([
							...Object.keys(planChange.before || {}),
							...Object.keys(planChange.after || {})
						])}
						{#each [...allKeys] as key}
							{@const before = planChange.before?.[key]}
							{@const after = planChange.after?.[key]}
							{@const changed = JSON.stringify(before) !== JSON.stringify(after)}
							{@const unknown = planChange.afterUnknown?.[key]}
							{#if changed || unknown}
								<div class="diff-field">
									<div class="diff-key">{key}</div>
									{#if before !== undefined}
										<div class="diff-before">- {JSON.stringify(before)}</div>
									{/if}
									{#if unknown}
										<div class="diff-after unknown">+ (known after apply)</div>
									{:else if after !== undefined}
										<div class="diff-after">+ {JSON.stringify(after)}</div>
									{/if}
								</div>
							{/if}
						{/each}
					{:else}
						<p class="no-source">No detailed diff available</p>
					{/if}
				</div>

			{:else if activeTab === 'eval'}
				<div class="eval-section">
					<div class="eval-header">
						<button
							class="eval-btn"
							onclick={() => node && workspace.evalData(node.address)}
							disabled={workspace.evaluating}
						>
							{workspace.evaluating ? 'Evaluating...' : 'Run Evaluation'}
						</button>
						<span class="eval-hint">Runs a targeted plan to fetch live values from the provider</span>
					</div>

					{#if workspace.evaluating}
						<div class="eval-loading">
							<div class="spinner"></div>
							<span>Querying provider...</span>
						</div>
					{:else if workspace.evalError}
						<div class="eval-error">
							<div class="eval-error-title">Evaluation Failed</div>
							<pre class="eval-error-detail">{workspace.evalError}</pre>
						</div>
					{:else if workspace.evalResult}
						<div class="eval-results">
							<h4 class="section-title">Returned Values ({Object.keys(workspace.evalResult).length})</h4>
							{#each Object.entries(workspace.evalResult).sort(([a], [b]) => a.localeCompare(b)) as [key, value]}
								<div class="eval-field">
									<div class="eval-key">{key}</div>
									<div class="eval-value">
										{#if typeof value === 'object' && value !== null}
											<pre class="eval-json">{JSON.stringify(value, null, 2)}</pre>
										{:else if value === null || value === undefined}
											<span class="eval-null">null</span>
										{:else}
											<span class="eval-literal">{String(value)}</span>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					{:else}
						<div class="eval-empty">
							Click "Run Evaluation" to query the provider and see what values this data source returns.
						</div>
					{/if}
				</div>
			{/if}
		</div>
	{:else}
		<div class="empty-inspector">
			<p>Select a node to inspect</p>
		</div>
	{/if}
</div>

{#snippet schemaField(sAttr: SchemaAttribute, currentValue: string, refs?: string[])}
	<div class="field">
		<div class="field-label">
			{sAttr.name}
			<span class="type-badge">{formatType(sAttr.type)}</span>
			{#if sAttr.required}
				<span class="required-badge">required</span>
			{/if}
			{#if sAttr.computed && sAttr.optional}
				<span class="computed-badge">computed</span>
			{/if}
			{#if editingField !== sAttr.name}
				<button class="edit-btn" onclick={() => startEdit(sAttr.name, currentValue)}>edit</button>
				{#if !sAttr.required}
					<button class="edit-btn remove" onclick={() => workspace.removeAttribute(sAttr.name)} title="Remove attribute">x</button>
				{/if}
			{/if}
		</div>
		{#if sAttr.description}
			<div class="field-desc">{sAttr.description}</div>
		{/if}
		{#if editingField === sAttr.name}
			{@render editField(sAttr.name)}
		{:else}
			<div class="field-value code">{currentValue || '(empty)'}</div>
		{/if}
		{#if refs && refs.length > 0}
			<div class="field-refs">
				{#each refs as ref}
					{@const targetId = workspace.resolveRef(ref)}{#if targetId}<button class="ref-tag clickable" onclick={() => workspace.selectNode(targetId)}>{ref}</button>{:else}<span class="ref-tag">{ref}</span>{/if}
				{/each}
			</div>
		{/if}
	</div>
{/snippet}

{#snippet unsetSchemaField(sAttr: SchemaAttribute)}
	<div class="field unset">
		<div class="field-label">
			{sAttr.name}
			<span class="type-badge">{formatType(sAttr.type)}</span>
			{#if sAttr.required}
				<span class="required-badge">required</span>
			{:else}
				<span class="optional-badge">optional</span>
			{/if}
			<button
				class="set-btn"
				onclick={() => startEdit(sAttr.name, defaultForType(sAttr.type))}
			>
				+ set
			</button>
		</div>
		{#if sAttr.description}
			<div class="field-desc">{sAttr.description}</div>
		{/if}
		{#if editingField === sAttr.name}
			{@render editField(sAttr.name)}
		{/if}
	</div>
{/snippet}

{#snippet schemaBlockType(bt: SchemaBlockType)}
	<div class="nested-block">
		<div class="block-header">
			{bt.name}
			<span class="type-badge">{bt.nestingMode}</span>
			{#if bt.minItems}
				<span class="required-badge">min: {bt.minItems}</span>
			{/if}
		</div>
		{#if bt.attributes}
			{#each bt.attributes.slice(0, 5) as attr}
				<div class="field nested computed">
					<div class="field-label">
						{attr.name}
						<span class="type-badge">{formatType(attr.type)}</span>
						{#if attr.required}<span class="required-badge">req</span>{/if}
					</div>
				</div>
			{/each}
			{#if bt.attributes.length > 5}
				<div class="field-desc nested">...and {bt.attributes.length - 5} more</div>
			{/if}
		{/if}
	</div>
{/snippet}

{#snippet editField(attrName: string)}
	<div class="edit-row">
		<input
			type="text"
			class="edit-input"
			bind:value={editValue}
			onkeydown={(e) => {
				if (e.key === 'Enter') saveEdit(attrName);
				if (e.key === 'Escape') cancelEdit();
			}}
		/>
		<button class="save-btn" onclick={() => saveEdit(attrName)}>Save</button>
		<button class="cancel-btn" onclick={cancelEdit}>x</button>
	</div>
{/snippet}

<style>
	.inspector {
		height: 100%;
		overflow-y: auto;
		font-size: 14px;
	}

	.inspector-header {
		padding: 12px 16px;
		border-left: 3px solid;
		background: rgba(255, 255, 255, 0.03);
	}

	.header-kind {
		font-size: 14px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		opacity: 0.6;
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.header-name {
		font-size: 18px;
		font-weight: 600;
		margin: 4px 0 2px;
	}

	.header-address {
		font-size: 14px;
		font-family: monospace;
		opacity: 0.5;
	}

	.header-file {
		font-size: 14px;
		opacity: 0.4;
		margin-top: 2px;
	}

	.delete-btn {
		margin-left: auto;
		width: 18px;
		height: 18px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: none;
		border: 1px solid transparent;
		border-radius: 3px;
		color: var(--text-subtle);
		cursor: pointer;
		font-size: 14px;
	}

	.delete-btn:hover {
		background: rgba(239, 68, 68, 0.15);
		border-color: #ef4444;
		color: #ef4444;
	}

	.schema-badge {
		font-size: 12px;
		margin-top: 4px;
		color: #22c55e;
		opacity: 0.6;
	}

	.schema-badge.loading {
		color: #f59e0b;
	}

	.plan-banner {
		margin: 8px 12px;
		padding: 8px 12px;
		border-radius: 6px;
		border-left: 3px solid;
		font-size: 14px;
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.plan-action-icon {
		font-weight: bold;
		font-size: 14px;
	}

	.edit-hcl-bar {
		padding: 8px 12px;
		border-bottom: 1px solid var(--border);
	}

	.edit-hcl-btn {
		width: 100%;
		padding: 8px 12px;
		font-size: 14px;
		font-weight: 500;
		background: var(--bg-base);
		border: 1px solid var(--border);
		border-radius: 6px;
		color: var(--accent, #7aa2f7);
		cursor: pointer;
		transition: all 0.15s;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
	}

	.edit-hcl-btn:hover {
		background: var(--bg-hover);
		border-color: var(--accent, #7aa2f7);
	}

	.connect-bar {
		padding: 8px 12px;
		border-bottom: 1px solid var(--border);
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 14px;
	}

	.connect-bar.active {
		background: rgba(34, 197, 94, 0.1);
		color: #22c55e;
	}

	.connect-btn {
		padding: 6px 12px;
		font-size: 14px;
		background: rgba(34, 197, 94, 0.1);
		border: 1px solid rgba(34, 197, 94, 0.3);
		border-radius: 6px;
		color: #22c55e;
		cursor: pointer;
		width: 100%;
		transition: all 0.15s;
	}

	.connect-btn:hover {
		background: rgba(34, 197, 94, 0.2);
		border-color: #22c55e;
	}

	.cancel-connect {
		padding: 4px 8px;
		font-size: 14px;
		background: none;
		border: 1px solid var(--border);
		border-radius: 4px;
		color: var(--text-muted);
		cursor: pointer;
		margin-left: auto;
	}

	.tabs {
		display: flex;
		border-bottom: 1px solid var(--border);
		padding: 0 12px;
	}

	.tabs button {
		padding: 8px 12px;
		font-size: 14px;
		color: var(--text-muted);
		background: none;
		border: none;
		border-bottom: 2px solid transparent;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.tabs button:hover {
		color: #a1a1aa;
	}

	.tabs button.active {
		color: var(--text);
		border-bottom-color: var(--accent);
	}

	.tab-content {
		padding: 12px 16px;
	}

	.section {
		margin-bottom: 16px;
	}

	.section-title {
		font-size: 14px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-muted);
		margin-bottom: 8px;
		padding-bottom: 4px;
		border-bottom: 1px solid var(--border);
	}

	.section-title.collapsible {
		border-bottom: none;
		padding-bottom: 0;
	}

	.collapse-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		font-size: 14px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		padding: 0;
	}

	.collapse-btn:hover {
		color: #a1a1aa;
	}

	.field {
		margin-bottom: 8px;
	}

	.field.nested {
		padding-left: 12px;
	}

	.field.unset {
		opacity: 0.7;
		border-left: 2px solid var(--border);
		padding-left: 8px;
	}

	.field.unset:hover {
		opacity: 1;
		border-left-color: var(--accent);
	}

	.field.computed {
		opacity: 0.5;
	}

	.field-label {
		font-size: 14px;
		color: var(--text-muted);
		margin-bottom: 2px;
		display: flex;
		align-items: center;
		gap: 4px;
		flex-wrap: wrap;
	}

	.field-value {
		color: var(--text);
	}

	.field-value.code {
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 14px;
		background: rgba(0, 0, 0, 0.2);
		padding: 2px 6px;
		border-radius: 3px;
		word-break: break-all;
	}

	.field-value.text {
		font-size: 14px;
		color: #a1a1aa;
	}

	.field-desc {
		font-size: 13px;
		color: #52525b;
		margin-top: 1px;
		line-height: 1.3;
	}

	.type-badge {
		font-size: 12px;
		padding: 0 4px;
		border-radius: 3px;
		background: rgba(99, 102, 241, 0.15);
		color: #818cf8;
		font-family: monospace;
	}

	.required-badge {
		font-size: 12px;
		padding: 0 4px;
		border-radius: 3px;
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
	}

	.optional-badge {
		font-size: 12px;
		padding: 0 4px;
		border-radius: 3px;
		background: rgba(113, 113, 122, 0.15);
		color: #71717a;
	}

	.computed-badge {
		font-size: 12px;
		padding: 0 4px;
		border-radius: 3px;
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
	}

	.set-btn {
		font-size: 12px;
		padding: 0 6px;
		background: none;
		border: 1px solid #27272a;
		border-radius: 3px;
		color: #3b82f6;
		cursor: pointer;
		margin-left: auto;
		opacity: 0.6;
		transition: all 0.1s;
	}

	.set-btn:hover {
		opacity: 1;
		background: rgba(59, 130, 246, 0.1);
		border-color: #3b82f6;
	}

	.field-refs {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin-top: 4px;
	}

	.ref-tag {
		font-size: 14px;
		padding: 1px 6px;
		border-radius: 3px;
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
		font-family: monospace;
		border: none;
	}

	.ref-tag.clickable {
		cursor: pointer;
		transition: all 0.1s;
	}

	.ref-tag.clickable:hover {
		background: rgba(59, 130, 246, 0.3);
		color: #93c5fd;
		text-decoration: underline;
	}

	.nested-block {
		margin-bottom: 12px;
		border-left: 2px solid var(--border);
		padding-left: 12px;
	}

	.block-header {
		font-size: 14px;
		font-weight: 600;
		color: #a1a1aa;
		margin-bottom: 6px;
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.block-label {
		font-size: 14px;
		color: #71717a;
	}

	.dep-item {
		font-family: monospace;
		font-size: 14px;
		padding: 4px 8px;
		background: rgba(245, 158, 11, 0.1);
		border-radius: 4px;
		margin-bottom: 4px;
		color: #fbbf24;
		border: none;
		display: block;
		width: 100%;
		text-align: left;
	}

	.dep-item.clickable {
		cursor: pointer;
		transition: all 0.1s;
	}

	.dep-item.clickable:hover {
		background: rgba(245, 158, 11, 0.2);
		text-decoration: underline;
	}

	.used-by-item {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 6px 8px;
		border: none;
		background: rgba(255, 255, 255, 0.02);
		border-radius: 4px;
		color: var(--text);
		cursor: pointer;
		text-align: left;
		margin-bottom: 4px;
		transition: background 0.1s;
	}

	.used-by-item:hover {
		background: rgba(59, 130, 246, 0.1);
	}

	.used-by-icon {
		font-size: 14px;
		flex-shrink: 0;
	}

	.used-by-info {
		flex: 1;
		min-width: 0;
	}

	.used-by-name {
		font-size: 14px;
		font-family: monospace;
		font-weight: 500;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.used-by-attr {
		font-size: 12px;
		color: #52525b;
	}

	.used-by-arrow {
		color: #3f3f46;
		font-size: 14px;
		flex-shrink: 0;
	}

	.source-view pre {
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 14px;
		line-height: 1.5;
		background: rgba(0, 0, 0, 0.3);
		padding: 12px;
		border-radius: 6px;
		overflow-x: auto;
		white-space: pre-wrap;
		word-break: break-all;
	}

	.no-source {
		color: #52525b;
		text-align: center;
		padding: 24px;
	}

	.diff-field {
		margin-bottom: 8px;
		font-family: monospace;
		font-size: 14px;
	}

	.diff-key {
		color: #a1a1aa;
		font-weight: 600;
		margin-bottom: 2px;
	}

	.diff-before {
		color: #ef4444;
		background: rgba(239, 68, 68, 0.1);
		padding: 2px 6px;
		border-radius: 3px;
	}

	.diff-after {
		color: #22c55e;
		background: rgba(34, 197, 94, 0.1);
		padding: 2px 6px;
		border-radius: 3px;
	}

	.diff-after.unknown {
		color: #f59e0b;
		background: rgba(245, 158, 11, 0.1);
		font-style: italic;
	}

	.edit-btn {
		font-size: 12px;
		padding: 0 4px;
		background: none;
		border: 1px solid #27272a;
		border-radius: 3px;
		color: #52525b;
		cursor: pointer;
		margin-left: 2px;
		opacity: 0;
		transition: opacity 0.1s;
	}

	.field:hover .edit-btn {
		opacity: 1;
	}

	.edit-btn:hover {
		color: #3b82f6;
		border-color: #3b82f6;
	}

	.edit-row {
		display: flex;
		gap: 4px;
		margin-top: 2px;
	}

	.edit-input {
		flex: 1;
		padding: 4px 6px;
		background: var(--bg-base);
		border: 1px solid var(--accent);
		border-radius: 3px;
		color: var(--text);
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 14px;
		outline: none;
	}

	.save-btn {
		padding: 4px 8px;
		font-size: 14px;
		background: var(--accent);
		border: none;
		border-radius: 3px;
		color: #fff;
		cursor: pointer;
	}

	.cancel-btn {
		padding: 4px 6px;
		font-size: 14px;
		background: none;
		border: 1px solid var(--border);
		border-radius: 3px;
		color: var(--text-muted);
		cursor: pointer;
	}

	.empty-inspector {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
		color: var(--text-subtle);
		font-size: 14px;
	}

	.eval-section {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.eval-header {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.eval-btn {
		padding: 8px 16px;
		font-size: 14px;
		font-weight: 500;
		background: rgba(139, 92, 246, 0.15);
		border: 1px solid rgba(139, 92, 246, 0.3);
		border-radius: 6px;
		color: #a78bfa;
		cursor: pointer;
		transition: all 0.15s;
	}

	.eval-btn:hover:not(:disabled) {
		background: rgba(139, 92, 246, 0.25);
		border-color: #8b5cf6;
	}

	.eval-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.eval-hint {
		font-size: 12px;
		color: #52525b;
	}

	.eval-loading {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 16px;
		color: #a78bfa;
		font-size: 14px;
	}

	.spinner {
		width: 16px;
		height: 16px;
		border: 2px solid #3f3f46;
		border-top-color: #8b5cf6;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.eval-error {
		padding: 12px;
		background: rgba(239, 68, 68, 0.05);
		border-radius: 6px;
	}

	.eval-error-title {
		font-weight: 600;
		color: #ef4444;
		margin-bottom: 6px;
	}

	.eval-error-detail {
		white-space: pre-wrap;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 13px;
		color: #f87171;
		background: rgba(0, 0, 0, 0.2);
		padding: 8px;
		border-radius: 4px;
		margin: 0;
	}

	.eval-results {
		display: flex;
		flex-direction: column;
	}

	.eval-field {
		padding: 6px 0;
		border-bottom: 1px solid rgba(39, 39, 42, 0.5);
	}

	.eval-key {
		font-size: 13px;
		font-weight: 500;
		color: #a78bfa;
		margin-bottom: 2px;
	}

	.eval-value {
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 13px;
	}

	.eval-json {
		background: rgba(0, 0, 0, 0.2);
		padding: 6px 8px;
		border-radius: 4px;
		margin: 0;
		font-size: 12px;
		color: #a1a1aa;
		overflow-x: auto;
	}

	.eval-null {
		color: #52525b;
		font-style: italic;
	}

	.eval-literal {
		color: var(--text);
	}

	.eval-empty {
		color: #52525b;
		text-align: center;
		padding: 24px 12px;
		font-size: 14px;
		line-height: 1.5;
	}
</style>

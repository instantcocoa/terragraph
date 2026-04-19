<script lang="ts">
	import { workspace } from '$lib/stores/workspace.svelte';

	let {
		name,
		expression,
		value,
		references
	}: {
		name: string;
		expression: string;
		value?: unknown;
		references?: string[];
	} = $props();

	let mode = $state<'visual' | 'raw'>('visual');
	let editingRaw = $state(false);
	let rawValue = $state('');

	// Parse expression type
	const exprType = $derived.by(() => {
		const e = expression.trim();
		if (!e) return 'empty';
		if (e.startsWith('"') && e.endsWith('"')) return 'string';
		if (e === 'true' || e === 'false') return 'bool';
		if (/^\d+(\.\d+)?$/.test(e)) return 'number';
		if (e.startsWith('[') && e.endsWith(']')) return 'list';
		if (e.startsWith('{') && e.endsWith('}')) return 'map';
		if (/^(var|local|data|module)\.\w/.test(e)) return 'reference';
		if (/^\w+\(/.test(e)) return 'function';
		if (e.includes('?') && e.includes(':')) return 'conditional';
		if (e.startsWith('"') && e.includes('${')) return 'template';
		return 'expression';
	});

	// For string values, extract the inner string
	const stringValue = $derived(
		exprType === 'string' ? expression.slice(1, -1) : ''
	);

	// For bool values
	const boolValue = $derived(expression.trim() === 'true');

	async function saveValue(newExpr: string) {
		await workspace.patchAttribute(name, newExpr);
	}

	async function saveRaw() {
		if (rawValue !== expression) {
			await workspace.patchAttribute(name, rawValue);
		}
		editingRaw = false;
	}
</script>

<div class="expr-editor">
	{#if mode === 'visual'}
		{#if exprType === 'string'}
			<div class="expr-string">
				<input
					type="text"
					value={stringValue}
					onchange={(e) => saveValue(`"${(e.target as HTMLInputElement).value}"`)}
					class="expr-input"
				/>
			</div>

		{:else if exprType === 'bool'}
			<label class="expr-bool">
				<input
					type="checkbox"
					checked={boolValue}
					onchange={(e) => saveValue((e.target as HTMLInputElement).checked ? 'true' : 'false')}
				/>
				<span>{boolValue ? 'true' : 'false'}</span>
			</label>

		{:else if exprType === 'number'}
			<input
				type="number"
				value={expression}
				onchange={(e) => saveValue((e.target as HTMLInputElement).value)}
				class="expr-input narrow"
			/>

		{:else if exprType === 'reference'}
			{@const targetId = workspace.resolveRef(expression)}
			<div class="expr-ref">
				{#if targetId}
					<button class="expr-ref-link" onclick={() => workspace.selectNode(targetId)}>
						{expression}
					</button>
				{:else}
					<span class="expr-ref-text">{expression}</span>
				{/if}
				<button class="expr-mode-btn" onclick={() => { mode = 'raw'; rawValue = expression; }}>
					edit
				</button>
			</div>

		{:else if exprType === 'function' || exprType === 'conditional' || exprType === 'template' || exprType === 'list' || exprType === 'map'}
			<div class="expr-complex">
				<code class="expr-code">{expression}</code>
				<button class="expr-mode-btn" onclick={() => { mode = 'raw'; rawValue = expression; }}>
					edit
				</button>
			</div>

		{:else}
			<div class="expr-complex">
				<code class="expr-code">{expression || '(empty)'}</code>
				<button class="expr-mode-btn" onclick={() => { mode = 'raw'; rawValue = expression; }}>
					edit
				</button>
			</div>
		{/if}

		{#if references && references.length > 0}
			<div class="expr-refs">
				{#each references as ref}
					{@const targetId = workspace.resolveRef(ref)}
					{#if targetId}
						<button class="expr-ref-tag" onclick={() => workspace.selectNode(targetId)}>{ref}</button>
					{:else}
						<span class="expr-ref-tag plain">{ref}</span>
					{/if}
				{/each}
			</div>
		{/if}

	{:else}
		<!-- Raw expression editor -->
		<div class="expr-raw">
			<textarea
				class="expr-textarea"
				bind:value={rawValue}
				onkeydown={(e) => {
					if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); saveRaw(); }
					if (e.key === 'Escape') { mode = 'visual'; }
				}}
				rows={Math.min(8, rawValue.split('\n').length + 1)}
			></textarea>
			<div class="expr-raw-actions">
				<button class="expr-save" onclick={saveRaw}>Save</button>
				<button class="expr-cancel" onclick={() => (mode = 'visual')}>Cancel</button>
			</div>
		</div>
	{/if}
</div>

<style>
	.expr-editor {
		margin-top: 2px;
	}

	.expr-input {
		width: 100%;
		padding: 4px 8px;
		background: var(--bg-input, var(--bg-base));
		border: 1px solid var(--border);
		border-radius: 4px;
		color: var(--text);
		font-size: 14px;
		outline: none;
	}

	.expr-input:focus {
		border-color: var(--accent);
	}

	.expr-input.narrow {
		width: 120px;
	}

	.expr-bool {
		display: flex;
		align-items: center;
		gap: 6px;
		cursor: pointer;
		font-size: 14px;
		color: #ff9e64;
	}

	.expr-bool input {
		accent-color: var(--accent);
	}

	.expr-ref {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.expr-ref-link {
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 13px;
		color: #73daca;
		background: none;
		border: none;
		cursor: pointer;
		text-decoration: underline;
		text-decoration-color: transparent;
		transition: text-decoration-color 0.1s;
	}

	.expr-ref-link:hover {
		text-decoration-color: #73daca;
	}

	.expr-ref-text {
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 13px;
		color: #73daca;
	}

	.expr-complex {
		display: flex;
		align-items: flex-start;
		gap: 6px;
	}

	.expr-code {
		flex: 1;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 13px;
		color: var(--text);
		background: rgba(0, 0, 0, 0.15);
		padding: 4px 8px;
		border-radius: 4px;
		word-break: break-all;
		line-height: 1.4;
	}

	.expr-mode-btn {
		font-size: 12px;
		padding: 2px 6px;
		background: none;
		border: 1px solid var(--border);
		border-radius: 3px;
		color: var(--text-subtle);
		cursor: pointer;
		flex-shrink: 0;
	}

	.expr-mode-btn:hover {
		color: var(--accent);
		border-color: var(--accent);
	}

	.expr-refs {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin-top: 4px;
	}

	.expr-ref-tag {
		font-size: 12px;
		padding: 1px 6px;
		border-radius: 3px;
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
		font-family: monospace;
		border: none;
		cursor: pointer;
	}

	.expr-ref-tag:hover {
		background: rgba(59, 130, 246, 0.3);
		text-decoration: underline;
	}

	.expr-ref-tag.plain {
		cursor: default;
	}

	.expr-ref-tag.plain:hover {
		background: rgba(59, 130, 246, 0.15);
		text-decoration: none;
	}

	.expr-raw {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.expr-textarea {
		width: 100%;
		padding: 8px;
		background: var(--bg-input, var(--bg-base));
		border: 1px solid var(--accent);
		border-radius: 4px;
		color: var(--text);
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 13px;
		outline: none;
		resize: vertical;
		line-height: 1.4;
	}

	.expr-raw-actions {
		display: flex;
		gap: 4px;
	}

	.expr-save {
		padding: 4px 10px;
		font-size: 12px;
		background: rgba(34, 197, 94, 0.15);
		border: 1px solid rgba(34, 197, 94, 0.3);
		border-radius: 4px;
		color: #22c55e;
		cursor: pointer;
	}

	.expr-cancel {
		padding: 4px 10px;
		font-size: 12px;
		background: none;
		border: 1px solid var(--border);
		border-radius: 4px;
		color: var(--text-muted);
		cursor: pointer;
	}
</style>

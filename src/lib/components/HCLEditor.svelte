<script lang="ts">
	import { workspace } from '$lib/stores/workspace.svelte';

	let {
		onclose
	}: {
		onclose: () => void;
	} = $props();

	const node = $derived(workspace.selectedNode);
	let content = $state('');
	let originalContent = $state('');
	let fullFileContent = $state('');
	let saving = $state(false);
	let error = $state<string | null>(null);
	let dirty = $state(false);
	let loading = $state(true);
	let fullFilePath = $state('');

	// Load the full file and extract the block
	$effect(() => {
		if (!node || !workspace.path) return;
		loading = true;
		error = null;

		const path = `${workspace.path}/${node.source.file}`;
		fullFilePath = path;

		fetch(`/api/workspace/file?path=${encodeURIComponent(path)}`)
			.then((r) => r.json())
			.then((data) => {
				fullFileContent = data.content;
				// Extract block by line range
				const lines = data.content.split('\n');
				const start = node.source.startLine - 1;
				const end = node.source.endLine;
				content = lines.slice(start, end).join('\n');
				originalContent = content;
				loading = false;
			})
			.catch((e) => {
				error = e.message || 'Failed to load file';
				// Fallback to rawHCL
				content = node.rawHCL || '';
				originalContent = content;
				loading = false;
			});
	});

	async function handleSave() {
		if (!node || !workspace.path || !dirty) return;
		saving = true;
		error = null;

		try {
			// Rebuild the file by replacing the line range
			const lines = fullFileContent.split('\n');
			const start = node.source.startLine - 1;
			const end = node.source.endLine;
			const before = lines.slice(0, start);
			const after = lines.slice(end);
			const updated = [...before, content, ...after].join('\n');

			const res = await fetch('/api/workspace/write-file', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ path: fullFilePath, content: updated })
			});

			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: 'Write failed' }));
				throw new Error(body.error || `Write failed: ${res.status}`);
			}

			const selectedId = node.id;
			await workspace.load(workspace.path);
			workspace.selectedNodeId = selectedId;
			await workspace.refreshHistory();
			dirty = false;
			onclose();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Save failed';
		} finally {
			saving = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 's' && (e.metaKey || e.ctrlKey)) {
			e.preventDefault();
			handleSave();
		}
		if (e.key === 'Escape') {
			if (dirty) {
				if (confirm('Discard changes?')) onclose();
			} else {
				onclose();
			}
		}
		if (e.key === 'Tab') {
			e.preventDefault();
			const target = e.target as HTMLTextAreaElement;
			const start = target.selectionStart;
			const end = target.selectionEnd;
			content = content.substring(0, start) + '  ' + content.substring(end);
			dirty = true;
			requestAnimationFrame(() => {
				target.selectionStart = target.selectionEnd = start + 2;
			});
		}
	}

	// Simple HCL syntax highlighting for the overlay
	function highlightHCL(code: string): string {
		return code
			// Strings
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			// Comments
			.replace(/(#.*$|\/\/.*$)/gm, '<span class="hl-comment">$1</span>')
			// Strings (double-quoted)
			.replace(/("(?:[^"\\]|\\.)*")/g, '<span class="hl-string">$1</span>')
			// Keywords
			.replace(/\b(resource|data|variable|output|locals|module|provider|terraform|required_providers|required_version|for_each|count|depends_on|lifecycle|dynamic|content)\b/g, '<span class="hl-keyword">$1</span>')
			// Booleans and null
			.replace(/\b(true|false|null)\b/g, '<span class="hl-bool">$1</span>')
			// Numbers
			.replace(/\b(\d+)\b/g, '<span class="hl-number">$1</span>')
			// Functions
			.replace(/\b([a-z_]+)\s*\(/g, '<span class="hl-func">$1</span>(')
			// Type keywords
			.replace(/\b(string|number|bool|list|map|set|object|tuple|any)\b/g, '<span class="hl-type">$1</span>')
			// References like var. local. data. module.
			.replace(/\b(var|local|data|module|each|self)\./g, '<span class="hl-ref">$1</span>.');
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="editor-overlay" onmousedown={onclose}>
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="editor-modal" onmousedown={(e) => e.stopPropagation()}>
		<div class="editor-header">
			<div class="editor-title">
				{#if node}
					<span class="editor-kind">{node.kind}</span>
					<span class="editor-name">{node.address}</span>
					<span class="editor-file">{node.source.file}:{node.source.startLine}-{node.source.endLine}</span>
				{/if}
			</div>
			<div class="editor-actions">
				{#if dirty}
					<span class="unsaved-indicator">unsaved</span>
				{/if}
				<button class="editor-btn save" onclick={handleSave} disabled={saving || !dirty}>
					{saving ? 'Saving...' : 'Save'} <kbd>Cmd+S</kbd>
				</button>
				<button class="editor-btn" onclick={onclose}>Close <kbd>Esc</kbd></button>
			</div>
		</div>

		{#if error}
			<div class="editor-error">{error}</div>
		{/if}

		<div class="editor-body">
			{#if loading}
				<div class="editor-loading">Loading...</div>
			{:else}
				<div class="line-numbers" aria-hidden="true">
					{#each content.split('\n') as _, i}
						<span>{(node?.source.startLine ?? 1) + i}</span>
					{/each}
				</div>
				<div class="code-area">
					<pre class="highlight-layer" aria-hidden="true">{@html highlightHCL(content)}{'\n'}</pre>
					<textarea
						class="editor-textarea"
						bind:value={content}
						oninput={() => (dirty = true)}
						onkeydown={handleKeydown}
						spellcheck="false"
						autocomplete="off"
					></textarea>
				</div>
			{/if}
		</div>
	</div>
</div>

<style>
	.editor-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 200;
		backdrop-filter: blur(4px);
	}

	.editor-modal {
		background: var(--bg-panel, #1f2133);
		border: 1px solid var(--border, #2f3146);
		border-radius: 12px;
		width: min(920px, 92vw);
		height: min(720px, 88vh);
		display: flex;
		flex-direction: column;
		box-shadow: 0 16px 64px rgba(0, 0, 0, 0.5);
	}

	.editor-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 20px;
		border-bottom: 1px solid var(--border, #2f3146);
		flex-shrink: 0;
	}

	.editor-title {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.editor-kind {
		font-size: 11px;
		padding: 2px 8px;
		border-radius: 4px;
		background: rgba(122, 162, 247, 0.15);
		color: var(--accent, #7aa2f7);
		text-transform: uppercase;
		font-weight: 600;
	}

	.editor-name {
		font-size: 15px;
		font-weight: 600;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		color: var(--text, #c0caf5);
	}

	.editor-file {
		font-size: 13px;
		color: var(--text-subtle, #565f89);
	}

	.editor-actions {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.unsaved-indicator {
		font-size: 12px;
		color: #f59e0b;
		padding: 2px 8px;
		background: rgba(245, 158, 11, 0.1);
		border-radius: 4px;
	}

	.editor-btn {
		padding: 6px 14px;
		font-size: 13px;
		border: 1px solid var(--border, #2f3146);
		border-radius: 6px;
		background: var(--bg-base, #1a1b26);
		color: var(--text, #c0caf5);
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 6px;
		transition: all 0.15s;
	}

	.editor-btn:hover:not(:disabled) {
		border-color: var(--text-subtle, #565f89);
	}

	.editor-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.editor-btn.save {
		background: rgba(34, 197, 94, 0.15);
		border-color: rgba(34, 197, 94, 0.3);
		color: #22c55e;
	}

	.editor-btn.save:hover:not(:disabled) {
		background: rgba(34, 197, 94, 0.25);
		border-color: #22c55e;
	}

	.editor-btn kbd {
		font-size: 10px;
		padding: 1px 4px;
		border-radius: 3px;
		background: rgba(255, 255, 255, 0.06);
		color: var(--text-subtle, #565f89);
		font-family: inherit;
	}

	.editor-error {
		padding: 10px 20px;
		background: rgba(239, 68, 68, 0.08);
		color: #ef4444;
		font-size: 13px;
		border-bottom: 1px solid var(--border, #2f3146);
	}

	.editor-body {
		flex: 1;
		display: flex;
		min-height: 0;
		overflow: hidden;
		border-radius: 0 0 12px 12px;
	}

	.editor-loading {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		color: var(--text-muted, #7982a9);
	}

	.line-numbers {
		display: flex;
		flex-direction: column;
		padding: 16px 0;
		min-width: 52px;
		text-align: right;
		padding-right: 12px;
		color: var(--text-subtle, #565f89);
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 14px;
		line-height: 1.6;
		user-select: none;
		border-right: 1px solid var(--border, #2f3146);
		background: var(--bg-base, #1a1b26);
		overflow: hidden;
	}

	.line-numbers span {
		padding: 0 4px;
	}

	.code-area {
		flex: 1;
		position: relative;
		overflow: auto;
		background: var(--bg-base, #1a1b26);
		border-bottom-right-radius: 12px;
	}

	.highlight-layer {
		position: absolute;
		inset: 0;
		padding: 16px;
		margin: 0;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 14px;
		line-height: 1.6;
		white-space: pre;
		pointer-events: none;
		color: transparent;
		overflow: hidden;
	}

	.editor-textarea {
		position: relative;
		width: 100%;
		height: 100%;
		padding: 16px;
		background: transparent;
		color: var(--text, #c0caf5);
		border: none;
		outline: none;
		resize: none;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 14px;
		line-height: 1.6;
		tab-size: 2;
		white-space: pre;
		caret-color: var(--accent, #7aa2f7);
		/* Make text transparent so highlight layer shows through */
		color: transparent;
		-webkit-text-fill-color: transparent;
	}

	/* Fallback: if highlight layer doesn't work, show text */
	.editor-textarea:focus {
		color: transparent;
		-webkit-text-fill-color: transparent;
	}

	/* Syntax highlighting colors */
	:global(.hl-comment) {
		color: var(--text-subtle, #565f89);
		font-style: italic;
	}
	:global(.hl-string) {
		color: #9ece6a;
	}
	:global(.hl-keyword) {
		color: #bb9af7;
		font-weight: 600;
	}
	:global(.hl-bool) {
		color: #ff9e64;
	}
	:global(.hl-number) {
		color: #ff9e64;
	}
	:global(.hl-func) {
		color: #7aa2f7;
	}
	:global(.hl-type) {
		color: #2ac3de;
	}
	:global(.hl-ref) {
		color: #73daca;
	}
</style>

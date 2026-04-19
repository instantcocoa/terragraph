<script lang="ts">
	import { workspace } from '$lib/stores/workspace.svelte';
	import { theme } from '$lib/stores/theme.svelte';

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
	let editorContainer: HTMLDivElement | undefined = $state();
	let editorInstance: import('monaco-editor').editor.IStandaloneCodeEditor | undefined;
	let monacoModule: typeof import('monaco-editor') | undefined;

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
				const extracted = lines.slice(start, end).join('\n');
				content = extracted;
				originalContent = extracted;
				loading = false;

				// If editor already exists, update its value
				if (editorInstance) {
					editorInstance.setValue(extracted);
				}
			})
			.catch((e) => {
				error = e.message || 'Failed to load file';
				// Fallback to rawHCL
				content = node.rawHCL || '';
				originalContent = content;
				loading = false;

				if (editorInstance) {
					editorInstance.setValue(content);
				}
			});
	});

	function getMonacoThemeName(appTheme: string): string {
		return appTheme === 'light' ? 'hcl-light' : 'hcl-dark';
	}

	// Switch Monaco theme when app theme changes
	$effect(() => {
		const currentTheme = theme.current;
		if (monacoModule) {
			monacoModule.editor.setTheme(getMonacoThemeName(currentTheme));
		}
	});

	// Initialize Monaco editor
	$effect(() => {
		if (!editorContainer || loading) return;
		if (typeof window === 'undefined') return;

		let disposed = false;
		let editor: import('monaco-editor').editor.IStandaloneCodeEditor | undefined;

		(async () => {
			const monaco = await import('monaco-editor');
			if (disposed) return;
			monacoModule = monaco;

			// Register HCL language if not already registered
			const langRegistered = monaco.languages.getLanguages().some((l) => l.id === 'hcl');
			if (!langRegistered) {
				monaco.languages.register({ id: 'hcl' });
				monaco.languages.setMonarchTokensProvider('hcl', {
					keywords: [
						'resource', 'data', 'variable', 'output', 'locals', 'module',
						'provider', 'terraform', 'required_providers', 'for_each', 'count',
						'depends_on', 'lifecycle', 'dynamic'
					],
					typeKeywords: [
						'string', 'number', 'bool', 'list', 'map', 'set', 'object', 'tuple', 'any'
					],
					constants: ['true', 'false', 'null'],
					tokenizer: {
						root: [
							// Line comments
							[/#.*$/, 'comment'],
							[/\/\/.*$/, 'comment'],
							// Block comments
							[/\/\*/, 'comment', '@comment'],
							// Strings with interpolation
							[/"/, 'string', '@string'],
							// References: var.x, local.x, data.x, module.x, each.x, self.x
							[/\b(var|local|data|module|each|self)\./, 'variable.reference'],
							// Functions
							[/[a-z_]\w*(?=\s*\()/, 'function'],
							// Keywords
							[/\b[a-zA-Z_]\w*\b/, {
								cases: {
									'@keywords': 'keyword',
									'@typeKeywords': 'type',
									'@constants': 'constant',
									'@default': 'identifier'
								}
							}],
							// Numbers
							[/\d+(\.\d+)?/, 'number'],
							// Braces, brackets
							[/[{}()\[\]]/, 'delimiter.bracket'],
							// Operators
							[/[=!<>]=?|&&|\|\|/, 'operator'],
							// Whitespace
							[/\s+/, 'white']
						],
						comment: [
							[/[^/*]+/, 'comment'],
							[/\*\//, 'comment', '@pop'],
							[/./, 'comment']
						],
						string: [
							[/\$\{/, 'string.interpolation', '@interpolation'],
							[/[^"$\\]+/, 'string'],
							[/\\./, 'string.escape'],
							[/"/, 'string', '@pop']
						],
						interpolation: [
							[/\}/, 'string.interpolation', '@pop'],
							{ include: 'root' }
						]
					}
				});
			}

			// Define dark theme (Tokyo Night)
			monaco.editor.defineTheme('hcl-dark', {
				base: 'vs-dark',
				inherit: true,
				rules: [
					{ token: 'keyword', foreground: 'bb9af7', fontStyle: 'bold' },
					{ token: 'type', foreground: '2ac3de' },
					{ token: 'constant', foreground: 'ff9e64' },
					{ token: 'comment', foreground: '565f89', fontStyle: 'italic' },
					{ token: 'string', foreground: '9ece6a' },
					{ token: 'string.escape', foreground: '89ddff' },
					{ token: 'string.interpolation', foreground: '7aa2f7' },
					{ token: 'number', foreground: 'ff9e64' },
					{ token: 'variable.reference', foreground: '73daca' },
					{ token: 'function', foreground: '7aa2f7' },
					{ token: 'operator', foreground: '89ddff' },
					{ token: 'delimiter.bracket', foreground: 'a9b1d6' },
					{ token: 'identifier', foreground: 'c0caf5' }
				],
				colors: {
					'editor.background': '#1a1b26',
					'editor.foreground': '#c0caf5',
					'editor.lineHighlightBackground': '#1e2030',
					'editorLineNumber.foreground': '#565f89',
					'editorLineNumber.activeForeground': '#c0caf5',
					'editor.selectionBackground': '#33467c55',
					'editor.inactiveSelectionBackground': '#33467c33',
					'editorCursor.foreground': '#7aa2f7',
					'editorWhitespace.foreground': '#2f3146',
					'editorIndentGuide.background': '#2f314680',
					'editorIndentGuide.activeBackground': '#565f8980',
					'scrollbarSlider.background': '#2f314660',
					'scrollbarSlider.hoverBackground': '#2f314690',
					'scrollbarSlider.activeBackground': '#2f3146b0'
				}
			});

			// Define light theme
			monaco.editor.defineTheme('hcl-light', {
				base: 'vs',
				inherit: true,
				rules: [
					{ token: 'keyword', foreground: '7c3aed', fontStyle: 'bold' },
					{ token: 'type', foreground: '0891b2' },
					{ token: 'constant', foreground: 'ea580c' },
					{ token: 'comment', foreground: '94a3b8', fontStyle: 'italic' },
					{ token: 'string', foreground: '16a34a' },
					{ token: 'string.escape', foreground: '0284c7' },
					{ token: 'string.interpolation', foreground: '2563eb' },
					{ token: 'number', foreground: 'ea580c' },
					{ token: 'variable.reference', foreground: '0d9488' },
					{ token: 'function', foreground: '2563eb' },
					{ token: 'operator', foreground: '0284c7' },
					{ token: 'delimiter.bracket', foreground: '475569' },
					{ token: 'identifier', foreground: '1e293b' }
				],
				colors: {
					'editor.background': '#ffffff',
					'editor.foreground': '#1e293b',
					'editor.lineHighlightBackground': '#f1f5f9',
					'editorLineNumber.foreground': '#94a3b8',
					'editorLineNumber.activeForeground': '#1e293b',
					'editor.selectionBackground': '#3b82f633',
					'editor.inactiveSelectionBackground': '#3b82f622',
					'editorCursor.foreground': '#2563eb',
					'editorWhitespace.foreground': '#e2e8f0',
					'editorIndentGuide.background': '#e2e8f080',
					'editorIndentGuide.activeBackground': '#94a3b880',
					'scrollbarSlider.background': '#94a3b830',
					'scrollbarSlider.hoverBackground': '#94a3b850',
					'scrollbarSlider.activeBackground': '#94a3b870'
				}
			});

			editor = monaco.editor.create(editorContainer!, {
				value: content,
				language: 'hcl',
				theme: getMonacoThemeName(theme.current),
				minimap: { enabled: false },
				fontSize: 14,
				lineNumbers: 'on',
				scrollBeyondLastLine: false,
				wordWrap: 'on',
				tabSize: 2,
				automaticLayout: true,
				padding: { top: 16 },
				fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
				renderLineHighlight: 'line',
				overviewRulerLanes: 0,
				hideCursorInOverviewRuler: true,
				overviewRulerBorder: false,
				scrollbar: {
					verticalScrollbarSize: 8,
					horizontalScrollbarSize: 8
				}
			});

			editorInstance = editor;

			// Track dirty state on content change
			editor.onDidChangeModelContent(() => {
				const currentValue = editor!.getValue();
				content = currentValue;
				dirty = currentValue !== originalContent;
			});

			// Cmd+S to save
			editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
				handleSave();
			});

			// Escape to close
			editor.addCommand(monaco.KeyCode.Escape, () => {
				if (dirty) {
					if (confirm('Discard changes?')) onclose();
				} else {
					onclose();
				}
			});

			// Focus the editor
			editor.focus();
		})();

		return () => {
			disposed = true;
			if (editor) {
				editor.dispose();
				editorInstance = undefined;
			}
		};
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
				<div class="monaco-wrapper" bind:this={editorContainer}></div>
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

	.monaco-wrapper {
		width: 100%;
		height: 100%;
		border-radius: 0 0 12px 12px;
		overflow: hidden;
	}
</style>

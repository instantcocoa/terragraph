import type {
	GraphNode,
	GraphEdge,
	Diagnostic,
	PlanChange,
	PlanResult,
	ValidateResult,
	WorkspaceGraph,
	ProviderSchemas,
	ResourceSchema,
	AddBlockRequest
} from '$lib/types';
import * as api from '$lib/api';

class WorkspaceStore {
	// Core state
	path = $state('');
	nodes = $state<GraphNode[]>([]);
	edges = $state<GraphEdge[]>([]);
	files = $state<string[]>([]);
	diagnostics = $state<Diagnostic[]>([]);
	planChanges = $state<PlanChange[]>([]);
	planSummary = $state<PlanResult['summary'] | null>(null);

	// Schema state
	schemas = $state<ProviderSchemas | null>(null);
	schemaLoading = $state(false);
	schemaError = $state<string | null>(null);

	// UI state
	selectedNodeId = $state<string | null>(null);
	loading = $state(false);
	error = $state<string | null>(null);
	validating = $state(false);
	validateResult = $state<'pass' | 'fail' | null>(null);
	planning = $state(false);
	showPlanOverlay = $state(false);
	showAddDialog = $state(false);
	showHCLEditor = $state(false);
	connectingMode = $state(false);
	bottomPanelTab = $state<'diagnostics' | 'plan' | 'logs'>('diagnostics');

	// Derived
	selectedNode = $derived(this.nodes.find((n) => n.id === this.selectedNodeId) ?? null);

	selectedNodeSchema = $derived.by(() => {
		const node = this.selectedNode;
		if (!node || !this.schemas) return null;
		if (node.kind === 'resource' && node.resourceType) {
			return this.schemas.resources[node.resourceType] ?? null;
		}
		if (node.kind === 'data' && node.resourceType) {
			return this.schemas.dataSources[node.resourceType] ?? null;
		}
		return null;
	});

	selectedNodePlan = $derived(
		this.selectedNode
			? (this.planChanges.find((c) => c.address === this.selectedNode!.address) ?? null)
			: null
	);

	planChangeMap = $derived(new Map(this.planChanges.map((c) => [c.address, c])));

	// Map from address -> node id for fast lookups
	addressToNodeId = $derived(new Map(this.nodes.map((n) => [n.address, n.id])));

	// Incoming references: nodes that reference the selected node
	incomingRefs = $derived.by(() => {
		const node = this.selectedNode;
		if (!node) return [];

		const results: Array<{ nodeId: string; nodeName: string; nodeAddress: string; nodeKind: string; attribute: string }> = [];

		for (const other of this.nodes) {
			if (other.id === node.id) continue;
			for (const attr of other.attributes ?? []) {
				for (const ref of attr.references ?? []) {
					if (this.refMatchesNode(ref, node)) {
						results.push({
							nodeId: other.id,
							nodeName: other.name,
							nodeAddress: other.address,
							nodeKind: other.kind,
							attribute: attr.name
						});
					}
				}
			}
			for (const nb of other.nestedBlocks ?? []) {
				for (const attr of nb.attributes ?? []) {
					for (const ref of attr.references ?? []) {
						if (this.refMatchesNode(ref, node)) {
							results.push({
								nodeId: other.id,
								nodeName: other.name,
								nodeAddress: other.address,
								nodeKind: other.kind,
								attribute: `${nb.type}.${attr.name}`
							});
						}
					}
				}
			}
			for (const dep of other.dependsOn ?? []) {
				if (this.refMatchesNode(dep, node)) {
					results.push({
						nodeId: other.id,
						nodeName: other.name,
						nodeAddress: other.address,
						nodeKind: other.kind,
						attribute: 'depends_on'
					});
				}
			}
		}
		return results;
	});

	private refMatchesNode(ref: string, node: GraphNode): boolean {
		// Direct match on address
		if (ref === node.address) return true;
		// Reference with attribute access: "aws_instance.web.id" matches "aws_instance.web"
		if (ref.startsWith(node.address + '.')) return true;
		return false;
	}

	// Resolve a reference string to a node id (for click-through)
	resolveRef(ref: string): string | null {
		// Try direct
		const direct = this.addressToNodeId.get(ref);
		if (direct) return direct;
		// Try trimming attribute access
		const parts = ref.split('.');
		for (let i = parts.length; i >= 2; i--) {
			const candidate = parts.slice(0, i).join('.');
			const id = this.addressToNodeId.get(candidate);
			if (id) return id;
		}
		return null;
	}

	// All resource types available from schema for the add-block palette
	availableResourceTypes = $derived.by(() => {
		if (!this.schemas) return [];
		return Object.keys(this.schemas.resources).sort();
	});

	availableDataSourceTypes = $derived.by(() => {
		if (!this.schemas) return [];
		return Object.keys(this.schemas.dataSources).sort();
	});

	// Providers extracted from resource type prefixes
	availableProviders = $derived.by(() => {
		if (!this.schemas) return [];
		const providers = new Set<string>();
		for (const key of Object.keys(this.schemas.resources)) {
			const prefix = key.split('_')[0];
			if (prefix) providers.add(prefix);
		}
		return [...providers].sort();
	});

	// Resource types grouped by provider prefix
	resourceTypesByProvider(provider: string): string[] {
		if (!this.schemas) return [];
		return Object.keys(this.schemas.resources)
			.filter((t) => t.startsWith(provider + '_'))
			.sort();
	}

	dataSourceTypesByProvider(provider: string): string[] {
		if (!this.schemas) return [];
		return Object.keys(this.schemas.dataSources)
			.filter((t) => t.startsWith(provider + '_'))
			.sort();
	}

	// For connection mode: which existing nodes could be referenced
	connectableNodes = $derived.by(() => {
		const node = this.selectedNode;
		if (!node || !this.connectingMode) return [];
		// Resources, data sources, variables, locals, and modules can all be referenced
		return this.nodes.filter(
			(n) =>
				n.id !== node.id &&
				['resource', 'data', 'variable', 'local', 'module'].includes(n.kind)
		);
	});

	async load(workspacePath: string) {
		this.loading = true;
		this.error = null;
		this.path = workspacePath;

		try {
			const result: WorkspaceGraph = await api.loadWorkspace(workspacePath);
			this.nodes = result.nodes ?? [];
			this.edges = result.edges ?? [];
			this.files = result.files ?? [];
			this.diagnostics = result.diagnostics ?? [];
			this.planChanges = [];
			this.planSummary = null;
			this.showPlanOverlay = false;
			// Reset schema when loading a new/different workspace
			this.schemas = null;
			this.schemaError = null;
			// Auto-load schema in the background
			this.loadSchema();
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Failed to load workspace';
		} finally {
			this.loading = false;
		}
	}

	async loadSchema() {
		if (!this.path || this.schemas || this.schemaLoading) return;
		this.schemaLoading = true;
		this.schemaError = null;
		try {
			this.schemas = await api.getSchema(this.path);
		} catch (e) {
			this.schemaError =
				e instanceof Error ? e.message : 'Failed to load provider schema';
		} finally {
			this.schemaLoading = false;
		}
	}

	async validate() {
		if (!this.path) return;
		this.validating = true;
		this.validateResult = null;
		try {
			const result: ValidateResult = await api.validateWorkspace(this.path);
			this.diagnostics = result.diagnostics ?? [];
			this.bottomPanelTab = 'diagnostics';
			this.validateResult = result.valid ? 'pass' : 'fail';
			// Auto-clear success indicator after 5s
			if (result.valid) {
				setTimeout(() => { if (this.validateResult === 'pass') this.validateResult = null; }, 5000);
			}
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Validation failed';
			this.validateResult = 'fail';
		} finally {
			this.validating = false;
		}
	}

	planError = $state<string | null>(null);
	planRawOutput = $state<string | null>(null);

	async plan() {
		if (!this.path) return;
		this.planning = true;
		this.planError = null;
		this.planRawOutput = null;
		this.bottomPanelTab = 'plan';
		try {
			const result: PlanResult = await api.planWorkspace(this.path);
			this.planChanges = result.changes ?? [];
			this.planSummary = result.summary;
			this.planRawOutput = result.rawOutput ?? null;
			if (result.planError) {
				this.planError = result.planError;
			} else {
				this.showPlanOverlay = true;
			}
		} catch (e) {
			this.planError = e instanceof Error ? e.message : 'Plan failed';
		} finally {
			this.planning = false;
		}
	}

	selectNode(id: string | null) {
		this.selectedNodeId = id;
		this.connectingMode = false;
		this.evalResult = null;
		this.evalError = null;
	}

	async patchAttribute(attribute: string, value: string) {
		const node = this.selectedNode;
		if (!node || !this.path) return;

		try {
			await api.patchAttribute(this.path, node.source.file, node.address, attribute, value);
			await this.load(this.path);
			this.selectedNodeId = node.id;
			await this.refreshHistory();
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Patch failed';
		}
	}

	async addBlock(req: Omit<AddBlockRequest, 'workspacePath'>, selectAfter = false) {
		if (!this.path) return;
		try {
			await api.addBlock({ ...req, workspacePath: this.path });
			await this.load(this.path);
			await this.refreshHistory();
			if (selectAfter && req.name) {
				// Find the newly created node and select it
				const address = req.blockType === 'data'
					? `data.${req.resourceType}.${req.name}`
					: req.blockType === 'variable'
						? `var.${req.name}`
						: req.blockType === 'output'
							? `output.${req.name}`
							: `${req.resourceType}.${req.name}`;
				const newNode = this.nodes.find((n) => n.address === address);
				if (newNode) this.selectedNodeId = newNode.id;
			}
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Failed to add block';
		}
	}

	async removeBlock(address: string, file: string) {
		if (!this.path) return;
		try {
			await api.removeBlock({ workspacePath: this.path, file, address });
			this.selectedNodeId = null;
			await this.load(this.path);
			await this.refreshHistory();
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Failed to remove block';
		}
	}

	getSchemaForType(kind: 'resource' | 'data', resourceType: string): ResourceSchema | null {
		if (!this.schemas) return null;
		if (kind === 'resource') return this.schemas.resources[resourceType] ?? null;
		if (kind === 'data') return this.schemas.dataSources[resourceType] ?? null;
		return null;
	}

	// Undo/redo
	undoCount = $state(0);
	redoCount = $state(0);

	async undo() {
		if (!this.path) return;
		try {
			const result = await api.undo(this.path);
			this.undoCount = result.undoCount;
			this.redoCount = result.redoCount;
			const selectedId = this.selectedNodeId;
			await this.load(this.path);
			this.selectedNodeId = selectedId;
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Undo failed';
		}
	}

	async redo() {
		if (!this.path) return;
		try {
			const result = await api.redo(this.path);
			this.undoCount = result.undoCount;
			this.redoCount = result.redoCount;
			const selectedId = this.selectedNodeId;
			await this.load(this.path);
			this.selectedNodeId = selectedId;
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Redo failed';
		}
	}

	async refreshHistory() {
		if (!this.path) return;
		try {
			const status = await api.getHistory(this.path);
			this.undoCount = status.undoCount;
			this.redoCount = status.redoCount;
		} catch {
			// Ignore
		}
	}

	// Update history counts after mutations
	private async afterMutation(reselect?: string | null) {
		await this.refreshHistory();
		const p = this.path;
		await this.load(p);
		if (reselect) this.selectedNodeId = reselect;
	}

	async renameBlock(newName: string) {
		const node = this.selectedNode;
		if (!node || !this.path) return;
		try {
			await api.renameBlock(this.path, node.source.file, node.address, newName);
			await this.load(this.path);
			await this.refreshHistory();
			// Find the renamed node
			const newAddr = node.kind === 'data'
				? `data.${node.resourceType}.${newName}`
				: node.kind === 'variable'
					? `var.${newName}`
					: node.kind === 'output'
						? `output.${newName}`
						: `${node.resourceType}.${newName}`;
			const renamed = this.nodes.find((n) => n.address === newAddr);
			if (renamed) this.selectedNodeId = renamed.id;
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Rename failed';
		}
	}

	async addNestedBlock(blockType: string, attributes?: Record<string, string>) {
		const node = this.selectedNode;
		if (!node || !this.path) return;
		try {
			await api.addNestedBlock(this.path, node.source.file, node.address, blockType, attributes);
			await this.load(this.path);
			this.selectedNodeId = node.id;
			await this.refreshHistory();
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Failed to add nested block';
		}
	}

	async removeNestedBlock(blockType: string, index: number) {
		const node = this.selectedNode;
		if (!node || !this.path) return;
		try {
			await api.removeNestedBlock(this.path, node.source.file, node.address, blockType, index);
			await this.load(this.path);
			this.selectedNodeId = node.id;
			await this.refreshHistory();
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Failed to remove nested block';
		}
	}

	async removeAttribute(attribute: string) {
		const node = this.selectedNode;
		if (!node || !this.path) return;
		try {
			await api.removeAttribute(this.path, node.source.file, node.address, attribute);
			await this.load(this.path);
			this.selectedNodeId = node.id;
			await this.refreshHistory();
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Failed to remove attribute';
		}
	}

	async addProvider(provider: string, attributes?: Record<string, string>) {
		if (!this.path) return;
		try {
			const result = await api.addProvider(this.path, provider, undefined, undefined, undefined, attributes);
			// Reload workspace and schema (schema cache was invalidated server-side)
			this.schemas = null;
			this.schemaError = null;
			await this.load(this.path);
			// Find the provider node and select it
			const providerNode = this.nodes.find((n) => n.kind === 'provider' && n.name === provider);
			if (providerNode) this.selectedNodeId = providerNode.id;
			if (result.initError) {
				this.error = `Provider added but init failed: ${result.initError}`;
			}
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Failed to add provider';
		}
	}

	// Data source evaluation
	evalResult = $state<Record<string, unknown> | null>(null);
	evalError = $state<string | null>(null);
	evaluating = $state(false);

	async evalData(address: string) {
		if (!this.path) return;
		this.evaluating = true;
		this.evalResult = null;
		this.evalError = null;
		try {
			const result = await api.evalData(this.path, address);
			if (result.valid && result.values) {
				this.evalResult = result.values;
			} else {
				this.evalError = result.error ?? 'Evaluation returned no values';
			}
		} catch (e) {
			this.evalError = e instanceof Error ? e.message : 'Evaluation failed';
		} finally {
			this.evaluating = false;
		}
	}

	// New project
	async initProject(path: string, provider: string, region?: string) {
		try {
			await api.initProject(path, provider, region);
			await this.load(path);
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Failed to create project';
		}
	}
}

export const workspace = new WorkspaceStore();

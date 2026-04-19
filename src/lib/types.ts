// Types matching the Go backend API responses

export type NodeKind =
	| 'resource'
	| 'data'
	| 'module'
	| 'provider'
	| 'variable'
	| 'local'
	| 'output'
	| 'terraform';

export type EdgeKind = 'reference' | 'depends_on' | 'provider' | 'module' | 'contains';

export type PlanAction = 'create' | 'update' | 'delete' | 'replace' | 'no-op' | 'read';

export interface SourceSpan {
	file: string;
	startLine: number;
	endLine: number;
	startCol?: number;
	endCol?: number;
}

export interface Attribute {
	name: string;
	value?: unknown;
	expression?: string;
	isComputed?: boolean;
	type?: string;
	references?: string[];
}

export interface NestedBlock {
	type: string;
	labels?: string[];
	attributes?: Attribute[];
	rawHCL?: string;
}

export interface GraphNode {
	id: string;
	kind: NodeKind;
	resourceType?: string;
	name: string;
	address: string;
	provider?: string;
	source: SourceSpan;
	attributes?: Attribute[];
	nestedBlocks?: NestedBlock[];
	rawHCL?: string;
	dependsOn?: string[];
	default?: unknown;
	description?: string;
	varType?: string;
	moduleSource?: string;
	moduleVersion?: string;
}

export interface GraphEdge {
	id: string;
	source: string;
	target: string;
	kind: EdgeKind;
	label?: string;
}

export interface Diagnostic {
	severity: string;
	summary: string;
	detail?: string;
	range?: SourceSpan;
	nodeId?: string;
}

export interface PlanChange {
	address: string;
	action: PlanAction;
	before?: Record<string, unknown>;
	after?: Record<string, unknown>;
	afterUnknown?: Record<string, unknown>;
}

export interface WorkspaceGraph {
	nodes: GraphNode[];
	edges: GraphEdge[];
	diagnostics?: Diagnostic[];
	files: string[];
}

export interface ValidateResult {
	valid: boolean;
	diagnostics: Diagnostic[];
	errorCount: number;
	warningCount: number;
}

export interface PlanResult {
	changes: PlanChange[];
	summary: {
		create: number;
		update: number;
		delete: number;
		replace: number;
	};
	rawOutput?: string;
	planError?: string;
}

// Provider schema types
export interface SchemaAttribute {
	name: string;
	type: unknown; // "string", "number", "bool", ["list","string"], ["map","string"], etc.
	required: boolean;
	optional: boolean;
	computed: boolean;
	description?: string;
	sensitive?: boolean;
}

export interface SchemaBlockType {
	name: string;
	nestingMode: string; // "list", "set", "single", "map"
	attributes?: SchemaAttribute[];
	blockTypes?: SchemaBlockType[];
	minItems?: number;
	maxItems?: number;
}

export interface ResourceSchema {
	name: string;
	provider: string;
	attributes: SchemaAttribute[];
	blockTypes?: SchemaBlockType[];
}

export interface ProviderSchemas {
	resources: Record<string, ResourceSchema>;
	dataSources: Record<string, ResourceSchema>;
}

// Add/Remove block types
export interface AddBlockRequest {
	workspacePath: string;
	file: string;
	blockType: 'resource' | 'data' | 'variable' | 'output';
	resourceType?: string;
	name: string;
	attributes?: Record<string, string>;
}

export interface RemoveBlockRequest {
	workspacePath: string;
	file: string;
	address: string;
}

// Node kind display config
export const NODE_KIND_CONFIG: Record<
	NodeKind,
	{ label: string; color: string; bgColor: string; icon: string }
> = {
	resource: {
		label: 'Resource',
		color: '#3b82f6',
		bgColor: '#1e3a5f',
		icon: '□'
	},
	data: {
		label: 'Data Source',
		color: '#8b5cf6',
		bgColor: '#3b1f6e',
		icon: '◇'
	},
	module: {
		label: 'Module',
		color: '#f59e0b',
		bgColor: '#5c3d0e',
		icon: '◻'
	},
	provider: {
		label: 'Provider',
		color: '#6366f1',
		bgColor: '#2d2f6e',
		icon: '⬡'
	},
	variable: {
		label: 'Variable',
		color: '#10b981',
		bgColor: '#0d4f3c',
		icon: '▸'
	},
	local: {
		label: 'Local',
		color: '#06b6d4',
		bgColor: '#0c4a5e',
		icon: '◆'
	},
	output: {
		label: 'Output',
		color: '#f43f5e',
		bgColor: '#5c1a2a',
		icon: '▹'
	},
	terraform: {
		label: 'Terraform',
		color: '#71717a',
		bgColor: '#2d2d30',
		icon: '⚙'
	}
};

export const PLAN_ACTION_CONFIG: Record<
	PlanAction,
	{ label: string; color: string; icon: string }
> = {
	create: { label: 'Create', color: '#22c55e', icon: '+' },
	update: { label: 'Update', color: '#f59e0b', icon: '~' },
	delete: { label: 'Delete', color: '#ef4444', icon: '-' },
	replace: { label: 'Replace', color: '#f97316', icon: '±' },
	'no-op': { label: 'No Change', color: '#71717a', icon: '=' },
	read: { label: 'Read', color: '#8b5cf6', icon: '?' }
};

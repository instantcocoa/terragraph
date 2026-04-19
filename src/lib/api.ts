import type {
	WorkspaceGraph,
	ValidateResult,
	PlanResult,
	ProviderSchemas,
	AddBlockRequest,
	RemoveBlockRequest
} from './types';

const API_BASE = '/api';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const res = await fetch(`${API_BASE}${path}`, {
		headers: { 'Content-Type': 'application/json' },
		...options
	});
	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(body.error || `API error: ${res.status}`);
	}
	return res.json();
}

export async function loadWorkspace(path: string): Promise<WorkspaceGraph> {
	return request<WorkspaceGraph>('/workspace/load', {
		method: 'POST',
		body: JSON.stringify({ path })
	});
}

export async function validateWorkspace(path: string): Promise<ValidateResult> {
	return request<ValidateResult>('/workspace/validate', {
		method: 'POST',
		body: JSON.stringify({ path })
	});
}

export async function planWorkspace(path: string): Promise<PlanResult> {
	return request<PlanResult>('/workspace/plan', {
		method: 'POST',
		body: JSON.stringify({ path })
	});
}

export async function getFile(path: string): Promise<{ path: string; content: string }> {
	return request(`/workspace/file?path=${encodeURIComponent(path)}`);
}

export async function patchAttribute(
	workspacePath: string,
	file: string,
	address: string,
	attribute: string,
	value: string
): Promise<{ file: string; content: string }> {
	return request('/workspace/patch', {
		method: 'POST',
		body: JSON.stringify({ workspacePath, file, address, attribute, value })
	});
}

export async function getSchema(path: string): Promise<ProviderSchemas> {
	return request<ProviderSchemas>('/workspace/schema', {
		method: 'POST',
		body: JSON.stringify({ path })
	});
}

export async function addBlock(req: AddBlockRequest): Promise<{ file: string; content: string }> {
	return request('/workspace/add-block', {
		method: 'POST',
		body: JSON.stringify(req)
	});
}

export async function removeBlock(
	req: RemoveBlockRequest
): Promise<{ file: string; content: string }> {
	return request('/workspace/remove-block', {
		method: 'POST',
		body: JSON.stringify(req)
	});
}

export async function undo(
	path: string
): Promise<{ file: string; undoCount: number; redoCount: number }> {
	return request('/workspace/undo', {
		method: 'POST',
		body: JSON.stringify({ path })
	});
}

export async function redo(
	path: string
): Promise<{ file: string; undoCount: number; redoCount: number }> {
	return request('/workspace/redo', {
		method: 'POST',
		body: JSON.stringify({ path })
	});
}

export async function getHistory(
	path: string
): Promise<{ undoCount: number; redoCount: number }> {
	return request('/workspace/history', {
		method: 'POST',
		body: JSON.stringify({ path })
	});
}

export interface EvalDataResult {
	address: string;
	values?: Record<string, unknown>;
	valid: boolean;
	error?: string;
	rawOutput?: string;
}

export async function evalData(
	path: string,
	address: string
): Promise<EvalDataResult> {
	return request('/workspace/eval-data', {
		method: 'POST',
		body: JSON.stringify({ path, address })
	});
}

export interface ScaffoldDependency {
	blockType: string;
	resourceType: string;
	name: string;
	attributes: Record<string, string>;
	linkAttr: string;
	linkExpr: string;
	reason: string;
	required: boolean;
}

export interface ScaffoldResult {
	resourceType: string;
	attributes: Record<string, string>;
	dependencies: ScaffoldDependency[];
}

export async function getScaffold(resourceType: string): Promise<ScaffoldResult> {
	return request('/workspace/scaffold', {
		method: 'POST',
		body: JSON.stringify({ resourceType })
	});
}

export async function renameBlock(
	workspacePath: string,
	file: string,
	address: string,
	newName: string
): Promise<{ file: string; content: string }> {
	return request('/workspace/rename-block', {
		method: 'POST',
		body: JSON.stringify({ workspacePath, file, address, newName })
	});
}

export async function addNestedBlock(
	workspacePath: string,
	file: string,
	address: string,
	blockType: string,
	attributes?: Record<string, string>
): Promise<{ file: string; content: string }> {
	return request('/workspace/add-nested-block', {
		method: 'POST',
		body: JSON.stringify({ workspacePath, file, address, blockType, attributes })
	});
}

export async function removeNestedBlock(
	workspacePath: string,
	file: string,
	address: string,
	blockType: string,
	index: number
): Promise<{ file: string; content: string }> {
	return request('/workspace/remove-nested-block', {
		method: 'POST',
		body: JSON.stringify({ workspacePath, file, address, blockType, index })
	});
}

export async function removeAttribute(
	workspacePath: string,
	file: string,
	address: string,
	attribute: string
): Promise<{ file: string; content: string }> {
	return request('/workspace/remove-attribute', {
		method: 'POST',
		body: JSON.stringify({ workspacePath, file, address, attribute })
	});
}

export async function initProject(
	path: string,
	provider: string,
	region?: string
): Promise<{ path: string; file: string }> {
	return request('/workspace/init-project', {
		method: 'POST',
		body: JSON.stringify({ path, provider, region })
	});
}

export async function addProvider(
	workspacePath: string,
	provider: string,
	file?: string,
	source?: string,
	version?: string,
	attributes?: Record<string, string>
): Promise<{ file: string; content: string; initError?: string }> {
	return request('/workspace/add-provider', {
		method: 'POST',
		body: JSON.stringify({ workspacePath, provider, file, source, version, attributes })
	});
}

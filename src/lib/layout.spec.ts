import { describe, it, expect } from 'vitest';
import { layoutGraph, layoutGraphAsync } from './layout';

describe('layoutGraph (sync fallback)', () => {
	it('returns positions for all nodes', () => {
		const nodes = [
			{ id: 'a', kind: 'resource', name: 'a' },
			{ id: 'b', kind: 'variable', name: 'b' },
			{ id: 'c', kind: 'output', name: 'c' }
		];
		const positions = layoutGraph(nodes, []);
		expect(positions.size).toBe(3);
	});

	it('places resources above outputs', () => {
		const nodes = [
			{ id: 'res', kind: 'resource', name: 'web' },
			{ id: 'out', kind: 'output', name: 'id' }
		];
		const positions = layoutGraph(nodes, []);
		expect(positions.get('res')!.y).toBeLessThan(positions.get('out')!.y);
	});

	it('places providers to the left', () => {
		const nodes = [
			{ id: 'prov', kind: 'provider', name: 'aws' },
			{ id: 'res', kind: 'resource', name: 'web' }
		];
		const positions = layoutGraph(nodes, []);
		expect(positions.get('prov')!.x).toBeLessThan(positions.get('res')!.x);
	});

	it('handles empty graph', () => {
		expect(layoutGraph([], []).size).toBe(0);
	});
});

describe('layoutGraphAsync (Graphviz)', () => {
	it('returns positions for all nodes', async () => {
		const nodes = [
			{ id: 'res.a', kind: 'resource', name: 'a' },
			{ id: 'var.b', kind: 'variable', name: 'b' },
			{ id: 'output.c', kind: 'output', name: 'c' }
		];
		const edges = [
			{ source: 'res.a', target: 'var.b' },
			{ source: 'output.c', target: 'res.a' }
		];
		const positions = await layoutGraphAsync(nodes, edges);
		expect(positions.size).toBe(3);
	});
});

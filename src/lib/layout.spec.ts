import { describe, it, expect } from 'vitest';
import { layoutGraph, layoutGraphAsync } from './layout';

describe('layoutGraph (sync)', () => {
	it('returns positions for all nodes', () => {
		const nodes = [
			{ id: 'a', kind: 'resource', name: 'a' },
			{ id: 'b', kind: 'resource', name: 'b' },
			{ id: 'c', kind: 'output', name: 'c' }
		];
		const positions = layoutGraph(nodes, []);
		expect(positions.size).toBe(3);
	});

	it('places resources above outputs', () => {
		const nodes = [
			{ id: 'out', kind: 'output', name: 'id' },
			{ id: 'res', kind: 'resource', name: 'web' }
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

describe('layoutGraphAsync (ELK)', () => {
	it('returns positions for all nodes', async () => {
		const nodes = [
			{ id: 'a', kind: 'resource', name: 'a' },
			{ id: 'b', kind: 'variable', name: 'b' },
			{ id: 'c', kind: 'output', name: 'c' }
		];
		const edges = [{ source: 'a', target: 'b' }, { source: 'c', target: 'a' }];
		const positions = await layoutGraphAsync(nodes, edges);
		expect(positions.size).toBe(3);
	});

	it('is deterministic', async () => {
		const nodes = [
			{ id: 'a', kind: 'resource', name: 'alpha' },
			{ id: 'b', kind: 'resource', name: 'beta' },
			{ id: 'c', kind: 'variable', name: 'region' }
		];
		const edges = [{ source: 'a', target: 'c' }];
		const pos1 = await layoutGraphAsync(nodes, edges);
		const pos2 = await layoutGraphAsync(nodes, edges);
		for (const node of nodes) {
			expect(pos1.get(node.id)!.x).toBe(pos2.get(node.id)!.x);
			expect(pos1.get(node.id)!.y).toBe(pos2.get(node.id)!.y);
		}
	});
});

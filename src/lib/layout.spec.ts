import { describe, it, expect } from 'vitest';
import { layoutGraph } from './layout';

describe('layoutGraph', () => {
	it('returns positions for all nodes', () => {
		const nodes = [
			{ id: 'a', kind: 'resource', name: 'a' },
			{ id: 'b', kind: 'resource', name: 'b' },
			{ id: 'c', kind: 'output', name: 'c' }
		];
		const edges = [
			{ source: 'a', target: 'b' },
			{ source: 'b', target: 'c' }
		];

		const positions = layoutGraph(nodes, edges);

		expect(positions.size).toBe(3);
		expect(positions.has('a')).toBe(true);
		expect(positions.has('b')).toBe(true);
		expect(positions.has('c')).toBe(true);
	});

	it('places variables above resources above outputs', () => {
		const nodes = [
			{ id: 'out', kind: 'output', name: 'id' },
			{ id: 'res', kind: 'resource', name: 'web' },
			{ id: 'var', kind: 'variable', name: 'region' }
		];
		const edges = [
			{ source: 'res', target: 'var' },
			{ source: 'out', target: 'res' }
		];

		const positions = layoutGraph(nodes, edges);

		const varY = positions.get('var')!.y;
		const resY = positions.get('res')!.y;
		const outY = positions.get('out')!.y;

		expect(varY).toBeLessThan(resY);
		expect(resY).toBeLessThan(outY);
	});

	it('is deterministic - same input gives same output', () => {
		const nodes = [
			{ id: 'a', kind: 'resource', name: 'alpha' },
			{ id: 'b', kind: 'resource', name: 'beta' },
			{ id: 'c', kind: 'variable', name: 'region' }
		];
		const edges = [{ source: 'a', target: 'c' }];

		const pos1 = layoutGraph(nodes, edges);
		const pos2 = layoutGraph(nodes, edges);

		for (const node of nodes) {
			expect(pos1.get(node.id)!.x).toBe(pos2.get(node.id)!.x);
			expect(pos1.get(node.id)!.y).toBe(pos2.get(node.id)!.y);
		}
	});

	it('handles empty graph', () => {
		const positions = layoutGraph([], []);
		expect(positions.size).toBe(0);
	});
});

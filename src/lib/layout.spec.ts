import { describe, it, expect } from 'vitest';
import { layoutGraph } from './layout';

describe('layoutGraph', () => {
	it('returns positions for all nodes', () => {
		const nodes = [
			{ id: 'a', kind: 'resource', name: 'a' },
			{ id: 'b', kind: 'resource', name: 'b' },
			{ id: 'c', kind: 'output', name: 'c' }
		];
		const edges = [{ source: 'a', target: 'c' }];

		const positions = layoutGraph(nodes, edges);
		expect(positions.size).toBe(3);
	});

	it('places resources above data above variables above outputs', () => {
		const nodes = [
			{ id: 'out', kind: 'output', name: 'id' },
			{ id: 'res', kind: 'resource', name: 'web' },
			{ id: 'dat', kind: 'data', name: 'ami' },
			{ id: 'var', kind: 'variable', name: 'region' }
		];
		const edges = [
			{ source: 'res', target: 'dat' },
			{ source: 'out', target: 'res' }
		];

		const positions = layoutGraph(nodes, edges);

		const resY = positions.get('res')!.y;
		const datY = positions.get('dat')!.y;
		const varY = positions.get('var')!.y;
		const outY = positions.get('out')!.y;

		expect(resY).toBeLessThan(datY);
		expect(datY).toBeLessThan(varY);
		expect(varY).toBeLessThan(outY);
	});

	it('places providers to the left of main graph', () => {
		const nodes = [
			{ id: 'prov', kind: 'provider', name: 'aws' },
			{ id: 'res', kind: 'resource', name: 'web' },
			{ id: 'var', kind: 'variable', name: 'region' }
		];
		const edges: Array<{ source: string; target: string }> = [];

		const positions = layoutGraph(nodes, edges);

		const provX = positions.get('prov')!.x;
		const resX = positions.get('res')!.x;

		expect(provX).toBeLessThan(resX);
	});

	it('is deterministic', () => {
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
		expect(layoutGraph([], []).size).toBe(0);
	});
});

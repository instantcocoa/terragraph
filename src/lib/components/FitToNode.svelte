<script lang="ts">
	import { useSvelteFlow } from '@xyflow/svelte';
	import { workspace } from '$lib/stores/workspace.svelte';

	const { fitView } = useSvelteFlow();

	let lastSelectedId = $state<string | null>(null);

	$effect(() => {
		const id = workspace.selectedNodeId;
		if (id && id !== lastSelectedId) {
			lastSelectedId = id;
			// Zoom to fit the selected node with some padding
			fitView({
				nodes: [{ id }],
				duration: 300,
				padding: 0.5,
				maxZoom: 1.5
			});
		}
	});
</script>

<script lang="ts">
	import { workspace } from '$lib/stores/workspace.svelte';

	let { collapsed = false }: { collapsed?: boolean } = $props();
</script>

<div class="bottom-panel" class:collapsed>
	<div class="panel-tabs">
		<button
			class:active={workspace.bottomPanelTab === 'diagnostics'}
			onclick={() => (workspace.bottomPanelTab = 'diagnostics')}
		>
			Diagnostics
			{#if workspace.diagnostics.length > 0}
				<span class="badge error">{workspace.diagnostics.length}</span>
			{/if}
		</button>
		<button
			class:active={workspace.bottomPanelTab === 'plan'}
			onclick={() => (workspace.bottomPanelTab = 'plan')}
		>
			Plan
			{#if workspace.planSummary}
				<span class="badge info">
					{workspace.planSummary.create + workspace.planSummary.update + workspace.planSummary.delete + workspace.planSummary.replace}
				</span>
			{/if}
		</button>
	</div>

	{#if !collapsed}
		<div class="panel-content">
			{#if workspace.bottomPanelTab === 'diagnostics'}
				{#if workspace.validating}
					<div class="validate-status running">Running terraform validate...</div>
				{:else if workspace.validateResult === 'pass' && workspace.diagnostics.length === 0}
					<div class="validate-status pass">Configuration is valid - no issues found</div>
				{:else if workspace.validateResult === 'fail' || workspace.diagnostics.length > 0}
					<div class="validate-status fail">
						{workspace.diagnostics.filter(d => d.severity === 'error').length} error{workspace.diagnostics.filter(d => d.severity === 'error').length !== 1 ? 's' : ''}, {workspace.diagnostics.filter(d => d.severity === 'warning').length} warning{workspace.diagnostics.filter(d => d.severity === 'warning').length !== 1 ? 's' : ''}
					</div>
				{:else if workspace.diagnostics.length === 0}
					<div class="empty">Click "Validate" to check your configuration</div>
				{/if}
				{#if workspace.diagnostics.length > 0}
					<div class="diag-list">
						{#each workspace.diagnostics as diag}
							<div class="diag-item" class:error={diag.severity === 'error'} class:warning={diag.severity === 'warning'}>
								<span class="diag-severity">{diag.severity === 'error' ? 'E' : 'W'}</span>
								<div class="diag-content">
									<div class="diag-summary">{diag.summary}</div>
									{#if diag.detail}
										<div class="diag-detail">{diag.detail}</div>
									{/if}
									{#if diag.range}
										<div class="diag-location">{diag.range.file}:{diag.range.startLine}</div>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				{/if}

			{:else if workspace.bottomPanelTab === 'plan'}
				{#if workspace.planSummary}
					<div class="plan-summary">
						{#if workspace.planSummary.create > 0}
							<span class="plan-stat create">+{workspace.planSummary.create} create</span>
						{/if}
						{#if workspace.planSummary.update > 0}
							<span class="plan-stat update">~{workspace.planSummary.update} update</span>
						{/if}
						{#if workspace.planSummary.delete > 0}
							<span class="plan-stat delete">-{workspace.planSummary.delete} delete</span>
						{/if}
						{#if workspace.planSummary.replace > 0}
							<span class="plan-stat replace">+/-{workspace.planSummary.replace} replace</span>
						{/if}
					</div>
					{#if workspace.planChanges.length > 0}
						<div class="plan-list">
							{#each workspace.planChanges as change}
								{#if change.action !== 'no-op'}
									<details class="plan-item-details">
										<summary class="plan-item">
											<span class="plan-action {change.action}">{change.action}</span>
											<span class="plan-address">{change.address}</span>
											{#if change.after}
												<span class="plan-field-count">{Object.keys(change.after).length} fields</span>
											{/if}
										</summary>
										<div class="plan-item-body">
											{#if change.action === 'create' && change.after}
												<div class="plan-detail-header">Will be created with:</div>
												{#each Object.entries(change.after).sort(([a], [b]) => a.localeCompare(b)) as [key, value]}
													{#if value !== null && JSON.stringify(value) !== '[]' && JSON.stringify(value) !== '{}'}
														<div class="plan-field">
															<span class="plan-field-key">+ {key}</span>
															<span class="plan-field-value">{typeof value === 'object' ? JSON.stringify(value) : String(value)}</span>
														</div>
													{/if}
												{/each}
												{#if change.afterUnknown}
													{#each Object.entries(change.afterUnknown) as [key, val]}
														{#if val === true}
															<div class="plan-field unknown">
																<span class="plan-field-key">+ {key}</span>
																<span class="plan-field-value">(known after apply)</span>
															</div>
														{/if}
													{/each}
												{/if}
											{:else if change.action === 'update' && change.before && change.after}
												<div class="plan-detail-header">Changes:</div>
												{#each Object.keys(change.after).sort() as key}
													{@const before = change.before[key]}
													{@const after = change.after[key]}
													{#if JSON.stringify(before) !== JSON.stringify(after)}
														<div class="plan-field changed">
															<span class="plan-field-key">~ {key}</span>
															{#if before !== undefined}
																<span class="plan-field-before">{typeof before === 'object' ? JSON.stringify(before) : String(before)}</span>
															{/if}
															<span class="plan-field-arrow">-></span>
															<span class="plan-field-after">{typeof after === 'object' ? JSON.stringify(after) : String(after)}</span>
														</div>
													{/if}
												{/each}
											{:else if change.action === 'delete' && change.before}
												<div class="plan-detail-header">Will be destroyed:</div>
												{#each Object.entries(change.before).sort(([a], [b]) => a.localeCompare(b)).slice(0, 10) as [key, value]}
													{#if value !== null}
														<div class="plan-field delete">
															<span class="plan-field-key">- {key}</span>
															<span class="plan-field-value">{typeof value === 'object' ? JSON.stringify(value) : String(value)}</span>
														</div>
													{/if}
												{/each}
											{:else if change.action === 'replace'}
												<div class="plan-detail-header">Must be replaced (destroy and recreate)</div>
											{/if}
										</div>
									</details>
								{:else}
									<div class="plan-item noop">
										<span class="plan-action {change.action}">no change</span>
										<span class="plan-address">{change.address}</span>
									</div>
								{/if}
							{/each}
						</div>
					{/if}
				{:else if workspace.planning}
					<div class="empty">Running plan...</div>
				{:else if workspace.planError}
					<div class="plan-error">
						<div class="plan-error-title">Plan Failed</div>
						<pre class="plan-error-detail">{workspace.planError}</pre>
						{#if workspace.planRawOutput}
							<details class="plan-raw">
								<summary>Raw Output</summary>
								<pre>{workspace.planRawOutput}</pre>
							</details>
						{/if}
					</div>
				{:else}
					<div class="empty">No plan data. Click "Plan" to preview changes.</div>
				{/if}
			{/if}
		</div>
	{/if}
</div>

<style>
	.bottom-panel {
		border-top: 1px solid var(--border);
		background: var(--bg-panel);
		display: flex;
		flex-direction: column;
		min-height: 36px;
	}

	.bottom-panel.collapsed {
		max-height: 36px;
		overflow: hidden;
	}

	.panel-tabs {
		display: flex;
		padding: 0 8px;
		border-bottom: 1px solid var(--border);
		flex-shrink: 0;
	}

	.panel-tabs button {
		padding: 8px 12px;
		font-size: 14px;
		color: var(--text-muted);
		background: none;
		border: none;
		border-bottom: 2px solid transparent;
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.panel-tabs button:hover {
		color: #a1a1aa;
	}

	.panel-tabs button.active {
		color: var(--text);
		border-bottom-color: var(--accent);
	}

	.badge {
		font-size: 12px;
		padding: 1px 6px;
		border-radius: 8px;
		font-weight: 600;
	}

	.badge.error {
		background: rgba(239, 68, 68, 0.2);
		color: #ef4444;
	}

	.badge.info {
		background: rgba(59, 130, 246, 0.2);
		color: #60a5fa;
	}

	.panel-content {
		overflow-y: auto;
		flex: 1;
		padding: 8px;
	}

	.empty {
		color: var(--text-subtle);
		text-align: center;
		padding: 16px;
		font-size: 14px;
	}

	.diag-list {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.diag-item {
		display: flex;
		gap: 8px;
		padding: 6px 8px;
		border-radius: 4px;
		font-size: 14px;
	}

	.diag-item.error {
		background: rgba(239, 68, 68, 0.05);
	}

	.diag-item.warning {
		background: rgba(245, 158, 11, 0.05);
	}

	.diag-severity {
		width: 20px;
		height: 20px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 14px;
		font-weight: 700;
		flex-shrink: 0;
	}

	.diag-item.error .diag-severity {
		background: rgba(239, 68, 68, 0.2);
		color: #ef4444;
	}

	.diag-item.warning .diag-severity {
		background: rgba(245, 158, 11, 0.2);
		color: #f59e0b;
	}

	.diag-summary {
		color: var(--text);
	}

	.diag-detail {
		color: var(--text-muted);
		margin-top: 2px;
	}

	.diag-location {
		color: var(--text-subtle);
		font-family: monospace;
		font-size: 13px;
		margin-top: 2px;
	}

	.plan-summary {
		display: flex;
		gap: 12px;
		padding: 8px;
		margin-bottom: 8px;
	}

	.plan-stat {
		font-size: 14px;
		font-weight: 600;
	}

	.plan-stat.create {
		color: #22c55e;
	}
	.plan-stat.update {
		color: #f59e0b;
	}
	.plan-stat.delete {
		color: #ef4444;
	}
	.plan-stat.replace {
		color: #f97316;
	}

	.plan-list {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.plan-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 4px 8px;
		font-size: 14px;
		border-radius: 4px;
	}

	.plan-action {
		font-size: 12px;
		padding: 2px 6px;
		border-radius: 3px;
		font-weight: 600;
		text-transform: uppercase;
	}

	.plan-action.create {
		background: rgba(34, 197, 94, 0.15);
		color: #22c55e;
	}
	.plan-action.update {
		background: rgba(245, 158, 11, 0.15);
		color: #f59e0b;
	}
	.plan-action.delete {
		background: rgba(239, 68, 68, 0.15);
		color: #ef4444;
	}
	.plan-action.replace {
		background: rgba(249, 115, 22, 0.15);
		color: #f97316;
	}

	.plan-address {
		font-family: monospace;
		color: var(--text-muted, #a1a1aa);
		flex: 1;
	}

	.plan-field-count {
		font-size: 12px;
		color: var(--text-subtle, #565f89);
		margin-left: auto;
	}

	.plan-item-details {
		border: 1px solid var(--border, #2f3146);
		border-radius: 6px;
		margin-bottom: 4px;
		overflow: hidden;
	}

	.plan-item-details summary {
		cursor: pointer;
		list-style: none;
	}

	.plan-item-details summary::-webkit-details-marker {
		display: none;
	}

	.plan-item-details[open] summary {
		border-bottom: 1px solid var(--border, #2f3146);
	}

	.plan-item.noop {
		opacity: 0.4;
	}

	.plan-item-body {
		padding: 8px 12px;
		font-size: 13px;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		max-height: 300px;
		overflow-y: auto;
	}

	.plan-detail-header {
		font-size: 12px;
		color: var(--text-muted);
		margin-bottom: 6px;
		font-family: -apple-system, sans-serif;
		font-weight: 500;
	}

	.plan-field {
		display: flex;
		align-items: baseline;
		gap: 8px;
		padding: 2px 0;
		font-size: 13px;
		line-height: 1.4;
	}

	.plan-field-key {
		color: var(--text);
		font-weight: 500;
		white-space: nowrap;
	}

	.plan-field .plan-field-key {
		color: #22c55e;
	}

	.plan-field.delete .plan-field-key {
		color: #ef4444;
	}

	.plan-field.changed .plan-field-key {
		color: #f59e0b;
	}

	.plan-field.unknown .plan-field-key {
		color: #22c55e;
	}

	.plan-field-value {
		color: var(--text-muted);
		word-break: break-all;
		max-width: 500px;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.plan-field.unknown .plan-field-value {
		color: var(--text-subtle);
		font-style: italic;
	}

	.plan-field-before {
		color: #ef4444;
		text-decoration: line-through;
		opacity: 0.7;
	}

	.plan-field-arrow {
		color: var(--text-subtle);
	}

	.plan-field-after {
		color: #22c55e;
	}

	.validate-status {
		padding: 10px 12px;
		font-size: 14px;
		font-weight: 500;
		border-radius: 6px;
		margin-bottom: 8px;
	}

	.validate-status.running {
		color: var(--accent, #7aa2f7);
		background: rgba(122, 162, 247, 0.08);
	}

	.validate-status.pass {
		color: #22c55e;
		background: rgba(34, 197, 94, 0.08);
	}

	.validate-status.fail {
		color: #ef4444;
		background: rgba(239, 68, 68, 0.08);
	}

	.plan-error {
		padding: 12px;
		color: var(--text);
		font-size: 14px;
		background: rgba(239, 68, 68, 0.05);
		border-radius: 4px;
		margin: 4px;
	}

	.plan-error-title {
		font-weight: 600;
		color: #ef4444;
		margin-bottom: 8px;
		font-size: 14px;
	}

	.plan-error-detail {
		white-space: pre-wrap;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 14px;
		color: #f87171;
		background: rgba(0, 0, 0, 0.2);
		padding: 8px;
		border-radius: 4px;
		margin: 0;
		line-height: 1.4;
	}

	.plan-raw {
		margin-top: 8px;
	}

	.plan-raw summary {
		cursor: pointer;
		color: var(--text-muted);
		font-size: 14px;
	}

	.plan-raw pre {
		white-space: pre-wrap;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 14px;
		color: #a1a1aa;
		background: rgba(0, 0, 0, 0.2);
		padding: 8px;
		border-radius: 4px;
		margin-top: 4px;
		max-height: 300px;
		overflow-y: auto;
	}
</style>

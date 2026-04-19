import { test, expect } from '@playwright/test';
import { exec } from 'child_process';

// Start backend before tests
let backendProcess: ReturnType<typeof exec>;

test.beforeAll(async () => {
	// Check if backend is already running
	try {
		const response = await fetch('http://localhost:3001/api/health');
		if (response.ok) return; // Already running
	} catch {
		// Start it
		backendProcess = exec('cd backend && go run ./cmd/server', { cwd: process.cwd() });
		// Wait for it to be ready
		for (let i = 0; i < 20; i++) {
			try {
				const response = await fetch('http://localhost:3001/api/health');
				if (response.ok) break;
			} catch {
				await new Promise((r) => setTimeout(r, 500));
			}
		}
	}
});

test.afterAll(() => {
	if (backendProcess) {
		backendProcess.kill();
	}
});

test.describe('TerraGraph IDE', () => {
	test('loads the IDE with empty state', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByText('TerraGraph')).toBeVisible();
		await expect(page.getByText('No workspace loaded')).toBeVisible();
		await expect(page.getByPlaceholder('Enter Terraform workspace path')).toBeVisible();
	});

	test('loads a workspace and shows graph', async ({ page }) => {
		await page.goto('/');

		// Enter workspace path
		const input = page.getByPlaceholder('Enter Terraform workspace path');
		await input.fill(process.cwd() + '/examples/simple');
		await page.getByRole('button', { name: 'Load' }).click();

		// Wait for graph to render
		await expect(page.getByText('instance_type')).toBeVisible({ timeout: 10000 });
		await expect(page.getByText('environment')).toBeVisible();
		await expect(page.getByText('web')).toBeVisible();
		await expect(page.getByText('web_sg')).toBeVisible();

		// Sidebar should show files and nodes
		await expect(page.getByText('main.tf')).toBeVisible();
		await expect(page.getByText('All (8)')).toBeVisible();
	});

	test('selects a node and shows inspector', async ({ page }) => {
		await page.goto('/');

		const input = page.getByPlaceholder('Enter Terraform workspace path');
		await input.fill(process.cwd() + '/examples/simple');
		await page.getByRole('button', { name: 'Load' }).click();

		// Wait for nodes to appear
		await expect(page.getByText('instance_type')).toBeVisible({ timeout: 10000 });

		// Click on a node in the sidebar
		await page.getByRole('button', { name: /instance_type/ }).first().click();

		// Inspector should show
		await expect(page.getByText('Inspector')).toBeVisible();
		await expect(page.getByText('Variable')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Attributes' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Source' })).toBeVisible();
	});

	test('runs validation and shows diagnostics', async ({ page }) => {
		await page.goto('/');

		const input = page.getByPlaceholder('Enter Terraform workspace path');
		await input.fill(process.cwd() + '/examples/simple');
		await page.getByRole('button', { name: 'Load' }).click();

		// Wait for load
		await expect(page.getByText('instance_type')).toBeVisible({ timeout: 10000 });

		// Click validate
		await page.getByRole('button', { name: 'Validate' }).click();

		// Should show diagnostics (missing provider since we haven't run init)
		await expect(page.getByText('Diagnostics').first()).toBeVisible({ timeout: 10000 });
	});

	test('loads multi-file workspace', async ({ page }) => {
		await page.goto('/');

		const input = page.getByPlaceholder('Enter Terraform workspace path');
		await input.fill(process.cwd() + '/examples/multi-file');
		await page.getByRole('button', { name: 'Load' }).click();

		// Wait for nodes
		await expect(page.getByText('All (18)')).toBeVisible({ timeout: 10000 });

		// Should show multiple files
		await expect(page.getByText('compute.tf')).toBeVisible();
		await expect(page.getByText('network.tf')).toBeVisible();
		await expect(page.getByText('variables.tf')).toBeVisible();
	});
});

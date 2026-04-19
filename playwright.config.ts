import { defineConfig } from '@playwright/test';

export default defineConfig({
	webServer: {
		command: 'bun run dev --port 4173',
		port: 4173,
		reuseExistingServer: true
	},
	testDir: 'tests/e2e',
	testMatch: '**/*.spec.ts'
});

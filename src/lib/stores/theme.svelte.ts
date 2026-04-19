type Theme = 'dark' | 'light' | 'solarized';

class ThemeStore {
	current = $state<Theme>('dark');

	toggle() {
		const themes: Theme[] = ['dark', 'light', 'solarized'];
		const idx = themes.indexOf(this.current);
		this.current = themes[(idx + 1) % themes.length];
	}

	set(theme: Theme) {
		this.current = theme;
	}
}

export const theme = new ThemeStore();

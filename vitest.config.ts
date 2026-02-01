import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'jsdom',
    include: ['web/**/*.test.ts'],
    globals: true,
  },
  resolve: {
    extensions: ['.ts', '.js'],
  },
  esbuild: {
    target: 'esnext',
  },
});

import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    css: true,
    // E2Eテストディレクトリを除外してユニットテストのみを実行
    exclude: [
      '**/node_modules/**',
      '**/tests/e2e/**',
      '**/playwright-report/**',
      '**/test-results/**'
    ],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'json'],
      exclude: [
        'node_modules/',
        'src/test/',
        'tests/e2e/',
        '**/*.d.ts',
        '**/*.config.*',
        'dist/',
        'playwright-report/',
        'test-results/',
      ],
    },
  },
})

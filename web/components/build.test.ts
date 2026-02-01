/**
 * Build Integration Tests
 *
 * These tests verify that the frontend build pipeline (UnoCSS + esbuild)
 * produces valid output files that can be used in the static site.
 *
 * Requirements covered:
 * - 4.4: UnoCSS for styling
 * - 4.5: Lit Web Components
 * - 4.6: esbuild bundling
 */
import { describe, it, expect, beforeAll } from 'vitest';
import { execSync } from 'node:child_process';
import { existsSync, readFileSync, mkdirSync, rmSync } from 'node:fs';
import { join } from 'node:path';

// Project root directory
const projectRoot = join(import.meta.dirname, '../..');
const distDir = join(projectRoot, 'dist');

describe('Build Pipeline Integration', () => {
  // Run build before tests to ensure we have fresh outputs
  beforeAll(() => {
    // Clean dist directory for fresh build
    if (existsSync(distDir)) {
      rmSync(distDir, { recursive: true, force: true });
    }
    mkdirSync(distDir, { recursive: true });

    // Run the full build - this MUST succeed
    try {
      execSync('npm run build', { cwd: projectRoot, stdio: 'pipe' });
    } catch (error) {
      // If build fails, throw with details
      const err = error as { stderr?: Buffer; stdout?: Buffer };
      const stderr = err.stderr?.toString() || '';
      const stdout = err.stdout?.toString() || '';
      throw new Error(`Build failed:\nstdout: ${stdout}\nstderr: ${stderr}`);
    }
  });

  describe('esbuild bundle (Requirement 4.6)', () => {
    it('should produce components.js bundle', () => {
      const bundlePath = join(distDir, 'components.js');
      expect(existsSync(bundlePath), 'components.js must exist after build').toBe(true);
    });

    it('should produce valid ESM output', () => {
      const bundlePath = join(distDir, 'components.js');
      const content = readFileSync(bundlePath, 'utf-8');

      // ESM format indicators
      expect(content.length).toBeGreaterThan(0);

      // Should NOT contain CommonJS patterns
      expect(content.includes('module.exports')).toBe(false);
      expect(content.includes('require(')).toBe(false);
    });

    it('should contain Lit component registration', () => {
      const bundlePath = join(distDir, 'components.js');
      const content = readFileSync(bundlePath, 'utf-8');

      // Should contain proposal-filter custom element
      expect(content.includes('proposal-filter') || content.includes('customElements')).toBe(true);
    });

    it('should be minified for production', () => {
      const bundlePath = join(distDir, 'components.js');
      const content = readFileSync(bundlePath, 'utf-8');
      const lines = content.split('\n').filter((line) => line.trim() !== '');

      // Minified code typically has very long lines or few lines total
      // If has sourcemap reference, that's also valid for minified builds
      const avgLineLength = content.length / (lines.length || 1);
      expect(avgLineLength > 50 || content.includes('//# sourceMappingURL=')).toBe(true);
    });

    it('should generate source map for debugging', () => {
      const mapPath = join(distDir, 'components.js.map');
      expect(existsSync(mapPath), 'components.js.map must exist after build').toBe(true);

      const content = readFileSync(mapPath, 'utf-8');
      const parsed = JSON.parse(content);

      // Should have version 3 source map format
      expect(parsed.version).toBe(3);
      expect(parsed.sources).toBeDefined();
      expect(parsed.mappings).toBeDefined();
    });
  });

  describe('UnoCSS output (Requirement 4.4)', () => {
    it('should produce styles.css', () => {
      const cssPath = join(distDir, 'styles.css');
      expect(existsSync(cssPath), 'styles.css must exist after build').toBe(true);
    });

    it('should generate valid CSS', () => {
      const cssPath = join(distDir, 'styles.css');
      const content = readFileSync(cssPath, 'utf-8');

      // Should have CSS content (not empty)
      expect(content.length).toBeGreaterThan(0);

      // Should be valid CSS (starts with a rule or comment or @charset)
      expect(content.match(/^(@|\/\*|[.#\[\*a-zA-Z:])/m)).toBeTruthy();
    });

    it('should extract utility classes from templ files', () => {
      const cssPath = join(distDir, 'styles.css');
      const content = readFileSync(cssPath, 'utf-8');

      // Common UnoCSS utility classes used in the templ templates
      // At least one of these should be present
      const utilityPatterns = [
        /\.container/, // container class
        /\.mx-auto/, // margin auto
        /\.px-\d/, // padding x
        /\.py-\d/, // padding y
        /\.bg-/, // background
        /\.text-/, // text color/size
        /\.flex/, // flexbox
        /\.min-h-/, // min height
      ];

      const hasUtilities = utilityPatterns.some((pattern) => pattern.test(content));
      expect(hasUtilities, 'CSS should contain UnoCSS utility classes from templ files').toBe(true);
    });

    it('should include status badge styling classes', () => {
      const cssPath = join(distDir, 'styles.css');
      const content = readFileSync(cssPath, 'utf-8');

      // Status badges use rounded-full and inline-flex utility classes
      // These should be extracted from the templ templates
      expect(
        content.includes('rounded-full') || content.includes('inline-flex'),
        'CSS should include status badge styling classes (rounded-full, inline-flex)',
      ).toBe(true);
    });
  });

  describe('Component Registration (Requirement 4.5)', () => {
    it('should export ProposalFilter class', async () => {
      const module = await import('./proposal-filter.js');
      expect(module.ProposalFilter).toBeDefined();
      expect(typeof module.ProposalFilter).toBe('function');
    });

    it('should register proposal-filter custom element', async () => {
      await import('./proposal-filter.js');
      const registered = customElements.get('proposal-filter');
      expect(registered).toBeDefined();
    });

    it('should export initializeFilters function', async () => {
      const module = await import('./index.js');
      expect(module.initializeFilters).toBeDefined();
      expect(typeof module.initializeFilters).toBe('function');
    });
  });

  describe('TypeScript Configuration', () => {
    it('should use ESNext target for modern features', () => {
      const tsconfigPath = join(projectRoot, 'tsconfig.json');
      expect(existsSync(tsconfigPath)).toBe(true);

      const tsconfig = JSON.parse(readFileSync(tsconfigPath, 'utf-8'));
      const target = tsconfig.compilerOptions?.target?.toLowerCase() || '';
      expect(['esnext', 'es2022', 'es2023', 'es2024'].some((t) => target.includes(t))).toBe(true);
    });

    it('should enable strict mode', () => {
      const tsconfigPath = join(projectRoot, 'tsconfig.json');
      const tsconfig = JSON.parse(readFileSync(tsconfigPath, 'utf-8'));
      expect(tsconfig.compilerOptions?.strict).toBe(true);
    });
  });
});

describe('UnoCSS Configuration', () => {
  it('should have uno.config.mjs with required presets', () => {
    const configPath = join(projectRoot, 'uno.config.mjs');
    expect(existsSync(configPath)).toBe(true);

    const content = readFileSync(configPath, 'utf-8');

    // Should import required presets
    expect(content.includes('presetUno')).toBe(true);
    expect(content.includes('presetTypography')).toBe(true);
    expect(content.includes('presetIcons')).toBe(true);
  });

  it('should have templ extractor configured', () => {
    const configPath = join(projectRoot, 'uno.config.mjs');
    const content = readFileSync(configPath, 'utf-8');

    // Should have templ extractor
    expect(content.includes('templExtractor') || content.includes('templ')).toBe(true);

    // Should target templ files
    expect(content.includes('.templ')).toBe(true);
  });

  it('should define status badge shortcuts', () => {
    const configPath = join(projectRoot, 'uno.config.mjs');
    const content = readFileSync(configPath, 'utf-8');

    // Should have status badge shortcuts for requirement 4.2
    expect(content.includes('badge-accepted')).toBe(true);
    expect(content.includes('badge-declined')).toBe(true);
    expect(content.includes('badge-hold')).toBe(true);
  });

  it('should define status colors in theme', () => {
    const configPath = join(projectRoot, 'uno.config.mjs');
    const content = readFileSync(configPath, 'utf-8');

    // Should have status colors defined
    expect(content.includes('accepted')).toBe(true);
    expect(content.includes('declined')).toBe(true);
    expect(content.includes('likelyAccept')).toBe(true);
  });
});

describe('Package Configuration', () => {
  it('should have all required build scripts', () => {
    const packagePath = join(projectRoot, 'package.json');
    const pkg = JSON.parse(readFileSync(packagePath, 'utf-8'));

    expect(pkg.scripts['build:css']).toBeDefined();
    expect(pkg.scripts['build:js']).toBeDefined();
    expect(pkg.scripts['build']).toBeDefined();

    // Build script should run both CSS and JS builds
    expect(pkg.scripts['build']).toContain('build:css');
    expect(pkg.scripts['build']).toContain('build:js');
  });

  it('should have required production dependencies', () => {
    const packagePath = join(projectRoot, 'package.json');
    const pkg = JSON.parse(readFileSync(packagePath, 'utf-8'));

    expect(pkg.dependencies?.lit).toBeDefined();
  });

  it('should have required dev dependencies', () => {
    const packagePath = join(projectRoot, 'package.json');
    const pkg = JSON.parse(readFileSync(packagePath, 'utf-8'));

    expect(pkg.devDependencies?.['@unocss/cli']).toBeDefined();
    expect(pkg.devDependencies?.esbuild).toBeDefined();
    expect(pkg.devDependencies?.typescript).toBeDefined();
    expect(pkg.devDependencies?.vitest).toBeDefined();
  });

  it('should use ESM module type', () => {
    const packagePath = join(projectRoot, 'package.json');
    const pkg = JSON.parse(readFileSync(packagePath, 'utf-8'));

    expect(pkg.type).toBe('module');
  });
});

describe('Build Output Integrity', () => {
  it('should have consistent file structure', () => {
    // All required output files should exist after build
    const requiredFiles = ['styles.css', 'components.js', 'components.js.map'];

    for (const file of requiredFiles) {
      const filePath = join(distDir, file);
      expect(existsSync(filePath), `${file} must exist in dist/`).toBe(true);
    }
  });

  it('should have non-empty output files', () => {
    const files = ['styles.css', 'components.js'];

    for (const file of files) {
      const filePath = join(distDir, file);
      const content = readFileSync(filePath, 'utf-8');
      expect(content.length, `${file} should not be empty`).toBeGreaterThan(100);
    }
  });
});

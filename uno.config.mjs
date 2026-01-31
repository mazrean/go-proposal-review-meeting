import {
  defineConfig,
  presetUno,
  presetTypography,
  presetIcons,
  transformerDirectives,
} from 'unocss';
import { extractorSplit } from '@unocss/core';

/**
 * Custom extractor for templ files
 * Extracts class names from templ template syntax:
 * - class="..." attributes
 * - templ.Attributes with class values
 */
const templExtractor = {
  name: 'templ',
  order: 0,
  extract({ code }) {
    const classMatches = code.matchAll(/class=["']([^"']+)["']/g);
    const classes = [];

    for (const match of classMatches) {
      classes.push(...match[1].split(/\s+/).filter(Boolean));
    }

    // Also extract from templ.Attributes patterns
    const attrMatches = code.matchAll(/templ\.Attributes\{[^}]*"class":\s*"([^"]+)"/g);
    for (const match of attrMatches) {
      classes.push(...match[1].split(/\s+/).filter(Boolean));
    }

    return new Set(classes);
  },
};

export default defineConfig({
  presets: [
    presetUno(),
    presetTypography(),
    presetIcons({
      scale: 1.2,
      cdn: 'https://esm.sh/',
    }),
  ],
  transformers: [
    transformerDirectives(),
  ],
  extractors: [
    extractorSplit,
    templExtractor,
  ],
  content: {
    filesystem: [
      'internal/**/*.templ',
      'web/components/**/*.ts',
    ],
  },
  theme: {
    colors: {
      // Status colors for proposals
      accepted: {
        DEFAULT: '#22c55e',
        light: '#dcfce7',
      },
      declined: {
        DEFAULT: '#ef4444',
        light: '#fee2e2',
      },
      hold: {
        DEFAULT: '#f59e0b',
        light: '#fef3c7',
      },
      active: {
        DEFAULT: '#3b82f6',
        light: '#dbeafe',
      },
      discussions: {
        DEFAULT: '#8b5cf6',
        light: '#ede9fe',
      },
      likelyAccept: {
        DEFAULT: '#10b981',
        light: '#d1fae5',
      },
      likelyDecline: {
        DEFAULT: '#f97316',
        light: '#ffedd5',
      },
    },
  },
  shortcuts: {
    // Status badge shortcuts
    'badge': 'inline-flex items-center px-2 py-1 rounded-md text-sm font-medium',
    'badge-accepted': 'badge bg-accepted-light text-accepted',
    'badge-declined': 'badge bg-declined-light text-declined',
    'badge-hold': 'badge bg-hold-light text-hold',
    'badge-active': 'badge bg-active-light text-active',
    'badge-discussions': 'badge bg-discussions-light text-discussions',
    'badge-likely-accept': 'badge bg-likelyAccept-light text-likelyAccept',
    'badge-likely-decline': 'badge bg-likelyDecline-light text-likelyDecline',
  },
});

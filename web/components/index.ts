/**
 * Web Components entry point
 * Lit components are exported from here and bundled by esbuild
 */

// ProposalFilter - Filter proposals by status on weekly index pages
export { ProposalFilter } from './proposal-filter.js';
export type { ProposalStatus, FilterStatus, FilterChangeDetail } from './proposal-filter.js';

import type { FilterChangeDetail, FilterStatus } from './proposal-filter.js';

/**
 * Valid filter status values.
 */
const VALID_FILTER_STATUSES: ReadonlySet<string> = new Set([
  'all',
  'accepted',
  'declined',
  'likely_accept',
  'likely_decline',
  'discussions',
  'active',
  'hold',
]);

/**
 * Type guard to validate FilterChangeDetail with strict status validation.
 */
function isValidFilterDetail(detail: unknown): detail is FilterChangeDetail {
  if (typeof detail !== 'object' || detail === null) {
    return false;
  }
  const d = detail as Record<string, unknown>;
  return typeof d.status === 'string' && VALID_FILTER_STATUSES.has(d.status);
}

/**
 * Initialize filter functionality on the page.
 * This connects the proposal-filter element to proposal list items.
 * Exported for testing purposes.
 */
export function initializeFilters(): void {
  document.querySelectorAll('proposal-filter').forEach((filter) => {
    filter.addEventListener('filter-change', (event: Event) => {
      const customEvent = event as CustomEvent<FilterChangeDetail>;

      // Guard against malformed event detail
      if (!isValidFilterDetail(customEvent.detail)) {
        console.warn('Invalid filter-change event detail:', customEvent.detail);
        return;
      }

      const status: FilterStatus = customEvent.detail.status;
      const container = filter.closest('.weekly-index');
      if (!container) return;

      const proposals = container.querySelectorAll('article[data-status]');
      proposals.forEach((proposal) => {
        const proposalStatus = proposal.getAttribute('data-status');
        if (status === 'all' || proposalStatus === status) {
          (proposal as HTMLElement).style.display = '';
        } else {
          (proposal as HTMLElement).style.display = 'none';
        }
      });
    });
  });
}

// Initialize when DOM is ready
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initializeFilters);
} else {
  initializeFilters();
}

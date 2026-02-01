import { LitElement, html, css } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';

/**
 * Valid proposal status values matching Go parser.Status.
 */
export type ProposalStatus =
  | 'accepted'
  | 'declined'
  | 'likely_accept'
  | 'likely_decline'
  | 'discussions'
  | 'active'
  | 'hold';

/**
 * Filter status including 'all' for no filter.
 */
export type FilterStatus = ProposalStatus | 'all';

/**
 * Event detail for filter-change events.
 */
export interface FilterChangeDetail {
  status: FilterStatus;
}

/**
 * Status labels mapping for display.
 * Maps internal status values to human-readable labels.
 */
const STATUS_LABELS: Record<ProposalStatus, string> = {
  accepted: 'Accepted',
  declined: 'Declined',
  likely_accept: 'Likely Accept',
  likely_decline: 'Likely Decline',
  discussions: 'Discussions',
  active: 'Active',
  hold: 'Hold',
};

/**
 * ProposalFilter is a Lit web component that provides filtering functionality
 * for proposal lists. It renders filter buttons for each status and emits
 * a 'filter-change' event when a filter is selected.
 *
 * @fires filter-change - Dispatched when a filter button is clicked
 * @example
 * ```html
 * <proposal-filter statuses='["accepted", "declined", "discussions"]'></proposal-filter>
 * ```
 */
@customElement('proposal-filter')
export class ProposalFilter extends LitElement {
  static styles = css`
    :host {
      display: block;
      font-family: 'Noto Sans JP', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    }

    .filter-container {
      display: flex;
      flex-wrap: wrap;
      gap: 0.5rem;
      padding: 0.5rem 0;
    }

    button {
      padding: 0.5rem 1rem;
      border: 1px solid #e5e7eb;
      border-radius: 9999px;
      background-color: white;
      color: #374151;
      font-size: 0.875rem;
      font-weight: 500;
      font-family: inherit;
      cursor: pointer;
      transition:
        background-color 0.2s,
        border-color 0.2s,
        color 0.2s;
    }

    button:hover {
      background-color: #f3f4f6;
      border-color: #d1d5db;
    }

    button:focus {
      outline: 2px solid #3b82f6;
      outline-offset: 2px;
    }

    button[aria-pressed='true'] {
      background-color: #3b82f6;
      border-color: #3b82f6;
      color: white;
    }

    button[aria-pressed='true']:hover {
      background-color: #2563eb;
      border-color: #2563eb;
    }
  `;

  /**
   * Array of status strings to display as filter buttons.
   * @example '["accepted", "declined", "discussions"]'
   */
  @property({ type: Array })
  statuses: ProposalStatus[] = [];

  /**
   * Currently selected filter status.
   * 'all' means no filter is applied.
   */
  @state()
  private selectedStatus: FilterStatus = 'all';

  /**
   * Get the display label for a status.
   */
  private getStatusLabel(status: ProposalStatus): string {
    return STATUS_LABELS[status] || status;
  }

  /**
   * Handle filter button click.
   * Updates the selected status and dispatches a filter-change event.
   */
  private handleClick(status: FilterStatus): void {
    this.selectedStatus = status;
    this.dispatchEvent(
      new CustomEvent<FilterChangeDetail>('filter-change', {
        detail: { status },
        bubbles: true,
        composed: true,
      }),
    );
  }

  render() {
    return html`
      <div class="filter-container" role="group" aria-label="ステータスフィルター">
        <button
          type="button"
          data-status="all"
          aria-pressed="${this.selectedStatus === 'all'}"
          @click="${() => this.handleClick('all')}"
        >
          すべて
        </button>
        ${this.statuses.map(
          (status) => html`
            <button
              type="button"
              data-status="${status}"
              aria-pressed="${this.selectedStatus === status}"
              @click="${() => this.handleClick(status)}"
            >
              ${this.getStatusLabel(status)}
            </button>
          `,
        )}
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'proposal-filter': ProposalFilter;
  }
}

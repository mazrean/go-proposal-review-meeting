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

    .dropdown-container {
      position: relative;
      display: inline-block;
      width: 100%;
      max-width: 320px;
    }

    .dropdown-button {
      width: 100%;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 0.75rem;
      padding: 0.75rem 1rem;
      border: 2px solid #e5e7eb;
      border-radius: 0.5rem;
      background-color: white;
      color: #1f2937;
      font-size: 0.9375rem;
      font-weight: 500;
      font-family: inherit;
      cursor: pointer;
      transition:
        border-color 0.2s,
        box-shadow 0.2s;
    }

    .dropdown-button:hover {
      border-color: #00acd7;
    }

    .dropdown-button:focus {
      outline: none;
      border-color: #00acd7;
      box-shadow: 0 0 0 3px rgba(0, 172, 215, 0.1);
    }

    .dropdown-button[aria-expanded='true'] {
      border-color: #00acd7;
      box-shadow: 0 0 0 3px rgba(0, 172, 215, 0.1);
    }

    .button-content {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      flex: 1;
      min-width: 0;
    }

    .status-icon {
      width: 0.625rem;
      height: 0.625rem;
      border-radius: 50%;
      flex-shrink: 0;
    }

    .button-text {
      flex: 1;
      text-align: left;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .chevron {
      width: 1.25rem;
      height: 1.25rem;
      flex-shrink: 0;
      transition: transform 0.2s;
      color: #6b7280;
    }

    .dropdown-button[aria-expanded='true'] .chevron {
      transform: rotate(180deg);
    }

    .dropdown-menu {
      position: absolute;
      top: calc(100% + 0.5rem);
      left: 0;
      width: 100%;
      max-height: 320px;
      overflow-y: auto;
      background-color: white;
      border: 1px solid #e5e7eb;
      border-radius: 0.5rem;
      box-shadow:
        0 4px 6px -1px rgba(0, 0, 0, 0.1),
        0 2px 4px -1px rgba(0, 0, 0, 0.06);
      z-index: 50;
      opacity: 0;
      transform: translateY(-8px) scale(0.95);
      transition:
        opacity 0.2s,
        transform 0.2s;
      pointer-events: none;
    }

    .dropdown-menu[data-open='true'] {
      opacity: 1;
      transform: translateY(0) scale(1);
      pointer-events: auto;
    }

    .menu-item {
      width: 100%;
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.75rem 1rem;
      border: none;
      background: none;
      color: #374151;
      font-size: 0.875rem;
      font-weight: 500;
      font-family: inherit;
      text-align: left;
      cursor: pointer;
      transition:
        background-color 0.15s,
        color 0.15s;
    }

    .menu-item:hover {
      background-color: #f9fafb;
    }

    .menu-item:focus {
      outline: none;
      background-color: #f3f4f6;
    }

    .menu-item[aria-selected='true'] {
      background-color: #eff6ff;
      color: #1e40af;
    }

    .menu-item[aria-selected='true'] .status-icon {
      box-shadow: 0 0 0 2px white, 0 0 0 4px #00acd7;
    }

    .menu-item-content {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      flex: 1;
    }

    .check-icon {
      width: 1rem;
      height: 1rem;
      color: #00acd7;
      flex-shrink: 0;
    }

    /* Status color mappings */
    .status-accepted { background-color: #10b981; }
    .status-declined { background-color: #ef4444; }
    .status-likely_accept { background-color: #34d399; }
    .status-likely_decline { background-color: #f97316; }
    .status-discussions { background-color: #a855f7; }
    .status-active { background-color: #0ea5e9; }
    .status-hold { background-color: #eab308; }
    .status-all { background-color: #6b7280; }
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
   * Whether the dropdown menu is open.
   */
  @state()
  private isOpen = false;

  /**
   * Index of the focused menu item (for keyboard navigation).
   */
  @state()
  private focusedIndex = -1;

  /**
   * Get the display label for a status.
   */
  private getStatusLabel(status: ProposalStatus): string {
    return STATUS_LABELS[status] || status;
  }

  /**
   * Get the display label for the current selected status.
   */
  private getSelectedLabel(): string {
    if (this.selectedStatus === 'all') {
      return 'すべて';
    }
    return this.getStatusLabel(this.selectedStatus as ProposalStatus);
  }

  /**
   * Toggle dropdown menu open/close.
   */
  private toggleDropdown(): void {
    this.isOpen = !this.isOpen;
    if (!this.isOpen) {
      this.focusedIndex = -1;
    }
  }

  /**
   * Close the dropdown menu.
   */
  private closeDropdown(): void {
    this.isOpen = false;
    this.focusedIndex = -1;
  }

  /**
   * Handle selection of a filter option.
   */
  private selectStatus(status: FilterStatus): void {
    this.selectedStatus = status;
    this.closeDropdown();
    this.dispatchEvent(
      new CustomEvent<FilterChangeDetail>('filter-change', {
        detail: { status },
        bubbles: true,
        composed: true,
      }),
    );
  }

  /**
   * Handle keyboard navigation.
   */
  private handleKeyDown(event: KeyboardEvent): void {
    const allStatuses: FilterStatus[] = ['all', ...this.statuses];

    switch (event.key) {
      case 'Escape':
        event.preventDefault();
        this.closeDropdown();
        break;

      case 'ArrowDown':
        event.preventDefault();
        if (!this.isOpen) {
          this.isOpen = true;
        } else {
          this.focusedIndex = Math.min(this.focusedIndex + 1, allStatuses.length - 1);
        }
        break;

      case 'ArrowUp':
        event.preventDefault();
        if (this.isOpen) {
          this.focusedIndex = Math.max(this.focusedIndex - 1, 0);
        }
        break;

      case 'Enter':
      case ' ':
        event.preventDefault();
        if (this.isOpen && this.focusedIndex >= 0) {
          this.selectStatus(allStatuses[this.focusedIndex]);
        } else {
          this.toggleDropdown();
        }
        break;

      case 'Home':
        if (this.isOpen) {
          event.preventDefault();
          this.focusedIndex = 0;
        }
        break;

      case 'End':
        if (this.isOpen) {
          event.preventDefault();
          this.focusedIndex = allStatuses.length - 1;
        }
        break;
    }
  }

  /**
   * Handle clicks outside the dropdown to close it.
   */
  private handleDocumentClick = (event: MouseEvent): void => {
    const path = event.composedPath();
    if (!path.includes(this)) {
      this.closeDropdown();
    }
  };

  /**
   * Lifecycle: Component connected to DOM.
   */
  connectedCallback(): void {
    super.connectedCallback();
    document.addEventListener('click', this.handleDocumentClick);
  }

  /**
   * Lifecycle: Component disconnected from DOM.
   */
  disconnectedCallback(): void {
    super.disconnectedCallback();
    document.removeEventListener('click', this.handleDocumentClick);
  }

  render() {
    const allStatuses: FilterStatus[] = ['all', ...this.statuses];

    return html`
      <div class="dropdown-container" @keydown="${this.handleKeyDown}">
        <button
          type="button"
          class="dropdown-button"
          aria-haspopup="listbox"
          aria-expanded="${this.isOpen}"
          aria-label="ステータスフィルター"
          @click="${this.toggleDropdown}"
        >
          <div class="button-content">
            <span class="status-icon status-${this.selectedStatus}"></span>
            <span class="button-text">${this.getSelectedLabel()}</span>
          </div>
          <svg
            class="chevron"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        <div class="dropdown-menu" role="listbox" data-open="${this.isOpen ? 'true' : 'false'}">
          ${allStatuses.map((status, index) => {
            const isSelected = this.selectedStatus === status;
            const isFocused = this.focusedIndex === index;
            const label = status === 'all' ? 'すべて' : this.getStatusLabel(status as ProposalStatus);

            return html`
              <button
                type="button"
                class="menu-item"
                role="option"
                aria-selected="${isSelected}"
                data-status="${status}"
                ?data-focused="${isFocused}"
                @click="${() => this.selectStatus(status)}"
                @mouseenter="${() => {
                  this.focusedIndex = index;
                }}"
              >
                <div class="menu-item-content">
                  <span class="status-icon status-${status}"></span>
                  <span>${label}</span>
                </div>
                ${isSelected
                  ? html`
                      <svg
                        class="check-icon"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2.5"
                          d="M5 13l4 4L19 7"
                        />
                      </svg>
                    `
                  : ''}
              </button>
            `;
          })}
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'proposal-filter': ProposalFilter;
  }
}

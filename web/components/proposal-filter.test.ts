import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import './proposal-filter.js';
import type { ProposalFilter } from './proposal-filter.js';

describe('ProposalFilter', () => {
  let container: HTMLDivElement;

  beforeEach(() => {
    container = document.createElement('div');
    document.body.appendChild(container);
  });

  afterEach(() => {
    container.remove();
  });

  it('should be defined as a custom element', () => {
    expect(customElements.get('proposal-filter')).toBeDefined();
  });

  it('should render filter buttons for provided statuses', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted", "declined", "discussions"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const buttons = element?.shadowRoot?.querySelectorAll('button');
    // Should have "All" + 3 status buttons = 4 buttons
    expect(buttons?.length).toBe(4);
  });

  it('should render "All" button as the first button', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted", "declined"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const buttons = element?.shadowRoot?.querySelectorAll('button');
    expect(buttons?.[0]?.textContent?.trim()).toBe('すべて');
  });

  it('should have "All" button active by default', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const allButton = element?.shadowRoot?.querySelector('button[data-status="all"]');
    expect(allButton?.getAttribute('aria-pressed')).toBe('true');
  });

  it('should emit filter-change event when a status button is clicked', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted", "declined"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    let eventDetail: string | null = null;
    element?.addEventListener('filter-change', ((e: CustomEvent) => {
      eventDetail = e.detail.status;
    }) as EventListener);

    const acceptedButton = element?.shadowRoot?.querySelector('button[data-status="accepted"]');
    acceptedButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    expect(eventDetail).toBe('accepted');
  });

  it('should update active button when clicked', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted", "declined"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const acceptedButton = element?.shadowRoot?.querySelector('button[data-status="accepted"]');
    acceptedButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    expect(acceptedButton?.getAttribute('aria-pressed')).toBe('true');

    const allButton = element?.shadowRoot?.querySelector('button[data-status="all"]');
    expect(allButton?.getAttribute('aria-pressed')).toBe('false');
  });

  it('should emit filter-change with "all" when All button is clicked', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    // First click on accepted
    const acceptedButton = element?.shadowRoot?.querySelector('button[data-status="accepted"]');
    acceptedButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    // Then click on all
    let eventDetail: string | null = null;
    element?.addEventListener('filter-change', ((e: CustomEvent) => {
      eventDetail = e.detail.status;
    }) as EventListener);

    const allButton = element?.shadowRoot?.querySelector('button[data-status="all"]');
    allButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    expect(eventDetail).toBe('all');
  });

  it('should display translated status labels', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted", "declined", "likely_accept"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const buttons = element?.shadowRoot?.querySelectorAll('button');
    const buttonTexts = Array.from(buttons || []).map((b) => b.textContent?.trim());

    expect(buttonTexts).toContain('Accepted');
    expect(buttonTexts).toContain('Declined');
    expect(buttonTexts).toContain('Likely Accept');
  });
});

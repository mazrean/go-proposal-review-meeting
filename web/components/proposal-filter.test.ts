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

  it('should render dropdown with menu items for provided statuses', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted", "declined", "discussions"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const dropdownButton = element?.shadowRoot?.querySelector('.dropdown-button');
    const menuItems = element?.shadowRoot?.querySelectorAll('.menu-item');

    // Should have 1 dropdown button
    expect(dropdownButton).toBeDefined();
    // Should have "All" + 3 status menu items = 4 menu items
    expect(menuItems?.length).toBe(4);
  });

  it('should display "All" as the first menu item', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted", "declined"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const menuItems = element?.shadowRoot?.querySelectorAll('.menu-item');
    const firstItemText = menuItems?.[0]?.textContent?.trim();
    expect(firstItemText).toContain('すべて');
  });

  it('should have "All" selected by default', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const allMenuItem = element?.shadowRoot?.querySelector('.menu-item[data-status="all"]');
    expect(allMenuItem?.getAttribute('aria-selected')).toBe('true');

    // Dropdown button should show "すべて"
    const dropdownButton = element?.shadowRoot?.querySelector('.dropdown-button');
    expect(dropdownButton?.textContent).toContain('すべて');
  });

  it('should emit filter-change event when a menu item is clicked', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted", "declined"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    let eventDetail: string | null = null;
    element?.addEventListener('filter-change', ((e: CustomEvent) => {
      eventDetail = e.detail.status;
    }) as EventListener);

    const acceptedMenuItem = element?.shadowRoot?.querySelector('.menu-item[data-status="accepted"]');
    acceptedMenuItem?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    expect(eventDetail).toBe('accepted');
  });

  it('should update selected menu item when clicked', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted", "declined"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const acceptedMenuItem = element?.shadowRoot?.querySelector('.menu-item[data-status="accepted"]');
    acceptedMenuItem?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    expect(acceptedMenuItem?.getAttribute('aria-selected')).toBe('true');

    const allMenuItem = element?.shadowRoot?.querySelector('.menu-item[data-status="all"]');
    expect(allMenuItem?.getAttribute('aria-selected')).toBe('false');

    // Dropdown button should show "Accepted"
    const dropdownButton = element?.shadowRoot?.querySelector('.dropdown-button');
    expect(dropdownButton?.textContent).toContain('Accepted');
  });

  it('should emit filter-change with "all" when All menu item is clicked', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    // First click on accepted
    const acceptedMenuItem = element?.shadowRoot?.querySelector('.menu-item[data-status="accepted"]');
    acceptedMenuItem?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    // Then click on all
    let eventDetail: string | null = null;
    element?.addEventListener('filter-change', ((e: CustomEvent) => {
      eventDetail = e.detail.status;
    }) as EventListener);

    const allMenuItem = element?.shadowRoot?.querySelector('.menu-item[data-status="all"]');
    allMenuItem?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    expect(eventDetail).toBe('all');
  });

  it('should display translated status labels in menu items', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted", "declined", "likely_accept"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const menuItems = element?.shadowRoot?.querySelectorAll('.menu-item');
    const menuTexts = Array.from(menuItems || []).map((item) => item.textContent?.trim());

    expect(menuTexts).toContain('Accepted');
    expect(menuTexts).toContain('Declined');
    expect(menuTexts).toContain('Likely Accept');
  });

  it('should toggle dropdown when button is clicked', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const dropdownButton = element?.shadowRoot?.querySelector('.dropdown-button');
    const dropdownMenu = element?.shadowRoot?.querySelector('.dropdown-menu');

    // Initially closed
    expect(dropdownButton?.getAttribute('aria-expanded')).toBe('false');
    expect(dropdownMenu?.getAttribute('data-open')).toBe('false');

    // Click to open
    dropdownButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    expect(dropdownButton?.getAttribute('aria-expanded')).toBe('true');
    expect(dropdownMenu?.getAttribute('data-open')).toBe('true');

    // Click to close
    dropdownButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    expect(dropdownButton?.getAttribute('aria-expanded')).toBe('false');
    expect(dropdownMenu?.getAttribute('data-open')).toBe('false');
  });

  it('should close dropdown after selecting a menu item', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted"]'></proposal-filter>
    `;
    const element = container.querySelector('proposal-filter');
    await element?.updateComplete;

    const dropdownButton = element?.shadowRoot?.querySelector('.dropdown-button');
    const acceptedMenuItem = element?.shadowRoot?.querySelector('.menu-item[data-status="accepted"]');

    // Open dropdown
    dropdownButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    expect(dropdownButton?.getAttribute('aria-expanded')).toBe('true');

    // Select menu item
    acceptedMenuItem?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await element?.updateComplete;

    // Should be closed
    expect(dropdownButton?.getAttribute('aria-expanded')).toBe('false');
  });
});

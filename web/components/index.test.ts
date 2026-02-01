import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import './proposal-filter.js';
import { initializeFilters } from './index.js';

describe('initializeFilters', () => {
  let container: HTMLDivElement;

  beforeEach(() => {
    container = document.createElement('div');
    document.body.appendChild(container);
  });

  afterEach(() => {
    container.remove();
  });

  it('should show all proposals when "all" filter is selected', async () => {
    container.innerHTML = `
      <div class="weekly-index">
        <proposal-filter statuses='["accepted", "declined"]'></proposal-filter>
        <article data-status="accepted">Proposal 1</article>
        <article data-status="declined">Proposal 2</article>
        <article data-status="accepted">Proposal 3</article>
      </div>
    `;

    const filter = container.querySelector('proposal-filter');
    await filter?.updateComplete;

    initializeFilters();

    // Click on "all" button (should already be selected by default, but click to trigger event)
    const allButton = filter?.shadowRoot?.querySelector('button[data-status="all"]');
    allButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await filter?.updateComplete;

    const proposals = container.querySelectorAll('article');
    proposals.forEach((proposal) => {
      expect((proposal as HTMLElement).style.display).toBe('');
    });
  });

  it('should hide non-matching proposals when a status filter is selected', async () => {
    container.innerHTML = `
      <div class="weekly-index">
        <proposal-filter statuses='["accepted", "declined"]'></proposal-filter>
        <article data-status="accepted">Proposal 1</article>
        <article data-status="declined">Proposal 2</article>
        <article data-status="accepted">Proposal 3</article>
      </div>
    `;

    const filter = container.querySelector('proposal-filter');
    await filter?.updateComplete;

    initializeFilters();

    // Click on "accepted" button
    const acceptedButton = filter?.shadowRoot?.querySelector('button[data-status="accepted"]');
    acceptedButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await filter?.updateComplete;

    const proposals = container.querySelectorAll('article');
    expect((proposals[0] as HTMLElement).style.display).toBe(''); // accepted - visible
    expect((proposals[1] as HTMLElement).style.display).toBe('none'); // declined - hidden
    expect((proposals[2] as HTMLElement).style.display).toBe(''); // accepted - visible
  });

  it('should restore all proposals when switching back to "all" filter', async () => {
    container.innerHTML = `
      <div class="weekly-index">
        <proposal-filter statuses='["accepted", "declined"]'></proposal-filter>
        <article data-status="accepted">Proposal 1</article>
        <article data-status="declined">Proposal 2</article>
      </div>
    `;

    const filter = container.querySelector('proposal-filter');
    await filter?.updateComplete;

    initializeFilters();

    // First select "accepted"
    const acceptedButton = filter?.shadowRoot?.querySelector('button[data-status="accepted"]');
    acceptedButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await filter?.updateComplete;

    // Then select "all"
    const allButton = filter?.shadowRoot?.querySelector('button[data-status="all"]');
    allButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await filter?.updateComplete;

    const proposals = container.querySelectorAll('article');
    proposals.forEach((proposal) => {
      expect((proposal as HTMLElement).style.display).toBe('');
    });
  });

  it('should handle multiple filters on the same page independently', async () => {
    container.innerHTML = `
      <div class="weekly-index" id="week1">
        <proposal-filter statuses='["accepted"]'></proposal-filter>
        <article data-status="accepted">Week1 Proposal</article>
      </div>
      <div class="weekly-index" id="week2">
        <proposal-filter statuses='["declined"]'></proposal-filter>
        <article data-status="declined">Week2 Proposal</article>
      </div>
    `;

    const filters = container.querySelectorAll('proposal-filter');
    await Promise.all(Array.from(filters).map((f) => f.updateComplete));

    initializeFilters();

    // Both should be visible initially
    const week1Proposal = container.querySelector('#week1 article') as HTMLElement;
    const week2Proposal = container.querySelector('#week2 article') as HTMLElement;

    expect(week1Proposal.style.display).toBe('');
    expect(week2Proposal.style.display).toBe('');
  });

  it('should warn on null event detail', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    container.innerHTML = `
      <div class="weekly-index">
        <proposal-filter statuses='["accepted"]'></proposal-filter>
        <article data-status="accepted">Proposal</article>
      </div>
    `;

    const filter = container.querySelector('proposal-filter');
    await filter?.updateComplete;

    initializeFilters();

    // Dispatch invalid event with null detail
    filter?.dispatchEvent(
      new CustomEvent('filter-change', {
        detail: null,
        bubbles: true,
      }),
    );

    expect(warnSpy).toHaveBeenCalledWith('Invalid filter-change event detail:', null);
    warnSpy.mockRestore();
  });

  it('should warn and ignore invalid status values', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    container.innerHTML = `
      <div class="weekly-index">
        <proposal-filter statuses='["accepted"]'></proposal-filter>
        <article data-status="accepted">Proposal</article>
      </div>
    `;

    const filter = container.querySelector('proposal-filter');
    await filter?.updateComplete;

    initializeFilters();

    const proposal = container.querySelector('article') as HTMLElement;
    const originalDisplay = proposal.style.display;

    // Dispatch event with invalid status value
    filter?.dispatchEvent(
      new CustomEvent('filter-change', {
        detail: { status: 'invalid_status' },
        bubbles: true,
      }),
    );

    // Should warn about invalid detail
    expect(warnSpy).toHaveBeenCalledWith('Invalid filter-change event detail:', { status: 'invalid_status' });
    // Should not modify proposals
    expect(proposal.style.display).toBe(originalDisplay);
    warnSpy.mockRestore();
  });

  it('should not throw if filter is outside .weekly-index', async () => {
    container.innerHTML = `
      <proposal-filter statuses='["accepted"]'></proposal-filter>
      <article data-status="accepted">Orphan Proposal</article>
    `;

    const filter = container.querySelector('proposal-filter');
    await filter?.updateComplete;

    initializeFilters();

    // Click should not throw
    const acceptedButton = filter?.shadowRoot?.querySelector('button[data-status="accepted"]');
    expect(() => {
      acceptedButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    }).not.toThrow();
  });
});

// ReMemory Shared Utilities
// Common functionality used by both recovery (app.ts) and creation (create-app.ts)

import type { ToastManager, ToastOptions, ToastAction, ToastType } from './types';

// ============================================
// Utility Functions
// ============================================

export function escapeHtml(str: string | null | undefined): string {
  if (str == null) return '';
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

// ============================================
// Toast Notification System
// ============================================

const TOAST_ICONS: Record<ToastType, string> = {
  error: '⚠️',
  warning: '⚡',
  success: '✓',
  info: 'ℹ️'
};

export const toast: ToastManager = {
  container: null,
  backdrop: null,
  errorCount: 0,

  init(): void {
    this.container = document.getElementById('toast-container');
    if (!this.container) {
      this.container = document.createElement('div');
      this.container.id = 'toast-container';
      this.container.className = 'toast-container';
      this.container.setAttribute('role', 'alert');
      this.container.setAttribute('aria-live', 'polite');
      document.body.appendChild(this.container);
    }

    // Create backdrop for error toasts
    this.backdrop = document.getElementById('toast-backdrop');
    if (!this.backdrop) {
      this.backdrop = document.createElement('div');
      this.backdrop.id = 'toast-backdrop';
      this.backdrop.className = 'toast-backdrop';
      document.body.appendChild(this.backdrop);

      // Clicking backdrop dismisses all error toasts
      this.backdrop.addEventListener('click', () => this.dismissAllErrors());
    }
  },

  showBackdrop(): void {
    if (!this.backdrop) this.init();
    this.backdrop?.classList.add('visible');
  },

  hideBackdrop(): void {
    this.backdrop?.classList.remove('visible');
  },

  dismissAllErrors(): void {
    if (!this.container) return;
    const errorToasts = this.container.querySelectorAll('.toast-error');
    errorToasts.forEach(t => this.dismiss(t as HTMLElement));
  },

  show(options: ToastOptions): HTMLElement {
    const {
      type = 'error',
      title,
      message,
      guidance,
      actions,
      duration = 8000
    } = options;

    if (!this.container) this.init();

    const toastEl = document.createElement('div');
    toastEl.className = `toast toast-${type}`;

    let actionsHtml = '';
    if (actions && actions.length > 0) {
      actionsHtml = `<div class="toast-actions">
        ${actions.map(a =>
          `<button class="toast-action ${a.primary ? 'toast-action-primary' : ''}" data-action="${escapeHtml(a.id)}">${escapeHtml(a.label)}</button>`
        ).join('')}
      </div>`;
    }

    toastEl.innerHTML = `
      <span class="toast-icon">${TOAST_ICONS[type]}</span>
      <div class="toast-content">
        ${title ? `<div class="toast-title">${escapeHtml(title)}</div>` : ''}
        <div class="toast-message">${escapeHtml(message)}</div>
        ${guidance ? `<div class="toast-guidance">${escapeHtml(guidance)}</div>` : ''}
        ${actionsHtml}
      </div>
      <button class="toast-close" aria-label="Dismiss">&times;</button>
    `;

    // Add event listeners
    const closeBtn = toastEl.querySelector('.toast-close');
    closeBtn?.addEventListener('click', () => this.dismiss(toastEl));

    if (actions) {
      actions.forEach(action => {
        if (action.onClick) {
          const btn = toastEl.querySelector(`[data-action="${action.id}"]`);
          btn?.addEventListener('click', () => {
            action.onClick?.();
            this.dismiss(toastEl);
          });
        }
      });
    }

    this.container?.appendChild(toastEl);

    // Show backdrop for error toasts
    if (type === 'error') {
      this.errorCount++;
      this.showBackdrop();
    }

    // Auto-dismiss (unless duration is 0)
    if (duration > 0) {
      setTimeout(() => this.dismiss(toastEl), duration);
    }

    return toastEl;
  },

  dismiss(toastEl: HTMLElement): void {
    if (!toastEl || !toastEl.parentNode) return;

    // Track error toast removal for backdrop
    const isError = toastEl.classList.contains('toast-error');

    toastEl.classList.add('toast-exit');
    setTimeout(() => {
      toastEl.remove();

      // Hide backdrop when all error toasts are gone
      if (isError) {
        this.errorCount = Math.max(0, this.errorCount - 1);
        if (this.errorCount === 0) {
          this.hideBackdrop();
        }
      }
    }, 200);
  },

  error(title: string, message: string, guidance?: string, actions?: ToastAction[]): HTMLElement {
    return this.show({ type: 'error', title, message, guidance, actions, duration: 0 });
  },

  warning(title: string, message: string, guidance?: string): HTMLElement {
    return this.show({ type: 'warning', title, message, guidance });
  },

  success(title: string, message: string): HTMLElement {
    return this.show({ type: 'success', title, message, duration: 5000 });
  },

  info(title: string, message: string, guidance?: string): HTMLElement {
    return this.show({ type: 'info', title, message, guidance });
  }
};

// ============================================
// Inline Error Display
// ============================================

export function showInlineError(
  targetElement: HTMLElement,
  message: string,
  guidance?: string
): void {
  // Remove any existing inline error for this element
  clearInlineError(targetElement);

  const errorEl = document.createElement('div');
  errorEl.className = 'inline-error';
  errorEl.dataset.inlineError = 'true';
  errorEl.innerHTML = `
    <span class="inline-error-icon">⚠️</span>
    <div class="inline-error-content">
      <div class="inline-error-message">${escapeHtml(message)}</div>
      ${guidance ? `<div class="inline-error-guidance">${escapeHtml(guidance)}</div>` : ''}
    </div>
    <button class="inline-error-dismiss" aria-label="Dismiss">&times;</button>
  `;

  const dismissBtn = errorEl.querySelector('.inline-error-dismiss');
  dismissBtn?.addEventListener('click', () => {
    errorEl.remove();
    targetElement.classList.remove('has-error');
  });

  targetElement.classList.add('has-error');
  targetElement.parentNode?.insertBefore(errorEl, targetElement.nextSibling);
}

export function clearInlineError(targetElement: HTMLElement): void {
  if (!targetElement?.parentNode) return;
  const existing = targetElement.parentNode.querySelector('[data-inline-error]');
  existing?.remove();
  targetElement.classList.remove('has-error');
}

// ============================================
// WASM Loading
// ============================================

export function waitForWasm(timeoutMs: number = 15000): Promise<void> {
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      reject(new Error('WASM load timed out'));
    }, timeoutMs);
    const check = (): void => {
      if (window.rememoryReady) {
        clearTimeout(timeout);
        resolve();
      } else {
        setTimeout(check, 50);
      }
    };
    check();
  });
}

// ============================================
// Export to global scope for use by app.ts and create-app.ts
// ============================================

window.rememoryUtils = {
  escapeHtml,
  formatSize,
  toast,
  showInlineError,
  clearInlineError,
  waitForWasm
};

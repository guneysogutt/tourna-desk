export interface ConfirmOptions {
  title?: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'danger' | 'warning' | 'info';
}

export function showConfirmModal(options: ConfirmOptions): Promise<boolean> {
  return new Promise((resolve) => {
    const variant = options.variant || 'danger';
    const isTR =
      (typeof document !== 'undefined' &&
        document.documentElement.lang?.toLowerCase().startsWith('tr')) ||
      localStorage.getItem('i18nextLng')?.toLowerCase().startsWith('tr') ||
      true;

    const titleText =
      options.title ||
      (variant === 'danger'
        ? isTR
          ? 'İşlemi Onayla'
          : 'Confirm Action'
        : variant === 'warning'
        ? isTR
          ? 'Uyarı'
          : 'Warning'
        : isTR
        ? 'Bilgi'
        : 'Information');

    const confirmLabel =
      options.confirmText ||
      (variant === 'danger'
        ? isTR
          ? 'Evet, Sil'
          : 'Yes, Delete'
        : isTR
        ? 'Onayla'
        : 'Confirm');

    const cancelLabel = options.cancelText || (isTR ? 'İptal' : 'Cancel');

    // 1. Create backdrop container
    const backdrop = document.createElement('div');
    backdrop.setAttribute('role', 'dialog');
    backdrop.setAttribute('aria-modal', 'true');
    backdrop.className =
      'fixed inset-0 z-[100] flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-xs opacity-0 transition-opacity duration-150 ease-out';

    // 2. Create modal card with scale transition
    const dialog = document.createElement('div');
    dialog.className =
      'w-full max-w-md overflow-hidden rounded-2xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-2xl transform scale-95 opacity-0 transition-all duration-150 ease-out';

    // Variant accent styles
    const iconWrapClasses =
      variant === 'danger'
        ? 'bg-rose-100 text-rose-600 dark:bg-rose-950/60 dark:text-rose-400 ring-1 ring-rose-200 dark:ring-rose-800/60'
        : variant === 'warning'
        ? 'bg-amber-100 text-amber-600 dark:bg-amber-950/60 dark:text-amber-400 ring-1 ring-amber-200 dark:ring-amber-800/60'
        : 'bg-blue-100 text-blue-600 dark:bg-blue-950/60 dark:text-blue-400 ring-1 ring-blue-200 dark:ring-blue-800/60';

    const confirmBtnClasses =
      variant === 'danger'
        ? 'bg-rose-600 hover:bg-rose-700 focus:ring-rose-500 text-white'
        : variant === 'warning'
        ? 'bg-amber-600 hover:bg-amber-700 focus:ring-amber-500 text-white'
        : 'bg-blue-600 hover:bg-blue-700 focus:ring-blue-500 text-white';

    const iconSvg =
      variant === 'info'
        ? `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/></svg>`
        : `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><path d="M12 9v4"/><path d="M12 17h.01"/></svg>`;

    // Body layout
    const body = document.createElement('div');
    body.className = 'p-5 sm:p-6 space-y-4';

    const headerRow = document.createElement('div');
    headerRow.className = 'flex items-start gap-3.5';

    const iconBadge = document.createElement('div');
    iconBadge.className = `flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ${iconWrapClasses}`;
    iconBadge.innerHTML = iconSvg;

    const textCol = document.createElement('div');
    textCol.className = 'space-y-1.5 min-w-0 flex-1';

    const titleEl = document.createElement('h3');
    titleEl.className =
      'text-base font-bold text-slate-900 dark:text-slate-100 leading-snug';
    titleEl.textContent = titleText;

    const msgEl = document.createElement('p');
    msgEl.className =
      'text-xs sm:text-sm text-slate-600 dark:text-slate-400 leading-relaxed';
    msgEl.textContent = options.message;

    textCol.appendChild(titleEl);
    textCol.appendChild(msgEl);

    headerRow.appendChild(iconBadge);
    headerRow.appendChild(textCol);

    // Footer buttons
    const footer = document.createElement('div');
    footer.className =
      'flex items-center justify-end gap-2.5 pt-3 border-t border-slate-100 dark:border-zinc-800';

    const cancelBtn = document.createElement('button');
    cancelBtn.type = 'button';
    cancelBtn.className =
      'inline-flex h-9 items-center justify-center rounded-lg border border-slate-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 px-4 text-xs sm:text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-zinc-700 focus:outline-none focus:ring-2 focus:ring-slate-400 transition-colors';
    cancelBtn.textContent = cancelLabel;

    const confirmBtn = document.createElement('button');
    confirmBtn.type = 'button';
    confirmBtn.className = `inline-flex h-9 items-center justify-center rounded-lg px-4 text-xs sm:text-sm font-semibold shadow-xs focus:outline-none focus:ring-2 transition-colors ${confirmBtnClasses}`;
    confirmBtn.textContent = confirmLabel;

    footer.appendChild(cancelBtn);
    footer.appendChild(confirmBtn);

    body.appendChild(headerRow);
    body.appendChild(footer);
    dialog.appendChild(body);
    backdrop.appendChild(dialog);
    document.body.appendChild(backdrop);

    // Trigger enter animation on next frame
    requestAnimationFrame(() => {
      backdrop.classList.remove('opacity-0');
      backdrop.classList.add('opacity-100');
      dialog.classList.remove('scale-95', 'opacity-0');
      dialog.classList.add('scale-100', 'opacity-100');
      confirmBtn.focus();
    });

    let settled = false;
    const cleanup = (result: boolean) => {
      if (settled) return;
      settled = true;
      document.removeEventListener('keydown', onKeyDown, true);

      backdrop.classList.remove('opacity-100');
      backdrop.classList.add('opacity-0');
      dialog.classList.remove('scale-100', 'opacity-100');
      dialog.classList.add('scale-95', 'opacity-0');

      window.setTimeout(() => {
        if (backdrop.parentNode) {
          backdrop.parentNode.removeChild(backdrop);
        }
        resolve(result);
      }, 150);
    };

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        e.stopPropagation();
        cleanup(false);
      }
    };

    document.addEventListener('keydown', onKeyDown, true);

    backdrop.addEventListener('mousedown', (e: MouseEvent) => {
      if (e.target === backdrop) {
        cleanup(false);
      }
    });

    cancelBtn.addEventListener('click', () => cleanup(false));
    confirmBtn.addEventListener('click', () => cleanup(true));
  });
}

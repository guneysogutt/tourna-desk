import React from 'react';
import { useTranslation } from 'react-i18next';
import { Disc3 } from 'lucide-react';
import { SportSlug } from '../types';

export const getSportBadgeClasses = (sport: SportSlug): string => {
  switch (sport) {
    case 'basketball':
      return 'bg-orange-50 text-orange-600 border-orange-200 dark:bg-orange-950/40 dark:text-orange-400 dark:border-orange-800';
    case 'football':
      return 'bg-emerald-50 text-emerald-600 border-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-400 dark:border-emerald-800';
    case 'futsal':
      return 'bg-blue-50 text-blue-600 border-blue-200 dark:bg-blue-950/40 dark:text-blue-400 dark:border-blue-800';
    case 'handball':
      return 'bg-teal-50 text-teal-600 border-teal-200 dark:bg-teal-950/40 dark:text-teal-400 dark:border-teal-800';
    case 'volleyball':
      return 'bg-amber-50 text-amber-700 border-amber-300 dark:bg-amber-950/40 dark:text-amber-400 dark:border-amber-800';
    default:
      return 'bg-emerald-50 text-emerald-600 border-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-400 dark:border-emerald-800';
  }
};

interface SportIconProps {
  sport: SportSlug;
  className?: string;
}

export const SportIcon: React.FC<SportIconProps> = ({
  sport,
  className = 'h-4 w-4 shrink-0',
}) => {
  switch (sport) {
    case 'basketball':
      // Dedicated Basketball SVG with curved seams
      return (
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={className}
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="10" />
          <path d="M4.93 4.93c4.5 4.5 4.5 9.64 0 14.14" />
          <path d="M19.07 4.93c-4.5 4.5-4.5 9.64 0 14.14" />
          <line x1="2" y1="12" x2="22" y2="12" />
          <line x1="12" y1="2" x2="12" y2="22" />
        </svg>
      );

    case 'football':
      // Dedicated Soccer Ball SVG (center pentagon + stitched seams)
      return (
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={className}
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="10" />
          <polygon points="12 7.5 16.2 10.5 14.6 15.5 9.4 15.5 7.8 10.5" />
          <line x1="12" y1="7.5" x2="12" y2="2" />
          <line x1="16.2" y1="10.5" x2="21.3" y2="8.8" />
          <line x1="14.6" y1="15.5" x2="17.8" y2="20" />
          <line x1="9.4" y1="15.5" x2="6.2" y2="20" />
          <line x1="7.8" y1="10.5" x2="2.7" y2="8.8" />
        </svg>
      );

    case 'futsal':
      // Dedicated Futsal Ball SVG (indoor star/diamond panel ball)
      return (
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={className}
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="10" />
          <polygon points="12 7 17 12 12 17 7 12" />
          <circle cx="12" cy="12" r="1.5" fill="currentColor" />
          <line x1="12" y1="2" x2="12" y2="7" />
          <line x1="12" y1="17" x2="12" y2="22" />
          <line x1="2" y1="12" x2="7" y2="12" />
          <line x1="17" y1="12" x2="22" y2="12" />
        </svg>
      );

    case 'volleyball':
      // Dedicated Volleyball SVG (swirl seams pattern)
      return (
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className={className}
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="10" />
          <path d="M12 12c0-5.5 4.5-10 4.5-10" />
          <path d="M12 12c4.8 2.8 6.5 8.8 6.5 8.8" />
          <path d="M12 12c-4.8 2.8-9.5 1.2-9.5 1.2" />
          <path d="M7.5 4.2c3 1.5 6.5 1.5 9.5 0" />
          <path d="M19.8 9c-1.5 3-3.5 5.5-6.8 7" />
          <path d="M4.2 9c1.5 3.5 4.5 6 7.8 7" />
        </svg>
      );

    case 'handball':
    default:
      return <Disc3 className={className} />;
  }
};

export interface SportBadgeProps {
  sport: SportSlug;
  size?: 'sm' | 'md';
  showLabel?: boolean;
  className?: string;
}

export const SportBadge: React.FC<SportBadgeProps> = ({
  sport,
  size = 'md',
  showLabel = true,
  className = '',
}) => {
  const { t } = useTranslation();
  const colorClasses = getSportBadgeClasses(sport);
  const sizeClasses =
    size === 'sm'
      ? 'px-2 py-0.5 text-[11px] gap-1'
      : 'px-2.5 py-1 text-xs gap-1.5';
  const iconSize = size === 'sm' ? 'h-3 w-3 shrink-0' : 'h-3.5 w-3.5 shrink-0';

  return (
    <span
      className={`inline-flex items-center rounded-lg border font-semibold ${colorClasses} ${sizeClasses} ${className}`}
    >
      <SportIcon sport={sport} className={iconSize} />
      {showLabel && <span>{t(`sports.${sport}`)}</span>}
    </span>
  );
};

import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  MapPin,
  Plus,
  Trash2,
  Edit3,
  X,
  Check,
  Building2,
} from 'lucide-react';
import {
  SportSlug,
  Venue,
  sportCodeToSlug,
  sportSlugToCode,
} from '../types';
import { SportBadge, SportIcon, getSportBadgeClasses } from './SportBadge';

const ALL_SPORT_SLUGS: SportSlug[] = [
  'basketball',
  'football',
  'futsal',
  'handball',
  'volleyball',
];

interface VenueManagerModalProps {
  open: boolean;
  venues: Venue[];
  onClose: () => void;
  onCreateVenue: (payload: {
    name: string;
    address: string;
    supported_sports: string[];
  }) => Promise<void>;
  onUpdateVenue: (
    id: number,
    payload: {
      name: string;
      address: string;
      supported_sports: string[];
    }
  ) => Promise<void>;
  onDeleteVenue: (id: number) => Promise<void>;
}

export const VenueManagerModal: React.FC<VenueManagerModalProps> = ({
  open,
  venues,
  onClose,
  onCreateVenue,
  onUpdateVenue,
  onDeleteVenue,
}) => {
  const { t } = useTranslation();
  const [editingId, setEditingId] = useState<number | null>(null);
  const [name, setName] = useState('');
  const [address, setAddress] = useState('');
  const [selectedSlugs, setSelectedSlugs] = useState<SportSlug[]>([
    'basketball',
    'volleyball',
    'futsal',
  ]);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) {
      setEditingId(null);
      setName('');
      setAddress('');
      setSelectedSlugs(['basketball', 'volleyball', 'futsal']);
    }
  }, [open]);

  if (!open) return null;

  const toggleSport = (slug: SportSlug) => {
    setSelectedSlugs((prev) =>
      prev.includes(slug) ? prev.filter((s) => s !== slug) : [...prev, slug]
    );
  };

  const handleStartEdit = (v: Venue) => {
    setEditingId(v.id);
    setName(v.name);
    setAddress(v.address || '');
    const slugs = (v.supported_sports || []).map((code) => sportCodeToSlug(code));
    setSelectedSlugs(slugs.length > 0 ? slugs : ['basketball']);
  };

  const handleCancelEdit = () => {
    setEditingId(null);
    setName('');
    setAddress('');
    setSelectedSlugs(['basketball', 'volleyball', 'futsal']);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setSubmitting(true);
    try {
      const sportsCodes =
        selectedSlugs.length > 0
          ? selectedSlugs.map((s) => sportSlugToCode(s))
          : ALL_SPORT_SLUGS.map((s) => sportSlugToCode(s));

      if (editingId !== null) {
        await onUpdateVenue(editingId, {
          name: name.trim(),
          address: address.trim(),
          supported_sports: sportsCodes,
        });
      } else {
        await onCreateVenue({
          name: name.trim(),
          address: address.trim(),
          supported_sports: sportsCodes,
        });
      }
      handleCancelEdit();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-xs overflow-y-auto">
      <div className="w-full max-w-2xl overflow-hidden rounded-2xl bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 shadow-2xl my-auto">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 dark:border-zinc-800 bg-slate-900 dark:bg-zinc-950 px-5 py-4 text-white">
          <div className="flex items-center gap-2.5">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-600/20 text-blue-400 ring-1 ring-blue-400/30">
              <Building2 className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-base font-bold">{t('venues.title')}</h3>
              <p className="text-xs text-slate-400">{t('venues.subtitle')}</p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1.5 text-slate-400 hover:bg-slate-800 hover:text-white transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-5 space-y-5 max-h-[80vh] overflow-y-auto">
          {/* Add / Edit Venue Form */}
          <form
            onSubmit={handleSubmit}
            className="rounded-xl border border-slate-200 dark:border-zinc-800 bg-slate-50/70 dark:bg-zinc-800/40 p-4 space-y-3.5"
          >
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-300">
                {editingId !== null ? t('venues.edit_title') : t('venues.add_title')}
              </span>
              {editingId !== null && (
                <button
                  type="button"
                  onClick={handleCancelEdit}
                  className="text-xs font-semibold text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-white"
                >
                  {t('modals.cancel')}
                </button>
              )}
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1">
                  {t('venues.name_label')}
                </label>
                <input
                  type="text"
                  required
                  placeholder={t('venues.name_placeholder')}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-primary focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1">
                  {t('venues.address_label')}
                </label>
                <input
                  type="text"
                  placeholder={t('venues.address_placeholder')}
                  value={address}
                  onChange={(e) => setAddress(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 px-3 py-2 text-sm text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-primary focus:outline-none"
                />
              </div>
            </div>

            {/* Supported Sports Multi-Select Pills */}
            <div>
              <label className="block text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                {t('venues.sports_label')}
              </label>
              <div className="flex flex-wrap items-center gap-2">
                {ALL_SPORT_SLUGS.map((slug) => {
                  const active = selectedSlugs.includes(slug);
                  return (
                    <button
                      key={slug}
                      type="button"
                      onClick={() => toggleSport(slug)}
                      className={`inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs font-semibold transition-all ${
                        active
                          ? `${getSportBadgeClasses(slug)} ring-2 ring-blue-500/30`
                          : 'border-slate-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 text-slate-500 dark:text-slate-400 opacity-70 hover:opacity-100'
                      }`}
                    >
                      <SportIcon sport={slug} className="h-3.5 w-3.5" />
                      <span>{t(`sports.${slug}`)}</span>
                      {active && <Check className="h-3 w-3 ml-0.5" />}
                    </button>
                  );
                })}
              </div>
            </div>

            <div className="flex justify-end pt-1">
              <button
                type="submit"
                disabled={submitting || !name.trim()}
                className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 px-4 text-xs font-semibold text-white disabled:opacity-50 transition"
              >
                {editingId !== null ? (
                  <>
                    <Check className="h-3.5 w-3.5" />
                    {t('modals.save')}
                  </>
                ) : (
                  <>
                    <Plus className="h-3.5 w-3.5" />
                    {t('modals.create')}
                  </>
                )}
              </button>
            </div>
          </form>

          {/* Venues List */}
          <div className="space-y-2.5">
            {venues.length === 0 ? (
              <div className="rounded-xl border border-dashed border-slate-300 dark:border-zinc-700 p-8 text-center text-xs text-slate-500 dark:text-slate-400">
                {t('venues.empty')}
              </div>
            ) : (
              venues.map((v) => (
                <div
                  key={v.id}
                  className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-3.5 shadow-2xs"
                >
                  <div className="space-y-1.5 min-w-0">
                    <div className="flex items-center gap-2">
                      <MapPin className="h-4 w-4 text-blue-600 dark:text-blue-400 shrink-0" />
                      <span className="text-sm font-bold text-slate-900 dark:text-slate-100 truncate">
                        {v.name}
                      </span>
                      {v.address && (
                        <span className="text-xs text-slate-500 dark:text-slate-400 truncate">
                          • {v.address}
                        </span>
                      )}
                    </div>
                    <div className="flex flex-wrap items-center gap-1.5">
                      {(v.supported_sports || []).map((spCode) => {
                        const slug = sportCodeToSlug(spCode);
                        return (
                          <SportBadge key={`${v.id}-${spCode}`} sport={slug} size="sm" />
                        );
                      })}
                    </div>
                  </div>

                  <div className="flex items-center gap-1.5 shrink-0 self-end sm:self-center">
                    <button
                      type="button"
                      onClick={() => handleStartEdit(v)}
                      title={t('venues.edit_title')}
                      className="rounded-lg border border-slate-200 dark:border-zinc-700 p-1.5 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-zinc-800 transition-colors"
                    >
                      <Edit3 className="h-3.5 w-3.5" />
                    </button>
                    <button
                      type="button"
                      onClick={() => onDeleteVenue(v.id)}
                      title={t('modals.delete')}
                      className="rounded-lg border border-slate-200 dark:border-zinc-700 p-1.5 text-slate-400 hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/50 transition-colors"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        <div className="flex justify-end border-t border-slate-200 dark:border-zinc-800 px-5 py-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-slate-300 dark:border-zinc-700 px-4 py-1.5 text-xs font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-zinc-800"
          >
            Kapat
          </button>
        </div>
      </div>
    </div>
  );
};

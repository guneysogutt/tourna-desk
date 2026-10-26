import React, { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Star,
  Shuffle,
  X,
  Search,
  Sparkles,
  LayoutGrid,
  GitBranch,
} from 'lucide-react';
import { SportSlug, Team, Venue, venueSupportsSport } from '../types';
import { calculateBracketMath } from './BracketTree';

interface SeedingModalProps {
  open: boolean;
  teams: Team[];
  sportSlug?: SportSlug;
  venues?: Venue[];
  onClose: () => void;
  onRunInitialDraw: (payload: {
    format?: 'group' | 'knockout';
    seeded_team_ids: number[];
    generate_fixtures: boolean;
    double_round_robin: boolean;
    default_venue: string;
    default_court: string;
    start_date: string;
  }) => Promise<void>;
}

export const SeedingModal: React.FC<SeedingModalProps> = ({
  open,
  teams,
  sportSlug,
  venues = [],
  onClose,
  onRunInitialDraw,
}) => {
  const { t } = useTranslation();
  const [drawFormat, setDrawFormat] = useState<'group' | 'knockout'>('group');
  const [selectedSeeds, setSelectedSeeds] = useState<number[]>([]);
  const [search, setSearch] = useState('');
  const [doubleRR, setDoubleRR] = useState(false);
  const [selectedVenue, setSelectedVenue] = useState('');
  const [startDate, setStartDate] = useState('2026-10-20');
  const [submitting, setSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const matchingVenues = useMemo(
    () => venues.filter((v) => venueSupportsSport(v, sportSlug)),
    [venues, sportSlug]
  );

  useEffect(() => {
    if (open) {
      if (matchingVenues.length > 0 && !selectedVenue) {
        setSelectedVenue(matchingVenues[0].name);
      } else if (matchingVenues.length === 0) {
        setSelectedVenue('');
      }
    }
  }, [open, matchingVenues]);

  const canUseKnockout = teams.length >= 2;
  const bracketMath = useMemo(() => calculateBracketMath(teams.length), [teams.length]);
  const maxSeeds =
    drawFormat === 'knockout'
      ? Math.max(1, Math.floor(bracketMath.bracketSize / 2))
      : Math.max(1, Math.ceil(teams.length / 4));

  useEffect(() => {
    if (!open) return;
    const preSeeded = teams.filter((tm) => tm.is_seeded).map((tm) => tm.id);
    if (preSeeded.length > 0) {
      setSelectedSeeds(preSeeded);
    } else {
      setSelectedSeeds([]);
    }
    setErrorMsg(null);
  }, [open, teams]);

  const filteredTeams = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return teams;
    return teams.filter((tm) => tm.school_name.toLowerCase().includes(q));
  }, [teams, search]);

  if (!open) return null;

  const toggleSeed = (teamId: number) => {
    setErrorMsg(null);
    if (selectedSeeds.includes(teamId)) {
      setSelectedSeeds(selectedSeeds.filter((id) => id !== teamId));
    } else {
      if (selectedSeeds.length >= maxSeeds) {
        setSelectedSeeds([...selectedSeeds.slice(1), teamId]);
      } else {
        setSelectedSeeds([...selectedSeeds, teamId]);
      }
    }
  };

  const expectedGroups = Math.max(
    1,
    Math.ceil(teams.length / 4),
    selectedSeeds.length
  );

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (drawFormat === 'knockout' && !canUseKnockout) {
      setErrorMsg(t('seeding.knockout_size_warning', { count: teams.length }));
      return;
    }
    setSubmitting(true);
    setErrorMsg(null);
    try {
      await onRunInitialDraw({
        format: drawFormat,
        seeded_team_ids: selectedSeeds,
        generate_fixtures: true,
        double_round_robin: doubleRR,
        default_venue: selectedVenue,
        default_court: '',
        start_date: startDate,
      });
      onClose();
    } catch (err: any) {
      setErrorMsg(err?.message || 'Draw failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-xs overflow-y-auto">
      <div className="w-full max-w-2xl overflow-hidden rounded-2xl bg-white shadow-2xl ring-1 ring-slate-200 my-auto dark:bg-slate-900 dark:ring-slate-800">
        <div className="flex items-center justify-between border-b border-slate-200 bg-slate-900 px-5 py-4 text-white dark:border-slate-800 dark:bg-slate-950">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-amber-500/20 text-amber-400 ring-1 ring-amber-400/30">
              <Star className="h-5 w-5 fill-amber-400" />
            </div>
            <div>
              <h3 className="text-base font-bold">{t('seeding.modal_title')}</h3>
              <p className="text-xs text-slate-400">{t('seeding.subtitle')}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-slate-400 hover:bg-slate-800 hover:text-white transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          {errorMsg && (
            <div className="rounded-lg border border-rose-200 bg-rose-50 px-3.5 py-2 text-xs font-semibold text-rose-700 dark:border-rose-800 dark:bg-rose-950/60 dark:text-rose-200">
              {errorMsg}
            </div>
          )}

          {/* Format Selection: Group Play vs Optional Pure Knockout */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <button
              type="button"
              onClick={() => {
                setDrawFormat('group');
                setErrorMsg(null);
              }}
              className={`flex items-center gap-2.5 rounded-xl border p-3.5 text-left transition-all ${
                drawFormat === 'group'
                  ? 'border-blue-600 bg-blue-50/80 text-blue-950 ring-2 ring-blue-500/20 dark:border-blue-500 dark:bg-blue-950/50 dark:text-blue-100'
                  : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-800/60 dark:text-slate-300'
              }`}
            >
              <LayoutGrid className="h-5 w-5 shrink-0 text-blue-600 dark:text-blue-400" />
              <div>
                <div className="text-xs font-extrabold">{t('seeding.format_group')}</div>
                <div className="text-[11px] opacity-80">
                  {expectedGroups} Grup • Resmi Anahtar (1-4, 2-3)
                </div>
              </div>
            </button>

            <button
              type="button"
              onClick={() => {
                setDrawFormat('knockout');
                setErrorMsg(null);
              }}
              className={`flex items-center gap-2.5 rounded-xl border p-3.5 text-left transition-all ${
                drawFormat === 'knockout'
                  ? 'border-amber-500 bg-amber-50/80 text-amber-950 ring-2 ring-amber-500/20 dark:border-amber-500 dark:bg-amber-950/50 dark:text-amber-100'
                  : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-800/60 dark:text-slate-300'
              }`}
            >
              <GitBranch className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
              <div>
                <div className="text-xs font-extrabold">{t('seeding.format_knockout')}</div>
                <div className="text-[11px] opacity-80">
                  {canUseKnockout
                    ? `${teams.length} Takım → ${bracketMath.bracketSize}'li Şema (${bracketMath.byeCount} BAY)`
                    : `En az 2 takım gerekir (Mevcut: ${teams.length})`}
                </div>
              </div>
            </button>
          </div>

          {drawFormat === 'knockout' && !canUseKnockout && (
            <div className="rounded-xl border border-amber-300 bg-amber-50 px-3.5 py-2 text-xs font-bold text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200">
              {t('seeding.knockout_size_warning', { count: teams.length })}
            </div>
          )}

          {/* Status Bar */}
          <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-amber-200 bg-amber-50/70 px-4 py-2.5 dark:border-amber-800/60 dark:bg-amber-950/30">
            <div className="flex items-center gap-2">
              <Star className="h-4 w-4 text-amber-600 fill-amber-500" />
              <span className="text-xs font-bold text-amber-900 dark:text-amber-200">
                {t('seeding.selected_count', {
                  count: selectedSeeds.length,
                  target: maxSeeds,
                })}
              </span>
            </div>
            <div className="text-xs font-semibold text-slate-700 dark:text-slate-300">
              {teams.length} {t('teams.team_count')} →{' '}
              {drawFormat === 'knockout' ? (
                <span className="font-extrabold text-amber-700 dark:text-amber-400">
                  {bracketMath.bracketSize}'li Eleme Ağacı ({bracketMath.byeCount} BAY • {bracketMath.totalMatches} Maç)
                </span>
              ) : (
                <span className="font-extrabold text-indigo-700 dark:text-indigo-400">
                  {expectedGroups} Grup (Grup A
                  {expectedGroups > 1
                    ? ` .. Grup ${String.fromCharCode(64 + expectedGroups)}`
                    : ''}
                  )
                </span>
              )}
            </div>
          </div>

          {/* Search input */}
          <div className="relative">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
            <input
              type="text"
              placeholder={t('seeding.search_placeholder')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-xl border border-slate-300 bg-white pl-9 pr-3 py-2 text-sm text-slate-900 focus:border-blue-600 focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </div>

          {/* Multi-select Team List */}
          <div className="max-h-56 overflow-y-auto divide-y divide-slate-100 rounded-xl border border-slate-200 bg-slate-50/50 dark:divide-slate-800 dark:border-slate-800 dark:bg-slate-950/50">
            {filteredTeams.map((tm, idx) => {
              const seedOrder = selectedSeeds.indexOf(tm.id);
              const isSeeded = seedOrder !== -1;

              return (
                <div
                  key={tm.id}
                  onClick={() => toggleSeed(tm.id)}
                  className={`flex cursor-pointer items-center justify-between px-4 py-2.5 transition-colors ${
                    isSeeded
                      ? 'bg-amber-50/90 hover:bg-amber-100/70 dark:bg-amber-950/40 dark:hover:bg-amber-950/60'
                      : 'bg-white hover:bg-slate-50 dark:bg-slate-900 dark:hover:bg-slate-800/60'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <span
                      className={`flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold ${
                        isSeeded
                          ? 'bg-amber-500 text-white shadow-2xs'
                          : 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300'
                      }`}
                    >
                      {isSeeded ? seedOrder + 1 : idx + 1}
                    </span>
                    <div className="text-sm font-bold text-slate-900 dark:text-slate-100">
                      {tm.school_name}
                    </div>
                  </div>

                  <div>
                    {isSeeded ? (
                      <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2.5 py-1 text-[11px] font-extrabold text-amber-800 ring-1 ring-amber-300 dark:bg-amber-500/20 dark:text-amber-300 dark:ring-amber-500/40">
                        <Star className="h-3 w-3 fill-amber-500 text-amber-600 dark:text-amber-400" />
                        {t('seeding.seed_badge')} #{seedOrder + 1} (1. Sıra)
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-2.5 py-1 text-xs font-semibold text-slate-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-400">
                        Seri Başı Seç
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* Fixture Generation Options (Unified Single-Step Workflow) */}
          <div className="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-800/50">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="text-xs font-bold text-slate-800 dark:text-slate-200 flex items-center gap-2">
                <Sparkles className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                {t('seeding.auto_fixture_label')}
              </div>

              {drawFormat === 'group' && (
                <label className="inline-flex items-center gap-2 text-xs font-semibold text-slate-700 dark:text-slate-300 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={doubleRR}
                    onChange={(e) => setDoubleRR(e.target.checked)}
                    className="h-4 w-4 rounded border-slate-300 text-blue-600"
                  />
                  {t('modals.double_round_robin')}
                </label>
              )}
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
              <div>
                <label className="block text-[11px] font-bold uppercase text-slate-500 dark:text-slate-400 mb-1">
                  {t('venues.select_venue')}
                </label>
                <select
                  value={selectedVenue}
                  onChange={(e) => setSelectedVenue(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-900 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
                >
                  <option value="">{t('venues.no_venue_selected')}</option>
                  {matchingVenues.map((v) => (
                    <option key={v.id} value={v.name}>
                      {v.name}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-[11px] font-bold uppercase text-slate-500 dark:text-slate-400 mb-1">
                  {t('modals.start_date')}
                </label>
                <input
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-xs text-slate-900 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
                />
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-200 dark:border-slate-800">
            <button
              type="button"
              onClick={onClose}
              className="inline-flex items-center gap-2 rounded-xl border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800 transition-colors"
            >
              {t('modals.cancel')}
            </button>
            <button
              type="submit"
              disabled={
                submitting ||
                teams.length < 2 ||
                (drawFormat === 'knockout' && !canUseKnockout)
              }
              className="inline-flex items-center gap-2 rounded-xl bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              <Shuffle className="h-4 w-4" />
              {submitting ? t('seeding.drawing') : t('seeding.run_draw_btn')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

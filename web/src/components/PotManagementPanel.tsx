import React, { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Layers,
  Shuffle,
  ArrowLeftRight,
  Trash2,
  RefreshCw,
  ShieldAlert,
  Trophy,
  GripVertical,
  GitBranch,
} from 'lucide-react';
import { PotAssignment, SportSlug, Stage, Team, Venue, venueSupportsSport } from '../types';

interface EditablePotItem {
  team_id: number;
  team_name: string;
  school_name: string;
  stage_id: number | null;
  stage_name: string;
  pot_number: 1 | 2;
}

interface PotManagementPanelProps {
  stages: Stage[];
  teams: Team[];
  potAssignments: PotAssignment[];
  currentRound: number;
  sportSlug?: SportSlug;
  venues?: Venue[];
  onPreparePots: (stageIds?: number[]) => Promise<void>;
  onSavePots: (
    pot1: { team_id: number; stage_id: number | null }[],
    pot2: { team_id: number; stage_id: number | null }[]
  ) => Promise<void>;
  onExecuteNextRoundDraw: (payload: {
    format?: 'group' | 'knockout';
    pot1_team_ids: number[];
    pot2_team_ids: number[];
    generate_fixtures: boolean;
    default_venue: string;
    default_court: string;
    start_date: string;
  }) => Promise<void>;
  onBuildFinalFour: (payload: {
    group_a_stage_id: number;
    group_b_stage_id: number;
    default_venue: string;
    default_court: string;
    start_date: string;
  }) => Promise<void>;
}

export const PotManagementPanel: React.FC<PotManagementPanelProps> = ({
  stages,
  teams,
  potAssignments,
  currentRound,
  sportSlug,
  venues = [],
  onPreparePots,
  onSavePots,
  onExecuteNextRoundDraw,
  onBuildFinalFour,
}) => {
  const { t } = useTranslation();
  const [items, setItems] = useState<EditablePotItem[]>([]);
  const [wildcardTeamId, setWildcardTeamId] = useState<number | ''>('');
  const [draggedTeamId, setDraggedTeamId] = useState<number | null>(null);
  const matchingVenues = useMemo(
    () => venues.filter((v) => venueSupportsSport(v, sportSlug)),
    [venues, sportSlug]
  );
  const [selectedVenue, setSelectedVenue] = useState('');
  const [startDate, setStartDate] = useState('2026-10-24');
  const [busy, setBusy] = useState(false);
  const [banner, setBanner] = useState<{ type: 'info' | 'error'; text: string } | null>(null);

  useEffect(() => {
    if (matchingVenues.length > 0 && !selectedVenue) {
      setSelectedVenue(matchingVenues[0].name);
    }
  }, [matchingVenues]);

  useEffect(() => {
    const mapped: EditablePotItem[] = potAssignments.map((pa) => ({
      team_id: pa.team_id,
      team_name: pa.school_name || pa.team_name,
      school_name: pa.school_name,
      stage_id: pa.stage_id ?? null,
      stage_name: pa.stage_name || '',
      pot_number: pa.pot_number,
    }));
    setItems(mapped);
  }, [potAssignments]);

  const pot1Items = useMemo(() => items.filter((i) => i.pot_number === 1), [items]);
  const pot2Items = useMemo(() => items.filter((i) => i.pot_number === 2), [items]);

  const availableWildcardTeams = useMemo(() => {
    const inPots = new Set(items.map((i) => i.team_id));
    return teams.filter((tm) => !inPots.has(tm.id));
  }, [teams, items]);

  const latestGroupStages = useMemo(() => {
    const groupStages = stages.filter((s) => s.stage_type === 'group');
    if (groupStages.length === 0) return [];
    const maxR = Math.max(...groupStages.map((s) => s.round_number || 1));
    return groupStages.filter((s) => (s.round_number || 1) === maxR);
  }, [stages]);

  const handleAutoPrepare = async () => {
    setBusy(true);
    setBanner(null);
    try {
      await onPreparePots();
      setBanner({
        type: 'info',
        text: "Puan durumuna göre grup 1.'leri 1. Torbaya, grup 2.'leri 2. Torbaya aktarıldı.",
      });
    } catch (err: any) {
      setBanner({ type: 'error', text: err?.message || 'Failed to prepare pots' });
    } finally {
      setBusy(false);
    }
  };

  const persistList = async (updated: EditablePotItem[]) => {
    setItems(updated);
    const p1 = updated
      .filter((i) => i.pot_number === 1)
      .map((i) => ({ team_id: i.team_id, stage_id: i.stage_id }));
    const p2 = updated
      .filter((i) => i.pot_number === 2)
      .map((i) => ({ team_id: i.team_id, stage_id: i.stage_id }));
    await onSavePots(p1, p2);
  };

  const handleSwapPot = async (teamId: number) => {
    const updated = items.map((item) =>
      item.team_id === teamId
        ? { ...item, pot_number: (item.pot_number === 1 ? 2 : 1) as 1 | 2 }
        : item
    );
    await persistList(updated);
  };

  const handleRemoveFromPot = async (teamId: number) => {
    const updated = items.filter((item) => item.team_id !== teamId);
    await persistList(updated);
  };

  const handleAddWildcard = async (targetPot: 1 | 2) => {
    if (!wildcardTeamId) return;
    const tm = teams.find((t) => t.id === Number(wildcardTeamId));
    if (!tm) return;
    const newItem: EditablePotItem = {
      team_id: tm.id,
      team_name: tm.school_name,
      school_name: tm.school_name,
      stage_id: null,
      stage_name: '',
      pot_number: targetPot,
    };
    const updated = [...items, newItem];
    setWildcardTeamId('');
    await persistList(updated);
  };

  const handleDropToPot = async (targetPot: 1 | 2) => {
    if (draggedTeamId === null) return;
    const updated = items.map((item) =>
      item.team_id === draggedTeamId ? { ...item, pot_number: targetPot } : item
    );
    setDraggedTeamId(null);
    await persistList(updated);
  };

  const handleExecuteDraw = async (format: 'group' | 'knockout' = 'group') => {
    if (items.length < 2) return;
    setBusy(true);
    setBanner(null);
    try {
      const p1 = pot1Items.map((i) => ({ team_id: i.team_id, stage_id: i.stage_id }));
      const p2 = pot2Items.map((i) => ({ team_id: i.team_id, stage_id: i.stage_id }));
      await onSavePots(p1, p2);

      await onExecuteNextRoundDraw({
        format,
        pot1_team_ids: pot1Items.map((i) => i.team_id),
        pot2_team_ids: pot2Items.map((i) => i.team_id),
        generate_fixtures: true,
        default_venue: selectedVenue,
        default_court: '',
        start_date: startDate,
      });
    } catch (err: any) {
      setBanner({ type: 'error', text: err?.message || 'Kura çekimi başarısız oldu' });
    } finally {
      setBusy(false);
    }
  };

  const handleQuickFinalFour = async () => {
    if (latestGroupStages.length < 2) return;
    setBusy(true);
    setBanner(null);
    try {
      await onBuildFinalFour({
        group_a_stage_id: latestGroupStages[0].id,
        group_b_stage_id: latestGroupStages[1].id,
        default_venue: selectedVenue,
        default_court: '',
        start_date: startDate,
      });
    } catch (err: any) {
      setBanner({ type: 'error', text: err?.message || 'Final Four oluşturulamadı' });
    } finally {
      setBusy(false);
    }
  };

  const totalPotTeams = pot1Items.length + pot2Items.length;
  const expectedNewGroups = Math.max(1, Math.ceil(totalPotTeams / 4));
  const canDrawKnockout = totalPotTeams >= 2;

  return (
    <div className="space-y-5">
      {/* Header & Lifecycle Actions */}
      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-xs space-y-4 dark:border-slate-800 dark:bg-slate-900">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <Layers className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
              <h3 className="text-base font-extrabold text-slate-900 dark:text-slate-100">
                {t('pots.title')}
              </h3>
              <span className="rounded-full bg-indigo-50 px-2.5 py-0.5 text-xs font-bold text-indigo-700 ring-1 ring-indigo-200 dark:bg-indigo-950/60 dark:text-indigo-300 dark:ring-indigo-800">
                {t('fixtures.stage_round')}: {currentRound}
              </span>
            </div>
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
              {t('pots.subtitle')}
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={handleAutoPrepare}
              className="inline-flex items-center gap-1.5 rounded-xl bg-slate-900 px-3.5 py-2 text-xs font-bold text-white hover:bg-slate-800 disabled:opacity-50 dark:bg-indigo-600 dark:hover:bg-indigo-700"
            >
              <RefreshCw className="h-3.5 w-3.5" />
              {t('pots.prepare_btn')}
            </button>
          </div>
        </div>

        {/* Constraint Banner */}
        <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-blue-200 bg-blue-50/70 px-3.5 py-2.5 text-xs text-blue-900 dark:border-blue-800/60 dark:bg-blue-950/40 dark:text-blue-200">
          <div className="flex items-center gap-2 font-semibold">
            <ShieldAlert className="h-4 w-4 shrink-0 text-blue-600 dark:text-blue-400" />
            <span>{t('pots.constraint_notice')}</span>
          </div>
          <span className="text-[11px] font-medium text-blue-700 dark:text-blue-300">
            {t('pots.drag_hint')}
          </span>
        </div>

        {banner && (
          <div
            className={`rounded-xl border px-3.5 py-2.5 text-xs font-semibold ${
              banner.type === 'error'
                ? 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-800 dark:bg-rose-950/50 dark:text-rose-200'
                : 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-200'
            }`}
          >
            {banner.text}
          </div>
        )}
      </div>

      {/* Two-Column Editable Pots (1. Torba & 2. Torba) */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
        {/* POT 1 */}
        <div
          onDragOver={(e) => e.preventDefault()}
          onDrop={() => handleDropToPot(1)}
          className="flex flex-col overflow-hidden rounded-2xl border-2 border-amber-200 bg-white shadow-sm dark:border-amber-800/60 dark:bg-slate-900"
        >
          <div className="flex items-center justify-between border-b border-amber-200 bg-gradient-to-r from-amber-500 to-amber-600 px-4 py-3 text-white dark:border-amber-800/60">
            <div className="flex items-center gap-2">
              <Trophy className="h-4 w-4 text-amber-100" />
              <span className="text-sm font-extrabold">{t('pots.pot1_title')}</span>
            </div>
            <span className="rounded-full bg-amber-900/30 px-2.5 py-0.5 text-xs font-bold">
              {pot1Items.length} {t('teams.team_count')}
            </span>
          </div>

          <div className="flex-1 divide-y divide-slate-100 p-2 min-h-[12rem] dark:divide-slate-800">
            {pot1Items.length === 0 ? (
              <div className="flex h-full items-center justify-center p-6 text-center text-xs text-slate-400">
                {t('pots.empty_pot')}
              </div>
            ) : (
              pot1Items.map((item, idx) => (
                <div
                  key={item.team_id}
                  draggable
                  onDragStart={() => setDraggedTeamId(item.team_id)}
                  onDragEnd={() => setDraggedTeamId(null)}
                  className="flex items-center justify-between gap-2 rounded-xl px-3 py-2.5 hover:bg-amber-50/50 transition-colors cursor-grab active:cursor-grabbing dark:hover:bg-slate-800/70"
                >
                  <div className="flex items-center gap-2.5 min-w-0">
                    <GripVertical className="h-4 w-4 shrink-0 text-slate-400" />
                    <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-amber-100 text-xs font-extrabold text-amber-800 dark:bg-amber-500/20 dark:text-amber-300">
                      {idx + 1}
                    </span>
                    <div className="min-w-0">
                      <div className="truncate text-sm font-bold text-slate-900 dark:text-slate-100">
                        {item.school_name || item.team_name}
                      </div>
                      <div className="flex items-center gap-1.5 mt-0.5">
                        {item.stage_name ? (
                          <span className="inline-block rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                            {t('pots.source_group', { stage: item.stage_name })}
                          </span>
                        ) : (
                          <span className="inline-block rounded bg-purple-100 px-1.5 py-0.5 text-[10px] font-semibold text-purple-700 dark:bg-purple-950/60 dark:text-purple-300">
                            {t('pots.wildcard_badge')}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-1 shrink-0">
                    <button
                      type="button"
                      onClick={() => handleSwapPot(item.team_id)}
                      title={t('pots.move_to_pot2')}
                      className="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-2 py-1 text-[11px] font-bold text-slate-700 hover:border-blue-400 hover:text-blue-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200"
                    >
                      <ArrowLeftRight className="h-3 w-3" />
                      2. Torba
                    </button>
                    <button
                      type="button"
                      onClick={() => handleRemoveFromPot(item.team_id)}
                      title={t('pots.remove_from_pot')}
                      className="rounded-lg p-1.5 text-slate-400 hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/50"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* POT 2 */}
        <div
          onDragOver={(e) => e.preventDefault()}
          onDrop={() => handleDropToPot(2)}
          className="flex flex-col overflow-hidden rounded-2xl border-2 border-blue-200 bg-white shadow-sm dark:border-blue-800/60 dark:bg-slate-900"
        >
          <div className="flex items-center justify-between border-b border-blue-200 bg-gradient-to-r from-blue-600 to-indigo-600 px-4 py-3 text-white dark:border-blue-800/60">
            <div className="flex items-center gap-2">
              <Layers className="h-4 w-4 text-blue-100" />
              <span className="text-sm font-extrabold">{t('pots.pot2_title')}</span>
            </div>
            <span className="rounded-full bg-blue-950/30 px-2.5 py-0.5 text-xs font-bold">
              {pot2Items.length} {t('teams.team_count')}
            </span>
          </div>

          <div className="flex-1 divide-y divide-slate-100 p-2 min-h-[12rem] dark:divide-slate-800">
            {pot2Items.length === 0 ? (
              <div className="flex h-full items-center justify-center p-6 text-center text-xs text-slate-400">
                {t('pots.empty_pot')}
              </div>
            ) : (
              pot2Items.map((item, idx) => (
                <div
                  key={item.team_id}
                  draggable
                  onDragStart={() => setDraggedTeamId(item.team_id)}
                  onDragEnd={() => setDraggedTeamId(null)}
                  className="flex items-center justify-between gap-2 rounded-xl px-3 py-2.5 hover:bg-blue-50/50 transition-colors cursor-grab active:cursor-grabbing dark:hover:bg-slate-800/70"
                >
                  <div className="flex items-center gap-2.5 min-w-0">
                    <GripVertical className="h-4 w-4 shrink-0 text-slate-400" />
                    <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-blue-100 text-xs font-extrabold text-blue-800 dark:bg-blue-500/20 dark:text-blue-300">
                      {idx + 1}
                    </span>
                    <div className="min-w-0">
                      <div className="truncate text-sm font-bold text-slate-900 dark:text-slate-100">
                        {item.school_name || item.team_name}
                      </div>
                      <div className="flex items-center gap-1.5 mt-0.5">
                        {item.stage_name ? (
                          <span className="inline-block rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                            {t('pots.source_group', { stage: item.stage_name })}
                          </span>
                        ) : (
                          <span className="inline-block rounded bg-purple-100 px-1.5 py-0.5 text-[10px] font-semibold text-purple-700 dark:bg-purple-950/60 dark:text-purple-300">
                            {t('pots.wildcard_badge')}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-1 shrink-0">
                    <button
                      type="button"
                      onClick={() => handleSwapPot(item.team_id)}
                      title={t('pots.move_to_pot1')}
                      className="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-2 py-1 text-[11px] font-bold text-slate-700 hover:border-amber-400 hover:text-amber-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200"
                    >
                      <ArrowLeftRight className="h-3 w-3" />
                      1. Torba
                    </button>
                    <button
                      type="button"
                      onClick={() => handleRemoveFromPot(item.team_id)}
                      title={t('pots.remove_from_pot')}
                      className="rounded-lg p-1.5 text-slate-400 hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/50"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Wildcard / Manual Team Addition & Draw Execution Bar */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Add Wild-card / Bypass Team */}
        <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-xs space-y-3 dark:border-slate-800 dark:bg-slate-900">
          <div className="text-xs font-extrabold uppercase tracking-wider text-slate-600 dark:text-slate-400">
            {t('pots.add_wildcard_title')}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <select
              value={wildcardTeamId}
              onChange={(e) =>
                setWildcardTeamId(e.target.value === '' ? '' : Number(e.target.value))
              }
              className="flex-1 min-w-[12rem] rounded-xl border border-slate-300 bg-white px-3 py-2 text-xs font-semibold text-slate-800 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            >
              <option value="">{t('pots.select_team_placeholder')}</option>
              {availableWildcardTeams.map((tm) => (
                <option key={tm.id} value={tm.id}>
                  {tm.school_name}
                </option>
              ))}
            </select>

            <button
              type="button"
              disabled={!wildcardTeamId}
              onClick={() => handleAddWildcard(1)}
              className="inline-flex items-center gap-1 rounded-xl bg-amber-500 px-3 py-2 text-xs font-bold text-white hover:bg-amber-600 disabled:opacity-40"
            >
              {t('pots.add_to_pot1')}
            </button>
            <button
              type="button"
              disabled={!wildcardTeamId}
              onClick={() => handleAddWildcard(2)}
              className="inline-flex items-center gap-1 rounded-xl bg-blue-600 px-3 py-2 text-xs font-bold text-white hover:bg-blue-700 disabled:opacity-40"
            >
              {t('pots.add_to_pot2')}
            </button>
          </div>
        </div>

        {/* Execute Next Round Draw (Group or Knockout) or Final Four */}
        <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-xs flex flex-col justify-between gap-3 dark:border-slate-800 dark:bg-slate-900">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
            <div>
              <label className="block text-[10px] font-bold uppercase text-slate-500 dark:text-slate-400 mb-1">
                {t('venues.select_venue')}
              </label>
              <select
                value={selectedVenue}
                onChange={(e) => setSelectedVenue(e.target.value)}
                className="w-full rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
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
              <label className="block text-[10px] font-bold uppercase text-slate-500 dark:text-slate-400 mb-1">
                {t('modals.start_date')}
              </label>
              <input
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="w-full rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-xs text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              />
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-end gap-2">
            {latestGroupStages.length === 2 && (
              <button
                type="button"
                disabled={busy}
                onClick={handleQuickFinalFour}
                className="inline-flex items-center gap-1.5 rounded-xl bg-amber-600 px-3.5 py-2.5 text-xs font-extrabold text-white shadow-sm hover:bg-amber-700 disabled:opacity-50"
              >
                <Trophy className="h-4 w-4" />
                {t('final_four.build_btn')}
              </button>
            )}

            {canDrawKnockout && (
              <button
                type="button"
                disabled={busy}
                onClick={() => handleExecuteDraw('knockout')}
                className="inline-flex items-center gap-1.5 rounded-xl bg-emerald-600 px-3.5 py-2.5 text-xs font-extrabold text-white shadow-sm hover:bg-emerald-700 disabled:opacity-50"
              >
                <GitBranch className="h-4 w-4" />
                {t('pots.execute_knockout_btn')} ({totalPotTeams} Takım)
              </button>
            )}

            <button
              type="button"
              disabled={busy || totalPotTeams < 2}
              onClick={() => handleExecuteDraw('group')}
              className="inline-flex items-center gap-1.5 rounded-xl bg-indigo-600 px-4 py-2.5 text-xs font-extrabold text-white shadow-sm hover:bg-indigo-700 disabled:opacity-50"
            >
              <Shuffle className="h-4 w-4" />
              {t('pots.execute_draw_btn')} ({totalPotTeams} Takım → {expectedNewGroups}{' '}
              Grup)
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

import React, { useState, useMemo, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Calendar,
  MapPin,
  Edit3,
  CheckCircle,
  Clock,
  Sparkles,
  Gavel,
  Coffee,
  ArrowLeftRight,
} from 'lucide-react';
import { Match, Stage } from '../types';

interface FixturePanelProps {
  stages: Stage[];
  matches: Match[];
  selectedStageFilter?: number | 'all';
  onSelectMatch: (match: Match) => void;
  onSwapMatchPairings?: (payload: {
    match_a_id: number;
    slot_a: 'home' | 'away';
    match_b_id: number;
    slot_b: 'home' | 'away';
  }) => Promise<void>;
  onOpenSeedingModal?: () => void;
}

export const FixturePanel: React.FC<FixturePanelProps> = ({
  stages,
  matches,
  selectedStageFilter,
  onSelectMatch,
  onSwapMatchPairings,
  onOpenSeedingModal,
}) => {
  const { t } = useTranslation();
  const [selectedStageId, setSelectedStageId] = useState<number | 'all'>('all');
  const [selectedRound, setSelectedRound] = useState<number | 'all'>(1);

  const [editPairingsMode, setEditPairingsMode] = useState(false);
  const [selectedSwapSlot, setSelectedSwapSlot] = useState<{
    matchId: number;
    slot: 'home' | 'away';
    teamName: string;
  } | null>(null);

  useEffect(() => {
    if (selectedStageFilter !== undefined) {
      setSelectedStageId(selectedStageFilter);
    }
  }, [selectedStageFilter]);

  const stageScopedMatches = useMemo(() => {
    const allowedStageIds = new Set(stages.map((s) => s.id));
    return matches.filter((m) => {
      if (!allowedStageIds.has(m.stage_id)) return false;
      if (selectedStageId !== 'all' && m.stage_id !== selectedStageId) return false;
      return true;
    });
  }, [matches, stages, selectedStageId]);

  const rounds = useMemo(() => {
    const rSet = new Set<number>();
    stageScopedMatches.forEach((m) => rSet.add(m.round));
    return Array.from(rSet).sort((a, b) => a - b);
  }, [stageScopedMatches]);

  useEffect(() => {
    if (rounds.length > 0 && selectedRound !== 'all' && !rounds.includes(selectedRound)) {
      setSelectedRound(rounds[0]);
    }
  }, [rounds, selectedRound]);

  const filteredMatches = useMemo(() => {
    return stageScopedMatches.filter((m) => {
      if (selectedRound !== 'all' && m.round !== selectedRound) return false;
      return true;
    });
  }, [stageScopedMatches, selectedRound]);

  const groupedByRound = useMemo(() => {
    const groups = new Map<number, Match[]>();
    filteredMatches.forEach((m) => {
      const list = groups.get(m.round) || [];
      list.push(m);
      groups.set(m.round, list);
    });
    return Array.from(groups.entries()).sort((a, b) => a[0] - b[0]);
  }, [filteredMatches]);

  const handleSlotClick = async (
    match: Match,
    slot: 'home' | 'away',
    teamName: string
  ) => {
    if (!editPairingsMode || !onSwapMatchPairings || match.status === 'completed') return;
    if (!selectedSwapSlot) {
      setSelectedSwapSlot({ matchId: match.id, slot, teamName });
      return;
    }
    if (selectedSwapSlot.matchId === match.id && selectedSwapSlot.slot === slot) {
      setSelectedSwapSlot(null);
      return;
    }
    const first = selectedSwapSlot;
    setSelectedSwapSlot(null);
    await onSwapMatchPairings({
      match_a_id: first.matchId,
      slot_a: first.slot,
      match_b_id: match.id,
      slot_b: slot,
    });
  };

  return (
    <div className="space-y-4">
      {/* Streamlined Pill Toolbar: Round (Hafta) Pills + Group Filter + Edit Matchups */}
      {stageScopedMatches.length > 0 && (
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-xs dark:border-slate-800 dark:bg-slate-900">
          <div className="flex flex-wrap items-center gap-2">
            {/* Round (Hafta) Pill Tabs */}
            {rounds.length > 0 && (
              <div className="inline-flex flex-wrap items-center gap-1 rounded-xl bg-slate-100 p-1 dark:bg-slate-800">
                {rounds.map((r) => {
                  const isActive = selectedRound === r;
                  return (
                    <button
                      key={r}
                      type="button"
                      onClick={() => setSelectedRound(r)}
                      className={`rounded-lg px-3.5 py-1.5 text-xs font-bold transition-all ${
                        isActive
                          ? 'bg-slate-900 text-white shadow-xs dark:bg-blue-600'
                          : 'text-slate-600 hover:bg-white/70 hover:text-slate-900 dark:text-slate-300 dark:hover:bg-slate-700'
                      }`}
                    >
                      {t('fixtures.week_pill', { num: r })}
                    </button>
                  );
                })}
                <button
                  type="button"
                  onClick={() => setSelectedRound('all')}
                  className={`rounded-lg px-3.5 py-1.5 text-xs font-bold transition-all ${
                    selectedRound === 'all'
                      ? 'bg-blue-600 text-white shadow-xs'
                      : 'text-slate-600 hover:bg-white/70 hover:text-slate-900 dark:text-slate-300 dark:hover:bg-slate-700'
                  }`}
                >
                  {t('fixtures.all_rounds')}
                </button>
              </div>
            )}

            {/* Optional Group Selector if not locked externally */}
            {selectedStageFilter === undefined && stages.length > 1 && (
              <select
                value={selectedStageId}
                onChange={(e) =>
                  setSelectedStageId(
                    e.target.value === 'all' ? 'all' : Number(e.target.value)
                  )
                }
                className="rounded-xl border border-slate-300 bg-slate-50 px-3 py-2 text-xs font-bold text-slate-700 focus:border-blue-600 focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200"
              >
                <option value="all">{t('fixtures.all_stages')}</option>
                {stages.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </select>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {onSwapMatchPairings && stageScopedMatches.length > 0 && (
              <button
                type="button"
                onClick={() => {
                  setEditPairingsMode(!editPairingsMode);
                  setSelectedSwapSlot(null);
                }}
                className={`inline-flex items-center justify-center gap-2 rounded-xl px-4 py-2 text-sm font-medium transition-colors ${
                  editPairingsMode
                    ? 'bg-amber-500 text-white shadow-xs'
                    : 'border border-slate-300 bg-slate-50 text-slate-700 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200'
                }`}
              >
                <ArrowLeftRight className="h-4 w-4" />
                {editPairingsMode
                  ? t('fixtures.exit_edit_pairings')
                  : t('fixtures.edit_pairings_mode')}
              </button>
            )}
          </div>
        </div>
      )}

      {editPairingsMode && (
        <div className="flex items-center justify-between gap-2 rounded-xl border border-amber-300 bg-amber-50 px-4 py-2.5 text-xs font-bold text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200">
          <span>
            {selectedSwapSlot
              ? `Seçilen Takım: "${selectedSwapSlot.teamName}" — Yer değiştirmek istediğiniz diğer takıma tıklayın.`
              : t('fixtures.select_slot_to_swap')}
          </span>
          {selectedSwapSlot && (
            <button
              type="button"
              onClick={() => setSelectedSwapSlot(null)}
              className="underline hover:no-underline"
            >
              Seçimi İptal Et
            </button>
          )}
        </div>
      )}

      {/* Matches List Grouped by Round (Hafta) */}
      {groupedByRound.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-slate-300 bg-white p-10 text-center space-y-4 dark:border-slate-700 dark:bg-slate-900">
          <p className="text-sm text-slate-600 dark:text-slate-400">
            {t('fixtures.empty')}
          </p>
          {onOpenSeedingModal && (
            <button
              type="button"
              onClick={onOpenSeedingModal}
              className="inline-flex items-center justify-center gap-2 rounded-xl bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-xs hover:bg-blue-700 transition-colors"
            >
              <Sparkles className="h-4 w-4" />
              {t('fixtures.draw_btn')}
            </button>
          )}
        </div>
      ) : (
        <div className="space-y-4">
          {groupedByRound.map(([roundNum, roundMatches]) => {
            const playedOrScheduledMatches = roundMatches.filter((m) => !m.is_bye);
            const byeMatches = roundMatches.filter((m) => m.is_bye);

            return (
              <div
                key={roundNum}
                className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xs dark:border-slate-800 dark:bg-slate-900"
              >
                <div className="flex items-center justify-between border-b border-slate-200 bg-slate-900 px-4 py-2.5 text-white dark:border-slate-800 dark:bg-slate-950">
                  <div className="flex items-center gap-2">
                    <Calendar className="h-4 w-4 text-blue-400" />
                    <span className="text-xs font-extrabold uppercase tracking-wider">
                      {t('fixtures.week_pill', { num: roundNum })}
                    </span>
                  </div>
                  <span className="rounded-full bg-slate-800 px-2.5 py-0.5 text-xs font-semibold text-slate-300">
                    {playedOrScheduledMatches.length} Maç
                    {byeMatches.length > 0 ? ` • ${byeMatches.length} BAY` : ''}
                  </span>
                </div>

                {/* Active Matches */}
                <div className="divide-y divide-slate-100 dark:divide-slate-800">
                  {playedOrScheduledMatches.map((m) => {
                    const isCompleted = m.status === 'completed';
                    const isForfeit = Boolean(m.is_forfeit);
                    const hasPenalties =
                      m.home_penalties !== undefined &&
                      m.home_penalties !== null &&
                      m.away_penalties !== undefined &&
                      m.away_penalties !== null;
                    const homeWon =
                      isCompleted &&
                      (m.home_score > m.away_score ||
                        (m.home_score === m.away_score &&
                          hasPenalties &&
                          (m.home_penalties ?? 0) > (m.away_penalties ?? 0)));
                    const awayWon =
                      isCompleted &&
                      (m.away_score > m.home_score ||
                        (m.home_score === m.away_score &&
                          hasPenalties &&
                          (m.away_penalties ?? 0) > (m.home_penalties ?? 0)));

                    const isHomeSelected =
                      selectedSwapSlot?.matchId === m.id &&
                      selectedSwapSlot?.slot === 'home';
                    const isAwaySelected =
                      selectedSwapSlot?.matchId === m.id &&
                      selectedSwapSlot?.slot === 'away';

                    return (
                      <div
                        key={m.id}
                        className={`flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 transition-colors ${
                          isForfeit
                            ? 'bg-rose-50/30 hover:bg-rose-50/60 dark:bg-rose-950/20 dark:hover:bg-rose-950/30'
                            : 'hover:bg-blue-50/30 dark:hover:bg-slate-800/60'
                        }`}
                      >
                        {/* Meta info & Stage Badge */}
                        <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400 sm:w-56">
                          <span className="rounded bg-slate-200/80 px-1.5 py-0.5 font-mono font-bold text-slate-700 dark:bg-slate-800 dark:text-slate-300">
                            #{m.match_number}
                          </span>
                          {m.stage_name && (
                            <span className="rounded-md bg-indigo-50 px-2 py-0.5 font-bold text-indigo-700 ring-1 ring-indigo-200/60 dark:bg-indigo-950/60 dark:text-indigo-300 dark:ring-indigo-800">
                              {m.stage_name}
                            </span>
                          )}
                          {m.scheduled_at && (
                            <span className="flex items-center gap-1 text-slate-500 dark:text-slate-400">
                              <Calendar className="h-3 w-3" />
                              {m.scheduled_at}
                            </span>
                          )}
                          {(m.venue || m.court) && (
                            <span className="flex items-center gap-1 text-slate-500 dark:text-slate-400">
                              <MapPin className="h-3 w-3" />
                              {[m.venue, m.court].filter(Boolean).join(' • ')}
                            </span>
                          )}
                        </div>

                        {/* Teams & Score Center */}
                        <div className="flex flex-1 items-center justify-between sm:justify-center gap-3">
                          <div
                            onClick={() =>
                              handleSlotClick(m, 'home', m.home_team_name)
                            }
                            className={`flex-1 text-right text-sm font-bold transition-all ${
                              editPairingsMode && !isCompleted
                                ? isHomeSelected
                                  ? 'cursor-pointer rounded-lg bg-amber-500 px-2 py-1 text-white ring-2 ring-amber-300'
                                  : 'cursor-pointer rounded-lg border border-dashed border-amber-400 px-2 py-1 hover:bg-amber-50 dark:hover:bg-amber-950/40'
                                : ''
                            } ${
                              homeWon
                                ? 'text-emerald-700 dark:text-emerald-400'
                                : 'text-slate-900 dark:text-slate-100'
                            }`}
                          >
                            {m.home_team_name}
                          </div>

                          <button
                            onClick={() => onSelectMatch(m)}
                            className={`min-w-[6.5rem] rounded-xl px-3 py-1.5 text-center font-extrabold transition-transform active:scale-95 ${
                              isForfeit
                                ? 'bg-rose-600 text-white shadow-xs ring-2 ring-rose-200'
                                : isCompleted
                                ? 'bg-slate-900 text-white shadow-xs dark:bg-slate-800 dark:ring-1 dark:ring-slate-700'
                                : 'border border-dashed border-blue-400 bg-blue-50 text-blue-700 hover:bg-blue-100 dark:border-blue-500/60 dark:bg-blue-950/40 dark:text-blue-300'
                            }`}
                          >
                            {m.status === 'scheduled' ? (
                              <span className="text-xs font-bold">v</span>
                            ) : (
                              <div>
                                <div className="text-base tracking-tight">
                                  {m.home_score} - {m.away_score}
                                </div>
                                {hasPenalties && (
                                  <div className="text-[10px] font-bold text-amber-300">
                                    P: {m.home_penalties} - {m.away_penalties}
                                  </div>
                                )}
                              </div>
                            )}
                          </button>

                          <div
                            onClick={() =>
                              handleSlotClick(m, 'away', m.away_team_name)
                            }
                            className={`flex-1 text-left text-sm font-bold transition-all ${
                              editPairingsMode && !isCompleted
                                ? isAwaySelected
                                  ? 'cursor-pointer rounded-lg bg-amber-500 px-2 py-1 text-white ring-2 ring-amber-300'
                                  : 'cursor-pointer rounded-lg border border-dashed border-amber-400 px-2 py-1 hover:bg-amber-50 dark:hover:bg-amber-950/40'
                                : ''
                            } ${
                              awayWon
                                ? 'text-emerald-700 dark:text-emerald-400'
                                : 'text-slate-900 dark:text-slate-100'
                            }`}
                          >
                            {m.away_team_name}
                          </div>
                        </div>

                        {/* Status Badge & Edit Score Action */}
                        <div className="flex items-center justify-end gap-2 sm:w-56">
                          {isForfeit ? (
                            <span className="inline-flex items-center gap-1 rounded-full bg-rose-100 px-2.5 py-1 text-[11px] font-extrabold text-rose-800 ring-1 ring-rose-300 dark:bg-rose-950/60 dark:text-rose-300 dark:ring-rose-800">
                              <Gavel className="h-3 w-3 text-rose-600 dark:text-rose-400" />
                              {t('status.forfeit', {
                                score: `${m.home_score}-${m.away_score}`,
                              })}
                            </span>
                          ) : isCompleted ? (
                            <span className="inline-flex items-center gap-1 rounded-full bg-emerald-100 px-2.5 py-1 text-[11px] font-extrabold text-emerald-800 ring-1 ring-emerald-200 dark:bg-emerald-950/60 dark:text-emerald-300 dark:ring-emerald-800">
                              <CheckCircle className="h-3 w-3 text-emerald-600 dark:text-emerald-400" />
                              {t('status.completed')}
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2.5 py-1 text-[11px] font-bold text-slate-600 ring-1 ring-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:ring-slate-700">
                              <Clock className="h-3 w-3 text-amber-500" />
                              {t('status.scheduled')}
                            </span>
                          )}

                          {!isCompleted && onSwapMatchPairings && (
                            <button
                              type="button"
                              onClick={() =>
                                onSwapMatchPairings({
                                  match_a_id: m.id,
                                  slot_a: 'home',
                                  match_b_id: m.id,
                                  slot_b: 'away',
                                })
                              }
                              title={t('fixtures.swap_home_away')}
                              className="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-semibold text-slate-600 hover:border-amber-500 hover:text-amber-600 shadow-2xs dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300"
                            >
                              <ArrowLeftRight className="h-3.5 w-3.5" />
                            </button>
                          )}

                          <button
                            onClick={() => onSelectMatch(m)}
                            title={t('fixtures.enter_score')}
                            className="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-semibold text-slate-700 hover:border-blue-500 hover:text-blue-600 shadow-2xs dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200"
                          >
                            <Edit3 className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>

                {/* Prominent BYE (BAY) Banner Rows */}
                {byeMatches.length > 0 && (
                  <div className="border-t border-amber-200 bg-amber-50/80 px-4 py-2.5 flex flex-wrap items-center gap-2 dark:border-amber-800/60 dark:bg-amber-950/30">
                    {byeMatches.map((bm) => (
                      <div
                        key={bm.id}
                        className="inline-flex items-center gap-2 rounded-xl border border-amber-300 bg-white px-3 py-1.5 text-xs font-extrabold text-amber-900 shadow-2xs dark:border-amber-700 dark:bg-slate-900 dark:text-amber-300"
                      >
                        <Coffee className="h-3.5 w-3.5 text-amber-600 dark:text-amber-400" />
                        {bm.stage_name && (
                          <span className="rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-bold text-amber-800 dark:bg-amber-500/20 dark:text-amber-300">
                            {bm.stage_name}
                          </span>
                        )}
                        <span>
                          {t('fixtures.bye_banner', { team: bm.home_team_name })}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

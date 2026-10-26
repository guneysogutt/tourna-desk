import React, { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Trophy,
  Award,
  Swords,
  Calendar,
  MapPin,
  Edit3,
  Sparkles,
  Crown,
  GitBranch,
  ArrowLeftRight,
} from 'lucide-react';
import { Match, SportSlug, Stage, Team, Venue, venueSupportsSport } from '../types';

interface FinalFourBracketProps {
  stages: Stage[];
  matches: Match[];
  teams?: Team[];
  sportSlug?: SportSlug;
  venues?: Venue[];
  onSelectMatch: (match: Match) => void;
  onBuildFinalFour: (payload: {
    group_a_stage_id: number;
    group_b_stage_id: number;
    default_venue: string;
    default_court: string;
    start_date: string;
  }) => Promise<void>;
  onBuildKnockout?: (payload: {
    team_ids?: number[];
    seeded_team_ids?: number[];
    default_venue: string;
    default_court: string;
    start_date: string;
  }) => Promise<void>;
  onSwapMatchPairings?: (payload: {
    match_a_id: number;
    slot_a: 'home' | 'away';
    match_b_id: number;
    slot_b: 'home' | 'away';
  }) => Promise<void>;
}

export const FinalFourBracket: React.FC<FinalFourBracketProps> = ({
  stages,
  matches,
  teams = [],
  sportSlug,
  venues = [],
  onSelectMatch,
  onBuildFinalFour,
  onBuildKnockout,
  onSwapMatchPairings,
}) => {
  const { t } = useTranslation();

  const groupStages = useMemo(
    () => stages.filter((s) => s.stage_type === 'group'),
    [stages]
  );

  const matchingVenues = useMemo(
    () => venues.filter((v) => venueSupportsSport(v, sportSlug)),
    [venues, sportSlug]
  );

  const [groupAId, setGroupAId] = useState<number>(0);
  const [groupBId, setGroupBId] = useState<number>(0);
  const [selectedVenue, setSelectedVenue] = useState('');
  const [startDate, setStartDate] = useState('2026-10-28');
  const [building, setBuilding] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    if (matchingVenues.length > 0 && !selectedVenue) {
      setSelectedVenue(matchingVenues[0].name);
    }
  }, [matchingVenues]);

  // Interactive swap state
  const [editPairingsMode, setEditPairingsMode] = useState(false);
  const [selectedSlot, setSelectedSlot] = useState<{
    matchId: number;
    slot: 'home' | 'away';
    teamName: string;
  } | null>(null);
  const [swapping, setSwapping] = useState(false);

  const canBuildDirectKnockout = teams.length >= 4 && teams.length % 4 === 0;

  useEffect(() => {
    if (groupStages.length >= 2) {
      const maxR = Math.max(...groupStages.map((s) => s.round_number || 1));
      const latest = groupStages.filter((s) => (s.round_number || 1) === maxR);
      if (latest.length >= 2) {
        setGroupAId(latest[0].id);
        setGroupBId(latest[1].id);
      } else {
        setGroupAId(groupStages[groupStages.length - 2].id);
        setGroupBId(groupStages[groupStages.length - 1].id);
      }
    } else if (groupStages.length === 1) {
      setGroupAId(groupStages[0].id);
    }
  }, [groupStages]);

  // Final Four strictly uses the final_four stage
  const bracketStage = useMemo(() => {
    const candidates = stages.filter((s) => s.stage_type === 'final_four');
    if (candidates.length === 0) return undefined;
    return candidates[candidates.length - 1];
  }, [stages]);

  const ffMatches = useMemo(() => {
    if (!bracketStage) return [];
    return matches
      .filter((m) => m.stage_id === bracketStage.id)
      .sort((a, b) => a.round - b.round || a.match_number - b.match_number);
  }, [matches, bracketStage]);

  const r32Matches = useMemo(
    () => ffMatches.filter((m) => m.bracket_slot?.startsWith('r32_')),
    [ffMatches]
  );
  const r16Matches = useMemo(
    () => ffMatches.filter((m) => m.bracket_slot?.startsWith('r16_')),
    [ffMatches]
  );
  const qfMatches = useMemo(
    () => ffMatches.filter((m) => m.bracket_slot?.startsWith('qf_')),
    [ffMatches]
  );

  const sf1 = useMemo(
    () =>
      ffMatches.find((m) => m.bracket_slot === 'semi_final_1') ||
      (r32Matches.length === 0 && r16Matches.length === 0 && qfMatches.length === 0
        ? ffMatches.find((m) => m.match_number === 1)
        : undefined),
    [ffMatches, r32Matches.length, r16Matches.length, qfMatches.length]
  );
  const sf2 = useMemo(
    () =>
      ffMatches.find((m) => m.bracket_slot === 'semi_final_2') ||
      (r32Matches.length === 0 && r16Matches.length === 0 && qfMatches.length === 0
        ? ffMatches.find((m) => m.match_number === 2)
        : undefined),
    [ffMatches, r32Matches.length, r16Matches.length, qfMatches.length]
  );
  const thirdPlace = useMemo(
    () =>
      ffMatches.find((m) => m.bracket_slot === 'third_place') ||
      (r32Matches.length === 0 && r16Matches.length === 0 && qfMatches.length === 0
        ? ffMatches.find((m) => m.match_number === 3)
        : undefined),
    [ffMatches, r32Matches.length, r16Matches.length, qfMatches.length]
  );
  const finalMatch = useMemo(
    () =>
      ffMatches.find((m) => m.bracket_slot === 'final') ||
      (r32Matches.length === 0 && r16Matches.length === 0 && qfMatches.length === 0
        ? ffMatches.find((m) => m.match_number === 4)
        : undefined),
    [ffMatches, r32Matches.length, r16Matches.length, qfMatches.length]
  );

  const resolveMatchWinnerName = (m: Match | undefined): string | null => {
    if (!m || m.status !== 'completed') return null;
    if (m.home_score > m.away_score) return m.home_team_name;
    if (m.away_score > m.home_score) return m.away_team_name;
    if (
      m.home_penalties !== null &&
      m.home_penalties !== undefined &&
      m.away_penalties !== null &&
      m.away_penalties !== undefined
    ) {
      if (m.home_penalties > m.away_penalties) return m.home_team_name;
      if (m.away_penalties > m.home_penalties) return m.away_team_name;
    }
    return null;
  };

  const championName = useMemo(() => resolveMatchWinnerName(finalMatch), [finalMatch]);
  const thirdPlaceWinnerName = useMemo(
    () => resolveMatchWinnerName(thirdPlace),
    [thirdPlace]
  );

  const handleBuild = async (e: React.FormEvent) => {
    e.preventDefault();
    setBuilding(true);
    setErrorMsg(null);
    try {
      await onBuildFinalFour({
        group_a_stage_id: groupAId,
        group_b_stage_id: groupBId,
        default_venue: selectedVenue,
        default_court: '',
        start_date: startDate,
      });
    } catch (err: any) {
      setErrorMsg(err?.message || 'Final Four oluşturulamadı');
    } finally {
      setBuilding(false);
    }
  };

  const handleBuildDirectKnockout = async () => {
    if (!onBuildKnockout) return;
    setBuilding(true);
    setErrorMsg(null);
    try {
      await onBuildKnockout({
        seeded_team_ids: teams.filter((tm) => tm.is_seeded).map((tm) => tm.id),
        default_venue: selectedVenue,
        default_court: '',
        start_date: startDate,
      });
    } catch (err: any) {
      setErrorMsg(err?.message || 'Eleme ağacı oluşturulamadı');
    } finally {
      setBuilding(false);
    }
  };

  const handleSlotClick = async (
    e: React.MouseEvent,
    match: Match,
    slot: 'home' | 'away'
  ) => {
    if (!editPairingsMode || !onSwapMatchPairings) return;
    e.stopPropagation();
    if (match.status === 'completed') return;

    const teamName = slot === 'home' ? match.home_team_name : match.away_team_name;

    if (!selectedSlot) {
      setSelectedSlot({ matchId: match.id, slot, teamName });
      return;
    }

    if (selectedSlot.matchId === match.id && selectedSlot.slot === slot) {
      setSelectedSlot(null);
      return;
    }

    setSwapping(true);
    try {
      await onSwapMatchPairings({
        match_a_id: selectedSlot.matchId,
        slot_a: selectedSlot.slot,
        match_b_id: match.id,
        slot_b: slot,
      });
    } finally {
      setSwapping(false);
      setSelectedSlot(null);
    }
  };

  const renderMatchCard = (
    match: Match | undefined,
    title: string,
    accent: 'blue' | 'gold' | 'bronze'
  ) => {
    if (!match) return null;
    const isCompleted = match.status === 'completed';
    const hasPenalties =
      match.home_penalties !== null &&
      match.home_penalties !== undefined &&
      match.away_penalties !== null &&
      match.away_penalties !== undefined;

    const homeWon =
      isCompleted &&
      (match.home_score > match.away_score ||
        (match.home_score === match.away_score &&
          hasPenalties &&
          (match.home_penalties || 0) > (match.away_penalties || 0)));
    const awayWon =
      isCompleted &&
      (match.away_score > match.home_score ||
        (match.home_score === match.away_score &&
          hasPenalties &&
          (match.away_penalties || 0) > (match.home_penalties || 0)));

    const isHomeSelected =
      editPairingsMode &&
      selectedSlot?.matchId === match.id &&
      selectedSlot?.slot === 'home';
    const isAwaySelected =
      editPairingsMode &&
      selectedSlot?.matchId === match.id &&
      selectedSlot?.slot === 'away';

    const headerClasses =
      accent === 'gold'
        ? 'bg-gradient-to-r from-amber-500 to-yellow-600 text-white'
        : accent === 'bronze'
        ? 'bg-gradient-to-r from-orange-600 to-amber-700 text-white'
        : 'bg-slate-900 dark:bg-slate-800 text-white';

    const borderClasses =
      accent === 'gold'
        ? 'border-2 border-amber-300 dark:border-amber-500/60 shadow-md'
        : accent === 'bronze'
        ? 'border-2 border-orange-200 dark:border-orange-500/50 shadow-sm'
        : 'border border-slate-200 dark:border-slate-700 shadow-sm';

    return (
      <div
        key={match.id}
        onClick={() => {
          if (!editPairingsMode) onSelectMatch(match);
        }}
        className={`group cursor-pointer overflow-hidden rounded-2xl bg-white dark:bg-slate-900 transition-all hover:ring-2 hover:ring-blue-500 ${borderClasses}`}
      >
        <div className={`flex items-center justify-between px-4 py-2.5 ${headerClasses}`}>
          <div className="flex items-center gap-2">
            {accent === 'gold' ? (
              <Trophy className="h-4 w-4 text-amber-100" />
            ) : accent === 'bronze' ? (
              <Award className="h-4 w-4 text-orange-100" />
            ) : (
              <Swords className="h-4 w-4 text-blue-400" />
            )}
            <span className="text-xs font-extrabold uppercase tracking-wider">
              {title}
            </span>
          </div>
          <span className="inline-flex items-center gap-1 rounded bg-black/20 px-2 py-0.5 text-[11px] font-bold">
            <Edit3 className="h-3 w-3" />
            #{match.match_number}
          </span>
        </div>

        <div className="divide-y divide-slate-100 dark:divide-slate-800 p-3 space-y-2">
          {/* Home Team Row */}
          <div
            onClick={(e) => handleSlotClick(e, match, 'home')}
            className={`flex items-center justify-between rounded-xl px-3 py-2 transition-all ${
              isHomeSelected
                ? 'ring-2 ring-amber-500 bg-amber-50 dark:bg-amber-950/60 text-amber-900 dark:text-amber-200 font-extrabold'
                : editPairingsMode && !isCompleted
                ? 'hover:bg-amber-50/70 dark:hover:bg-amber-950/30 border border-dashed border-amber-300 dark:border-amber-700 text-slate-800 dark:text-slate-100 font-semibold'
                : homeWon
                ? 'bg-emerald-50 dark:bg-emerald-950/50 text-emerald-900 dark:text-emerald-200 font-extrabold'
                : 'text-slate-800 dark:text-slate-200 font-semibold'
            }`}
          >
            <div className="flex items-center gap-2 min-w-0">
              <span className="truncate text-sm">{match.home_team_name}</span>
            </div>
            <div className="ml-2 flex items-center gap-1">
              <span
                className={`inline-flex h-7 min-w-[2rem] items-center justify-center rounded-lg px-2 text-sm font-black ${
                  isCompleted
                    ? homeWon
                      ? 'bg-emerald-600 text-white'
                      : 'bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-300'
                    : 'bg-slate-100 dark:bg-slate-800 text-slate-400 dark:text-slate-500'
                }`}
              >
                {isCompleted ? match.home_score : '-'}
              </span>
              {hasPenalties && (
                <span className="rounded bg-amber-100 dark:bg-amber-900/50 px-1.5 py-0.5 text-[10px] font-extrabold text-amber-800 dark:text-amber-300">
                  P:{match.home_penalties}
                </span>
              )}
            </div>
          </div>

          {/* Away Team Row */}
          <div
            onClick={(e) => handleSlotClick(e, match, 'away')}
            className={`flex items-center justify-between rounded-xl px-3 py-2 pt-2.5 transition-all ${
              isAwaySelected
                ? 'ring-2 ring-amber-500 bg-amber-50 dark:bg-amber-950/60 text-amber-900 dark:text-amber-200 font-extrabold'
                : editPairingsMode && !isCompleted
                ? 'hover:bg-amber-50/70 dark:hover:bg-amber-950/30 border border-dashed border-amber-300 dark:border-amber-700 text-slate-800 dark:text-slate-100 font-semibold'
                : awayWon
                ? 'bg-emerald-50 dark:bg-emerald-950/50 text-emerald-900 dark:text-emerald-200 font-extrabold'
                : 'text-slate-800 dark:text-slate-200 font-semibold'
            }`}
          >
            <div className="flex items-center gap-2 min-w-0">
              <span className="truncate text-sm">{match.away_team_name}</span>
            </div>
            <div className="ml-2 flex items-center gap-1">
              <span
                className={`inline-flex h-7 min-w-[2rem] items-center justify-center rounded-lg px-2 text-sm font-black ${
                  isCompleted
                    ? awayWon
                      ? 'bg-emerald-600 text-white'
                      : 'bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-300'
                    : 'bg-slate-100 dark:bg-slate-800 text-slate-400 dark:text-slate-500'
                }`}
              >
                {isCompleted ? match.away_score : '-'}
              </span>
              {hasPenalties && (
                <span className="rounded bg-amber-100 dark:bg-amber-900/50 px-1.5 py-0.5 text-[10px] font-extrabold text-amber-800 dark:text-amber-300">
                  P:{match.away_penalties}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Footer Metadata */}
        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-slate-100 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/60 px-3.5 py-2 text-[11px] text-slate-500 dark:text-slate-400">
          <div className="flex items-center gap-2">
            {match.scheduled_at && (
              <span className="flex items-center gap-1">
                <Calendar className="h-3 w-3" />
                {match.scheduled_at}
              </span>
            )}
            {(match.venue || match.court) && (
              <span className="flex items-center gap-1">
                <MapPin className="h-3 w-3" />
                {[match.venue, match.court].filter(Boolean).join(' • ')}
              </span>
            )}
          </div>
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onSelectMatch(match);
            }}
            className="font-bold text-blue-600 dark:text-blue-400 hover:underline"
          >
            {t('fixtures.enter_score')}
          </button>
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      {/* Configuration & Generation Bar */}
      <form
        onSubmit={handleBuild}
        className="rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-5 shadow-xs space-y-4"
      >
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <Trophy className="h-5 w-5 text-amber-500" />
              <h3 className="text-base font-extrabold text-slate-900 dark:text-slate-100">
                {t('final_four.title')}
              </h3>
            </div>
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
              {t('final_four.subtitle')}
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {onSwapMatchPairings && ffMatches.length > 0 && (
              <button
                type="button"
                onClick={() => {
                  setEditPairingsMode((prev) => !prev);
                  setSelectedSlot(null);
                }}
                className={`inline-flex items-center gap-1.5 rounded-xl px-3.5 py-2.5 text-xs font-extrabold transition-all ${
                  editPairingsMode
                    ? 'bg-amber-500 text-white shadow-sm'
                    : 'border border-amber-300 dark:border-amber-700 bg-amber-50 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300 hover:bg-amber-100'
                }`}
              >
                <ArrowLeftRight className="h-3.5 w-3.5" />
                {editPairingsMode
                  ? t('fixtures.edit_pairings_active')
                  : t('fixtures.edit_pairings_mode')}
              </button>
            )}

            {onBuildKnockout && canBuildDirectKnockout && (
              <button
                type="button"
                disabled={building}
                onClick={handleBuildDirectKnockout}
                className="inline-flex items-center gap-2 rounded-xl bg-purple-600 px-4 py-2.5 text-xs font-extrabold text-white shadow-sm hover:bg-purple-700 disabled:opacity-50"
              >
                <GitBranch className="h-4 w-4" />
                {t('final_four.build_knockout_btn')} ({teams.length})
              </button>
            )}

            <button
              type="submit"
              disabled={building || groupStages.length < 2 || groupAId === groupBId}
              className="inline-flex items-center gap-2 rounded-xl bg-amber-600 px-4 py-2.5 text-xs font-extrabold text-white shadow-sm hover:bg-amber-700 disabled:opacity-50"
            >
              <Sparkles className="h-4 w-4" />
              {t('final_four.build_btn')}
            </button>
          </div>
        </div>

        {editPairingsMode && (
          <div className="flex items-center justify-between rounded-xl border border-amber-300 dark:border-amber-700 bg-amber-50/90 dark:bg-amber-950/40 px-4 py-2.5 text-xs font-semibold text-amber-900 dark:text-amber-200">
            <div className="flex items-center gap-2">
              <ArrowLeftRight className="h-4 w-4 text-amber-600 dark:text-amber-400 shrink-0" />
              <span>
                {selectedSlot
                  ? `Seçili Takım: "${selectedSlot.teamName}" — Yer değiştirmek istediğiniz diğer takım kutusuna tıklayın.`
                  : t('fixtures.swap_hint')}
              </span>
            </div>
            {swapping && (
              <span className="text-[11px] font-bold uppercase text-amber-700 dark:text-amber-300">
                ...
              </span>
            )}
          </div>
        )}

        {errorMsg && (
          <div className="rounded-xl border border-rose-200 dark:border-rose-800 bg-rose-50 dark:bg-rose-950/50 px-3.5 py-2 text-xs font-semibold text-rose-700 dark:text-rose-300">
            {errorMsg}
          </div>
        )}

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 pt-2 border-t border-slate-100 dark:border-slate-800">
          <div>
            <label className="block text-[11px] font-bold uppercase text-slate-500 dark:text-slate-400 mb-1">
              {t('final_four.select_group_a')}
            </label>
            <select
              value={groupAId}
              onChange={(e) => setGroupAId(Number(e.target.value))}
              className="w-full rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-2.5 py-1.5 text-xs font-semibold"
            >
              {groupStages.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} (Tur {s.round_number || 1})
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-[11px] font-bold uppercase text-slate-500 dark:text-slate-400 mb-1">
              {t('final_four.select_group_b')}
            </label>
            <select
              value={groupBId}
              onChange={(e) => setGroupBId(Number(e.target.value))}
              className="w-full rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-2.5 py-1.5 text-xs font-semibold"
            >
              {groupStages.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} (Tur {s.round_number || 1})
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-[11px] font-bold uppercase text-slate-500 dark:text-slate-400 mb-1">
              {t('venues.select_venue')}
            </label>
            <select
              value={selectedVenue}
              onChange={(e) => setSelectedVenue(e.target.value)}
              className="w-full rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-2.5 py-1.5 text-xs font-semibold"
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
              className="w-full rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-2.5 py-1.5 text-xs"
            />
          </div>
        </div>
      </form>

      {/* Champion & Podium Banner if completed */}
      {(championName || thirdPlaceWinnerName) && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {championName && (
            <div className="flex items-center gap-3.5 rounded-2xl border-2 border-amber-300 dark:border-amber-500/60 bg-gradient-to-r from-amber-50 to-yellow-50 dark:from-amber-950/50 dark:to-yellow-950/30 p-4 shadow-sm">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-amber-500 text-white shadow-md">
                <Crown className="h-6 w-6" />
              </div>
              <div>
                <div className="text-xs font-extrabold uppercase tracking-wider text-amber-700 dark:text-amber-300">
                  {t('final_four.champion')}
                </div>
                <div className="text-lg font-black text-slate-900 dark:text-slate-100">
                  {championName}
                </div>
              </div>
            </div>
          )}

          {thirdPlaceWinnerName && (
            <div className="flex items-center gap-3.5 rounded-2xl border-2 border-orange-200 dark:border-orange-500/50 bg-orange-50/70 dark:bg-orange-950/30 p-4 shadow-sm">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-orange-600 text-white shadow-md">
                <Award className="h-6 w-6" />
              </div>
              <div>
                <div className="text-xs font-extrabold uppercase tracking-wider text-orange-700 dark:text-orange-300">
                  {t('final_four.third_place_winner')}
                </div>
                <div className="text-lg font-black text-slate-900 dark:text-slate-100">
                  {thirdPlaceWinnerName}
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Visual Bracket Tree */}
      {!bracketStage || ffMatches.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 p-12 text-center text-sm text-slate-500 dark:text-slate-400">
          {t('final_four.empty')}
        </div>
      ) : (
        <div className="rounded-2xl border border-slate-200 dark:border-slate-800 bg-slate-100/70 dark:bg-slate-900/60 p-5 sm:p-6 space-y-8">
          {/* Earlier Knockout Rounds (Round of 32, Round of 16, Quarter-Finals) */}
          {r32Matches.length > 0 && (
            <div className="space-y-3">
              <div className="text-xs font-extrabold uppercase tracking-wider text-purple-600 dark:text-purple-400 px-1">
                {t('final_four.round_of_32')} ({r32Matches.length})
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
                {r32Matches.map((m, idx) =>
                  renderMatchCard(m, `${t('final_four.round_of_32')} #${idx + 1}`, 'blue')
                )}
              </div>
            </div>
          )}

          {r16Matches.length > 0 && (
            <div className="space-y-3">
              <div className="text-xs font-extrabold uppercase tracking-wider text-indigo-600 dark:text-indigo-400 px-1">
                {t('final_four.round_of_16')} ({r16Matches.length})
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
                {r16Matches.map((m, idx) =>
                  renderMatchCard(m, `${t('final_four.round_of_16')} #${idx + 1}`, 'blue')
                )}
              </div>
            </div>
          )}

          {qfMatches.length > 0 && (
            <div className="space-y-3">
              <div className="text-xs font-extrabold uppercase tracking-wider text-blue-600 dark:text-blue-400 px-1">
                {t('final_four.quarter_finals')} ({qfMatches.length})
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
                {qfMatches.map((m, idx) =>
                  renderMatchCard(m, `${t('final_four.quarter_finals')} #${idx + 1}`, 'blue')
                )}
              </div>
            </div>
          )}

          {/* Final Four Stage: Semi-Finals -> Final & 3rd Place */}
          <div className="grid grid-cols-1 lg:grid-cols-11 items-center gap-6">
            {/* Left Column: Semi-Finals */}
            <div className="lg:col-span-5 space-y-6">
              <div className="text-xs font-extrabold uppercase tracking-wider text-slate-500 dark:text-slate-400 px-1">
                {t('final_four.semi_finals')}
              </div>
              {renderMatchCard(sf1, t('final_four.semi_final_1'), 'blue')}
              {renderMatchCard(sf2, t('final_four.semi_final_2'), 'blue')}
            </div>

            {/* Center Connector Tree Lines */}
            <div className="hidden lg:flex lg:col-span-1 flex-col items-center justify-center h-full py-12">
              <div className="w-full h-44 border-y-2 border-r-2 border-slate-300 dark:border-slate-700 rounded-r-xl relative">
                <div className="absolute -right-6 top-1/2 w-6 border-t-2 border-amber-400" />
              </div>
            </div>

            {/* Right Column: Final & 3rd Place Match */}
            <div className="lg:col-span-5 space-y-6">
              <div className="text-xs font-extrabold uppercase tracking-wider text-amber-700 dark:text-amber-400 px-1">
                Şampiyonluk & Üçüncülük Maçları
              </div>
              {renderMatchCard(finalMatch, t('final_four.final_match'), 'gold')}
              {renderMatchCard(thirdPlace, t('final_four.third_place_match'), 'bronze')}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

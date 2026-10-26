import React, { useState, useMemo, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Trophy,
  Award,
  Swords,
  Calendar,
  MapPin,
  Edit3,
  Crown,
  GitBranch,
  ArrowLeftRight,
  CheckCircle2,
  Clock,
  FastForward,
  Inbox,
  Users,
} from 'lucide-react';
import { Match, SportSlug, Stage, Team, Venue, venueSupportsSport } from '../types';

export function calculateBracketMath(teamCount: number): {
  bracketSize: number;
  byeCount: number;
  totalMatches: number;
} {
  if (teamCount <= 0) {
    return { bracketSize: 0, byeCount: 0, totalMatches: 0 };
  }
  let bracketSize = 4;
  while (bracketSize < teamCount) {
    bracketSize *= 2;
  }
  const byeCount = bracketSize - teamCount;
  const totalMatches = Math.max(0, teamCount - 1);
  return { bracketSize, byeCount, totalMatches };
}

interface BracketTreeProps {
  stages: Stage[];
  matches: Match[];
  teams: Team[];
  sportSlug?: SportSlug;
  venues?: Venue[];
  onSelectMatch: (match: Match) => void;
  onBuildKnockout: (payload: {
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
  onManageTeams?: () => void;
}

export const BracketTree: React.FC<BracketTreeProps> = ({
  stages,
  matches,
  teams,
  sportSlug,
  venues = [],
  onSelectMatch,
  onBuildKnockout,
  onSwapMatchPairings,
  onManageTeams,
}) => {
  const { t } = useTranslation();

  const matchingVenues = useMemo(
    () => venues.filter((v) => venueSupportsSport(v, sportSlug)),
    [venues, sportSlug]
  );

  const [selectedVenue, setSelectedVenue] = useState('');
  const [startDate, setStartDate] = useState('2026-10-28');
  const [building, setBuilding] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    if (matchingVenues.length > 0 && !selectedVenue) {
      setSelectedVenue(matchingVenues[0].name);
    }
  }, [matchingVenues]);

  // Interactive team swap state
  const [editPairingsMode, setEditPairingsMode] = useState(false);
  const [selectedSlot, setSelectedSlot] = useState<{
    matchId: number;
    slot: 'home' | 'away';
    teamName: string;
  } | null>(null);
  const [swapping, setSwapping] = useState(false);

  const knockoutStage = useMemo(() => {
    const koStages = stages.filter((s) => s.stage_type === 'knockout');
    if (koStages.length > 0) return koStages[koStages.length - 1];
    return undefined;
  }, [stages]);

  const bracketMatches = useMemo(() => {
    if (!knockoutStage) return [];
    return matches
      .filter((m) => m.stage_id === knockoutStage.id)
      .sort((a, b) => a.round - b.round || a.match_number - b.match_number);
  }, [matches, knockoutStage]);

  const { roundsList, thirdPlaceMatch, finalMatch } = useMemo(() => {
    const byRound = new Map<number, Match[]>();
    let thirdPlace: Match | undefined;
    let finalM: Match | undefined;

    for (const m of bracketMatches) {
      if (m.bracket_slot === 'third_place') {
        thirdPlace = m;
        continue;
      }
      if (m.bracket_slot === 'final') {
        finalM = m;
      }
      const list = byRound.get(m.round) || [];
      list.push(m);
      byRound.set(m.round, list);
    }

    const sortedRounds = Array.from(byRound.entries())
      .sort((a, b) => a[0] - b[0])
      .map(([roundNumber, roundMatches]) => ({
        roundNumber,
        matches: roundMatches.sort((a, b) => a.match_number - b.match_number),
      }));

    return {
      roundsList: sortedRounds,
      thirdPlaceMatch: thirdPlace,
      finalMatch: finalM,
    };
  }, [bracketMatches]);

  const mathSummary = useMemo(
    () => calculateBracketMath(teams.length),
    [teams.length]
  );

  const resolveMatchWinnerName = (m: Match | undefined): string | null => {
    if (!m) return null;
    if (m.is_bye || m.status === 'bye_advanced') {
      return m.home_team_name;
    }
    if (m.status !== 'completed') return null;
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
    () => resolveMatchWinnerName(thirdPlaceMatch),
    [thirdPlaceMatch]
  );

  const getRoundHeaderLabel = (matchesInRound: number, roundNumber: number): string => {
    switch (matchesInRound) {
      case 32:
        return t('final_four.round_of_64');
      case 16:
        return t('final_four.round_of_32');
      case 8:
        return t('final_four.round_of_16');
      case 4:
        return t('final_four.quarter_finals');
      case 2:
        return t('final_four.semi_finals');
      case 1:
        return t('final_four.final_match');
      default:
        return `${roundNumber}. Tur`;
    }
  };

  const handleBuildBracket = async (e: React.FormEvent) => {
    e.preventDefault();
    if (teams.length < 2) {
      setErrorMsg(t('seeding.knockout_size_warning', { count: teams.length }));
      return;
    }
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

  const renderMatchNode = (
    match: Match,
    accent: 'default' | 'gold' | 'bronze' = 'default'
  ) => {
    const isBye = match.is_bye || match.status === 'bye_advanced';
    const isCompleted = match.status === 'completed';
    const hasPenalties =
      match.home_penalties !== null &&
      match.home_penalties !== undefined &&
      match.away_penalties !== null &&
      match.away_penalties !== undefined;

    const homeWon =
      isBye ||
      (isCompleted &&
        (match.home_score > match.away_score ||
          (match.home_score === match.away_score &&
            hasPenalties &&
            (match.home_penalties || 0) > (match.away_penalties || 0))));

    const awayWon =
      !isBye &&
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

    const headerBg =
      accent === 'gold'
        ? 'bg-gradient-to-r from-amber-500 to-yellow-600 text-white'
        : accent === 'bronze'
        ? 'bg-gradient-to-r from-orange-600 to-amber-700 text-white'
        : isBye
        ? 'bg-amber-900/90 dark:bg-amber-950 text-amber-100'
        : 'bg-slate-900 dark:bg-slate-800 text-white';

    const cardBorder =
      accent === 'gold'
        ? 'border-2 border-amber-400 dark:border-amber-500/70 shadow-md'
        : accent === 'bronze'
        ? 'border-2 border-orange-300 dark:border-orange-500/60 shadow-sm'
        : isBye
        ? 'border border-amber-300/80 dark:border-amber-800/70 bg-amber-50/20 dark:bg-slate-900/80'
        : 'border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 shadow-xs';

    return (
      <div
        key={match.id}
        onClick={() => {
          if (!editPairingsMode && !isBye) {
            onSelectMatch(match);
          }
        }}
        className={`group relative w-72 shrink-0 overflow-hidden rounded-2xl transition-all ${
          !isBye ? 'cursor-pointer hover:ring-2 hover:ring-blue-500' : ''
        } ${cardBorder}`}
      >
        {/* Top Header */}
        <div className={`flex items-center justify-between px-3.5 py-2 ${headerBg}`}>
          <div className="flex items-center gap-1.5 min-w-0">
            {accent === 'gold' ? (
              <Trophy className="h-3.5 w-3.5 text-amber-100 shrink-0" />
            ) : accent === 'bronze' ? (
              <Award className="h-3.5 w-3.5 text-orange-100 shrink-0" />
            ) : (
              <Swords className="h-3.5 w-3.5 text-blue-400 shrink-0" />
            )}
            <span className="truncate text-[11px] font-extrabold uppercase tracking-wider">
              #{match.match_number} • {match.notes || `Tur ${match.round}`}
            </span>
          </div>

          {isBye ? (
            <span className="inline-flex items-center gap-1 rounded-md bg-amber-400/20 px-2 py-0.5 text-[10px] font-black uppercase text-amber-200 ring-1 ring-amber-300/40">
              <FastForward className="h-2.5 w-2.5" />
              {t('status.bye')}
            </span>
          ) : isCompleted ? (
            <span className="inline-flex items-center gap-1 rounded-md bg-emerald-500/20 px-2 py-0.5 text-[10px] font-extrabold text-emerald-200 ring-1 ring-emerald-400/30">
              <CheckCircle2 className="h-2.5 w-2.5" />
              {t('status.completed')}
            </span>
          ) : (
            <span className="inline-flex items-center gap-1 rounded-md bg-white/15 px-2 py-0.5 text-[10px] font-bold text-slate-200">
              <Clock className="h-2.5 w-2.5" />
              {t('status.scheduled')}
            </span>
          )}
        </div>

        {/* Team Rows */}
        <div className="divide-y divide-slate-100 dark:divide-slate-800 p-2.5 space-y-1.5 bg-white dark:bg-slate-900">
          {/* Home Slot */}
          <div
            onClick={(e) => handleSlotClick(e, match, 'home')}
            className={`flex items-center justify-between rounded-xl px-2.5 py-1.5 transition-all ${
              isHomeSelected
                ? 'ring-2 ring-amber-500 bg-amber-50 dark:bg-amber-950/60 text-amber-900 dark:text-amber-200 font-extrabold'
                : editPairingsMode && !isCompleted
                ? 'cursor-pointer hover:bg-amber-50/70 dark:hover:bg-amber-950/30 border border-dashed border-amber-300 dark:border-amber-700 text-slate-800 dark:text-slate-100 font-semibold'
                : homeWon
                ? 'bg-emerald-50 dark:bg-emerald-950/50 text-emerald-900 dark:text-emerald-200 font-extrabold'
                : 'text-slate-800 dark:text-slate-200 font-semibold'
            }`}
          >
            <span className="truncate text-xs" title={match.home_team_name}>
              {match.home_team_name}
            </span>
            <div className="ml-2 flex items-center gap-1 shrink-0">
              {isBye ? (
                <span className="rounded bg-emerald-100 dark:bg-emerald-900/50 px-1.5 py-0.5 text-[10px] font-extrabold text-emerald-800 dark:text-emerald-300">
                  TUR ATLADI
                </span>
              ) : (
                <>
                  <span
                    className={`inline-flex h-6 min-w-[1.75rem] items-center justify-center rounded-lg px-1.5 text-xs font-black ${
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
                    <span className="rounded bg-amber-100 dark:bg-amber-900/50 px-1 py-0.5 text-[10px] font-extrabold text-amber-800 dark:text-amber-300">
                      P:{match.home_penalties}
                    </span>
                  )}
                </>
              )}
            </div>
          </div>

          {/* Away Slot */}
          <div
            onClick={(e) => handleSlotClick(e, match, 'away')}
            className={`flex items-center justify-between rounded-xl px-2.5 py-1.5 pt-2 transition-all ${
              isAwaySelected
                ? 'ring-2 ring-amber-500 bg-amber-50 dark:bg-amber-950/60 text-amber-900 dark:text-amber-200 font-extrabold'
                : editPairingsMode && !isCompleted
                ? 'cursor-pointer hover:bg-amber-50/70 dark:hover:bg-amber-950/30 border border-dashed border-amber-300 dark:border-amber-700 text-slate-800 dark:text-slate-100 font-semibold'
                : isBye
                ? 'bg-amber-50/60 dark:bg-amber-950/20 text-amber-700 dark:text-amber-400 italic font-bold'
                : awayWon
                ? 'bg-emerald-50 dark:bg-emerald-950/50 text-emerald-900 dark:text-emerald-200 font-extrabold'
                : 'text-slate-800 dark:text-slate-200 font-semibold'
            }`}
          >
            <span className="truncate text-xs" title={match.away_team_name}>
              {isBye ? `${match.home_team_name} - ${t('status.bye')}` : match.away_team_name}
            </span>
            <div className="ml-2 flex items-center gap-1 shrink-0">
              {isBye ? (
                <span className="rounded bg-amber-100 dark:bg-amber-900/50 px-1.5 py-0.5 text-[10px] font-black text-amber-800 dark:text-amber-300">
                  {t('status.bye')}
                </span>
              ) : (
                <>
                  <span
                    className={`inline-flex h-6 min-w-[1.75rem] items-center justify-center rounded-lg px-1.5 text-xs font-black ${
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
                    <span className="rounded bg-amber-100 dark:bg-amber-900/50 px-1 py-0.5 text-[10px] font-extrabold text-amber-800 dark:text-amber-300">
                      P:{match.away_penalties}
                    </span>
                  )}
                </>
              )}
            </div>
          </div>
        </div>

        {/* Footer Metadata */}
        <div className="flex items-center justify-between gap-2 border-t border-slate-100 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/60 px-3 py-1.5 text-[10px] text-slate-500 dark:text-slate-400">
          <div className="flex items-center gap-2 truncate">
            {match.scheduled_at && !isBye && (
              <span className="flex items-center gap-1 truncate">
                <Calendar className="h-3 w-3 shrink-0" />
                {match.scheduled_at}
              </span>
            )}
            {match.venue && !isBye && (
              <span className="flex items-center gap-1 truncate">
                <MapPin className="h-3 w-3 shrink-0" />
                {match.venue}
              </span>
            )}
            {isBye && <span>{t('bracket.bye_advanced')}</span>}
          </div>
          {!isBye && (
            <span className="inline-flex items-center gap-0.5 font-bold text-blue-600 dark:text-blue-400 group-hover:underline shrink-0">
              <Edit3 className="h-2.5 w-2.5" />
              Skor
            </span>
          )}
        </div>
      </div>
    );
  };

  const canBuild = teams.length >= 2;

  return (
    <div className="space-y-6">
      {/* Control & Bracket Math Bar */}
      <form
        onSubmit={handleBuildBracket}
        className="rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-5 shadow-xs space-y-4"
      >
        <div className="flex flex-col xl:flex-row xl:items-center justify-between gap-4">
          <div>
            <div className="flex flex-wrap items-center gap-2.5">
              <GitBranch className="h-5 w-5 text-purple-600 dark:text-purple-400" />
              <h3 className="text-base font-extrabold text-slate-900 dark:text-slate-100">
                {t('bracket.title')}
              </h3>
              {teams.length >= 2 && (
                <span className="inline-flex items-center gap-1.5 rounded-full bg-purple-100 dark:bg-purple-950/70 border border-purple-200 dark:border-purple-800 px-3 py-0.5 text-xs font-extrabold text-purple-800 dark:text-purple-300">
                  {t('bracket.stats_badge', {
                    teams: teams.length,
                    size: mathSummary.bracketSize,
                    byes: mathSummary.byeCount,
                    matches: mathSummary.totalMatches,
                  })}
                </span>
              )}
            </div>
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
              {t('bracket.subtitle')}
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {onSwapMatchPairings && bracketMatches.length > 0 && (
              <button
                type="button"
                onClick={() => {
                  setEditPairingsMode((prev) => !prev);
                  setSelectedSlot(null);
                }}
                className={`inline-flex items-center gap-2 rounded-xl px-4 py-2 text-sm font-medium transition-all ${
                  editPairingsMode
                    ? 'bg-amber-500 text-white shadow-sm'
                    : 'border border-amber-300 dark:border-amber-700 bg-amber-50 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300 hover:bg-amber-100'
                }`}
              >
                <ArrowLeftRight className="h-4 w-4" />
                {editPairingsMode
                  ? t('fixtures.edit_pairings_active')
                  : t('fixtures.edit_pairings_mode')}
              </button>
            )}

            <span
              title={
                !canBuild
                  ? t('fixtures.tooltip_need_teams_knockout')
                  : t('bracket.build_btn')
              }
            >
              <button
                type="submit"
                disabled={building || !canBuild}
                title={
                  !canBuild
                    ? t('fixtures.tooltip_need_teams_knockout')
                    : t('bracket.build_btn')
                }
                className="inline-flex items-center gap-2 rounded-xl bg-purple-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-purple-700 disabled:cursor-not-allowed disabled:opacity-50 transition-all"
              >
                <GitBranch className="h-4 w-4" />
                {t('bracket.build_btn')}
              </button>
            </span>
          </div>
        </div>

        {editPairingsMode && (
          <div className="flex items-center justify-between rounded-xl border border-amber-300 dark:border-amber-700 bg-amber-50/90 dark:bg-amber-950/40 px-4 py-2.5 text-xs font-semibold text-amber-900 dark:text-amber-200">
            <div className="flex items-center gap-2">
              <ArrowLeftRight className="h-4 w-4 text-amber-600 dark:text-amber-400 shrink-0" />
              <span>
                {selectedSlot
                  ? `Seçili Slotta: "${selectedSlot.teamName}" — Yer değiştirmek istediğiniz diğer takım veya BAY kutusuna tıklayın.`
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

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2 border-t border-slate-100 dark:border-slate-800">
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

      {/* Horizontal Visual Bracket Tree */}
      {roundsList.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 p-10 text-center space-y-4">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400">
            <Inbox className="h-6 w-6" />
          </div>
          <div className="space-y-1 max-w-md mx-auto">
            <h4 className="text-base font-bold text-slate-900 dark:text-slate-100">
              {teams.length === 0 ? t('standings.empty') : t('nav.awaiting_draw')}
            </h4>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {t('bracket.empty')}
            </p>
          </div>
          {onManageTeams && (
            <div className="flex items-center justify-center gap-2">
              <button
                type="button"
                onClick={onManageTeams}
                className="inline-flex items-center justify-center gap-2 rounded-xl bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-xs hover:bg-blue-700 transition-colors"
              >
                <Users className="h-4 w-4" />
                {t('standings.manage_teams_cta')}
              </button>
            </div>
          )}
        </div>
      ) : (
        <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-slate-800 bg-slate-100/80 dark:bg-slate-900/70 p-6">
          <div className="inline-flex min-w-full flex-col space-y-6">
            {/* Main Left-to-Right Elimination Tree */}
            <div className="flex items-stretch gap-0">
              {roundsList.map((roundCol, colIdx) => {
                const isLastRound = colIdx === roundsList.length - 1;
                const headerLabel = getRoundHeaderLabel(
                  roundCol.matches.length,
                  roundCol.roundNumber
                );

                const pairs: Match[][] = [];
                for (let i = 0; i < roundCol.matches.length; i += 2) {
                  pairs.push(roundCol.matches.slice(i, i + 2));
                }

                return (
                  <div key={roundCol.roundNumber} className="flex flex-col">
                    {/* Round Column Header */}
                    <div className="mb-4 flex items-center justify-between rounded-xl bg-slate-900 dark:bg-slate-800 px-4 py-2 text-white shadow-xs w-72">
                      <span className="text-xs font-extrabold uppercase tracking-wider">
                        {headerLabel}
                      </span>
                      <span className="rounded-full bg-white/15 px-2 py-0.5 text-[10px] font-bold">
                        {roundCol.matches.length} Maç
                      </span>
                    </div>

                    {/* Vertically Distributed Match Pairs + Connector Arms */}
                    <div className="flex flex-1 flex-col justify-around gap-6">
                      {pairs.map((pair, pIdx) => (
                        <div key={pIdx} className="flex items-stretch">
                          <div className="flex flex-col justify-around gap-5 py-2">
                            {pair.map((m) =>
                              renderMatchNode(
                                m,
                                m.bracket_slot === 'final' ? 'gold' : 'default'
                              )
                            )}
                          </div>

                          {/* SVG / CSS Connector Lines to Next Round */}
                          {!isLastRound && pair.length === 2 && (
                            <div className="flex w-12 items-center">
                              <div className="h-1/2 w-6 border-y-2 border-r-2 border-slate-300 dark:border-slate-600 rounded-r-xl" />
                              <div className="w-6 border-t-2 border-slate-300 dark:border-slate-600" />
                            </div>
                          )}
                          {!isLastRound && pair.length === 1 && (
                            <div className="flex w-12 items-center">
                              <div className="w-12 border-t-2 border-slate-300 dark:border-slate-600" />
                            </div>
                          )}
                        </div>
                      ))}
                    </div>

                    {/* Dedicated 3rd Place Playoff Box below Final in the Last Column */}
                    {isLastRound && thirdPlaceMatch && (
                      <div className="mt-8 pt-6 border-t border-dashed border-slate-300 dark:border-slate-700">
                        <div className="mb-3 flex items-center gap-1.5 px-1 text-xs font-extrabold uppercase tracking-wider text-orange-700 dark:text-orange-400">
                          <Award className="h-4 w-4" />
                          {t('final_four.third_place_match')}
                        </div>
                        {renderMatchNode(thirdPlaceMatch, 'bronze')}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

import React, { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  X,
  Calendar,
  MapPin,
  CheckCircle2,
  ShieldAlert,
  Gavel,
  ArrowLeftRight,
  Target,
  Users,
} from 'lucide-react';
import { Match, MatchStatus, SportSlug, Team, Venue, venueSupportsSport } from '../types';

interface ScoreModalProps {
  match: Match | null;
  sportSlug: SportSlug;
  teams?: Team[];
  venues?: Venue[];
  isElimination?: boolean;
  onClose: () => void;
  onSave: (
    matchId: number,
    payload: {
      home_team_id?: number | null;
      away_team_id?: number | null;
      home_team_name?: string;
      away_team_name?: string;
      home_score: number;
      away_score: number;
      home_penalties: number | null;
      away_penalties: number | null;
      status: MatchStatus;
      is_forfeit: boolean;
      forfeit_winner_team_id: number | null;
      venue: string;
      court: string;
      scheduled_at: string;
      notes: string;
    }
  ) => Promise<void>;
}

function getOfficialForfeitWinScore(sportSlug: SportSlug): number {
  switch (sportSlug) {
    case 'futsal':
      return 5;
    case 'basketball':
      return 20;
    case 'volleyball':
      return 3;
    case 'handball':
      return 10;
    case 'football':
    default:
      return 3;
  }
}

export const ScoreModal: React.FC<ScoreModalProps> = ({
  match,
  sportSlug,
  teams = [],
  venues = [],
  isElimination = false,
  onClose,
  onSave,
}) => {
  const { t } = useTranslation();
  const matchingVenues = useMemo(
    () => venues.filter((v) => venueSupportsSport(v, sportSlug)),
    [venues, sportSlug]
  );
  const [homeTeamId, setHomeTeamId] = useState<number | null>(null);
  const [awayTeamId, setAwayTeamId] = useState<number | null>(null);
  const [homeTeamName, setHomeTeamName] = useState('');
  const [awayTeamName, setAwayTeamName] = useState('');
  const [homeScore, setHomeScore] = useState<number>(0);
  const [awayScore, setAwayScore] = useState<number>(0);
  const [homePenalties, setHomePenalties] = useState<string>('');
  const [awayPenalties, setAwayPenalties] = useState<string>('');
  const [status, setStatus] = useState<MatchStatus>('completed');
  const [isForfeit, setIsForfeit] = useState<boolean>(false);
  const [forfeitWinnerId, setForfeitWinnerId] = useState<number | null>(null);
  const [venue, setVenue] = useState('');
  const [court, setCourt] = useState('');
  const [scheduledAt, setScheduledAt] = useState('');
  const [notes, setNotes] = useState('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const isVolleyball = sportSlug === 'volleyball';
  const isFutsal = sportSlug === 'futsal';
  const officialWinScore = getOfficialForfeitWinScore(sportSlug);
  const effectiveElimination = isElimination || Boolean(match?.bracket_slot);

  useEffect(() => {
    if (!match) return;
    setErrorMsg(null);
    setHomeTeamId(match.home_team_id ?? null);
    setAwayTeamId(match.away_team_id ?? null);
    setHomeTeamName(match.home_team_name || '');
    setAwayTeamName(match.away_team_name || '');

    if (match.status === 'scheduled') {
      setHomeScore(0);
      setAwayScore(0);
      setHomePenalties('');
      setAwayPenalties('');
      setStatus('completed');
      setIsForfeit(false);
      setForfeitWinnerId(null);
    } else {
      setHomeScore(match.home_score ?? 0);
      setAwayScore(match.away_score ?? 0);
      setHomePenalties(
        match.home_penalties !== undefined && match.home_penalties !== null
          ? String(match.home_penalties)
          : ''
      );
      setAwayPenalties(
        match.away_penalties !== undefined && match.away_penalties !== null
          ? String(match.away_penalties)
          : ''
      );
      setStatus(match.status);
      setIsForfeit(Boolean(match.is_forfeit));
      setForfeitWinnerId(match.forfeit_winner_team_id ?? null);
    }
    setVenue(match.venue || '');
    setCourt(match.court || '');
    setScheduledAt(match.scheduled_at || '');
    setNotes(match.notes || '');
  }, [match]);

  if (!match) return null;

  const isRegulationDraw =
    status === 'completed' && !isForfeit && homeScore === awayScore;
  const showPenaltySection =
    isRegulationDraw && (effectiveElimination || isFutsal || sportSlug === 'football' || sportSlug === 'handball');

  const handleSwapHomeAway = () => {
    const prevHomeId = homeTeamId;
    const prevHomeName = homeTeamName;
    setHomeTeamId(awayTeamId);
    setHomeTeamName(awayTeamName);
    setAwayTeamId(prevHomeId);
    setAwayTeamName(prevHomeName);

    const prevHScore = homeScore;
    setHomeScore(awayScore);
    setAwayScore(prevHScore);

    const prevHPen = homePenalties;
    setHomePenalties(awayPenalties);
    setAwayPenalties(prevHPen);
  };

  const applyForfeitWinner = (winnerSide: 'home' | 'away') => {
    setIsForfeit(true);
    setStatus('completed');
    setHomePenalties('');
    setAwayPenalties('');
    if (winnerSide === 'home') {
      setHomeScore(officialWinScore);
      setAwayScore(0);
      setForfeitWinnerId(homeTeamId);
    } else {
      setHomeScore(0);
      setAwayScore(officialWinScore);
      setForfeitWinnerId(awayTeamId);
    }
  };

  const handleToggleForfeit = () => {
    if (isForfeit) {
      setIsForfeit(false);
      setForfeitWinnerId(null);
      setHomeScore(0);
      setAwayScore(0);
    } else {
      applyForfeitWinner('home');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    let parsedHomePen: number | null = null;
    let parsedAwayPen: number | null = null;

    if (status === 'completed' && !isForfeit && homeScore === awayScore) {
      const hasHP = homePenalties.trim() !== '';
      const hasAP = awayPenalties.trim() !== '';

      if (effectiveElimination) {
        if (!hasHP || !hasAP) {
          setErrorMsg(t('modals.elimination_penalty_hint'));
          return;
        }
        parsedHomePen = Math.max(0, parseInt(homePenalties, 10) || 0);
        parsedAwayPen = Math.max(0, parseInt(awayPenalties, 10) || 0);
        if (parsedHomePen === parsedAwayPen) {
          setErrorMsg(t('modals.elimination_penalty_hint'));
          return;
        }
      } else if (hasHP && hasAP) {
        parsedHomePen = Math.max(0, parseInt(homePenalties, 10) || 0);
        parsedAwayPen = Math.max(0, parseInt(awayPenalties, 10) || 0);
        if ((parsedHomePen > 0 || parsedAwayPen > 0) && parsedHomePen === parsedAwayPen) {
          setErrorMsg('Penaltı atışları berabere bitemez.');
          return;
        }
      }
    }

    setSaving(true);
    try {
      const effectiveForfeit = status === 'completed' && isForfeit;
      await onSave(match.id, {
        home_team_id: homeTeamId,
        away_team_id: awayTeamId,
        home_team_name: homeTeamName,
        away_team_name: awayTeamName,
        home_score: status === 'scheduled' ? 0 : homeScore,
        away_score: status === 'scheduled' ? 0 : awayScore,
        home_penalties: status === 'scheduled' ? null : parsedHomePen,
        away_penalties: status === 'scheduled' ? null : parsedAwayPen,
        status,
        is_forfeit: effectiveForfeit,
        forfeit_winner_team_id: effectiveForfeit ? forfeitWinnerId : null,
        venue,
        court,
        scheduled_at: scheduledAt,
        notes,
      });
      onClose();
    } catch (err: any) {
      setErrorMsg(err?.message || 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-3 backdrop-blur-xs overflow-y-auto">
      <div className="w-full max-w-xl overflow-hidden rounded-2xl bg-white shadow-2xl ring-1 ring-slate-200 my-auto dark:bg-slate-900 dark:ring-slate-800">
        <div className="flex items-center justify-between border-b border-slate-200 bg-slate-900 px-5 py-4 text-white dark:border-slate-800 dark:bg-slate-950">
          <div>
            <span className="text-xs font-semibold uppercase tracking-wider text-blue-400">
              {match.stage_name} • {t('fixtures.round')} {match.round} • #{match.match_number}
            </span>
            <h3 className="text-base font-bold">{t('modals.score_modal_title')}</h3>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-slate-400 hover:bg-slate-800 hover:text-white"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          {errorMsg && (
            <div className="rounded-xl border border-rose-300 bg-rose-50 px-3.5 py-2.5 text-xs font-bold text-rose-800 dark:border-rose-800 dark:bg-rose-950/60 dark:text-rose-200">
              {errorMsg}
            </div>
          )}

          {isVolleyball && !isForfeit && (
            <div className="rounded-lg bg-sky-50 border border-sky-200 px-3 py-2 text-xs font-medium text-sky-800 dark:bg-sky-950/50 dark:border-sky-800 dark:text-sky-200">
              {t('modals.volleyball_hint')}
            </div>
          )}

          {/* Editable Matchup Pairings Section */}
          {teams.length > 0 && (
            <div className="rounded-xl border border-slate-200 bg-slate-50/70 p-3 space-y-2.5 dark:border-slate-800 dark:bg-slate-800/50">
              <div className="flex items-center justify-between gap-2">
                <span className="inline-flex items-center gap-1.5 text-[11px] font-extrabold uppercase tracking-wider text-slate-600 dark:text-slate-300">
                  <Users className="h-3.5 w-3.5 text-indigo-500" />
                  {t('modals.edit_pairing_title')}
                </span>
                <button
                  type="button"
                  onClick={handleSwapHomeAway}
                  className="inline-flex items-center gap-1 rounded-lg border border-indigo-200 bg-white px-2.5 py-1 text-[11px] font-bold text-indigo-700 hover:bg-indigo-50 dark:border-indigo-800 dark:bg-slate-900 dark:text-indigo-300 dark:hover:bg-slate-800"
                >
                  <ArrowLeftRight className="h-3 w-3" />
                  {t('fixtures.swap_home_away')}
                </button>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                <select
                  value={homeTeamId ?? ''}
                  onChange={(e) => {
                    const val = e.target.value ? Number(e.target.value) : null;
                    setHomeTeamId(val);
                    const found = teams.find((tm) => tm.id === val);
                    if (found) setHomeTeamName(found.school_name);
                  }}
                  className="w-full rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-xs font-bold text-slate-900 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
                >
                  <option value="">{homeTeamName || 'Ev Sahibi'}</option>
                  {teams.map((tm) => (
                    <option key={tm.id} value={tm.id}>
                      {tm.school_name}
                    </option>
                  ))}
                </select>

                <select
                  value={awayTeamId ?? ''}
                  onChange={(e) => {
                    const val = e.target.value ? Number(e.target.value) : null;
                    setAwayTeamId(val);
                    const found = teams.find((tm) => tm.id === val);
                    if (found) setAwayTeamName(found.school_name);
                  }}
                  className="w-full rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-xs font-bold text-slate-900 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100"
                >
                  <option value="">{awayTeamName || 'Deplasman'}</option>
                  {teams.map((tm) => (
                    <option key={tm.id} value={tm.id}>
                      {tm.school_name}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          )}

          {/* Forfeit (Hükmen Galip Belirle) Toggle & Winner Selector */}
          <div className="rounded-xl border border-amber-200 bg-amber-50/70 p-3.5 space-y-3 dark:border-amber-800/60 dark:bg-amber-950/30">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 text-xs font-extrabold text-amber-900 dark:text-amber-300">
                <Gavel className="h-4 w-4 text-amber-600 dark:text-amber-400" />
                <span>{t('modals.forfeit_toggle')}</span>
              </div>
              <button
                type="button"
                onClick={handleToggleForfeit}
                className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-extrabold transition-all ${
                  isForfeit
                    ? 'bg-rose-600 text-white shadow-xs'
                    : 'bg-white text-amber-900 border border-amber-300 hover:bg-amber-100 dark:bg-slate-900 dark:text-amber-300 dark:border-amber-700'
                }`}
              >
                <ShieldAlert className="h-3.5 w-3.5" />
                {isForfeit ? t('modals.forfeit_active') : `[ ${t('modals.forfeit_toggle')} ]`}
              </button>
            </div>

            {isForfeit && (
              <div className="space-y-2.5 pt-1 border-t border-amber-200/80 dark:border-amber-800/60">
                <div className="text-[11px] font-bold text-amber-900 dark:text-amber-200">
                  {t('modals.forfeit_select_winner')}
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={() => applyForfeitWinner('home')}
                    className={`rounded-xl border px-3 py-2 text-xs font-extrabold transition-all ${
                      homeScore > awayScore
                        ? 'border-emerald-600 bg-emerald-600 text-white shadow-xs'
                        : 'border-amber-200 bg-white text-slate-800 hover:bg-amber-100/60 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200'
                    }`}
                  >
                    <div className="truncate">{homeTeamName}</div>
                    <div className="text-[10px] opacity-90 mt-0.5">
                      Hükmen ({officialWinScore} - 0)
                    </div>
                  </button>
                  <button
                    type="button"
                    onClick={() => applyForfeitWinner('away')}
                    className={`rounded-xl border px-3 py-2 text-xs font-extrabold transition-all ${
                      awayScore > homeScore
                        ? 'border-emerald-600 bg-emerald-600 text-white shadow-xs'
                        : 'border-amber-200 bg-white text-slate-800 hover:bg-amber-100/60 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200'
                    }`}
                  >
                    <div className="truncate">{awayTeamName}</div>
                    <div className="text-[10px] opacity-90 mt-0.5">
                      Hükmen (0 - {officialWinScore})
                    </div>
                  </button>
                </div>
                <p className="text-[11px] font-medium text-amber-800 dark:text-amber-300">
                  {t('modals.forfeit_rule_hint', {
                    score: `${officialWinScore} - 0`,
                  })}
                </p>
              </div>
            )}
          </div>

          {/* Main Scoreboard Input */}
          <div className="grid grid-cols-7 items-center gap-2 rounded-xl bg-slate-50 p-4 border border-slate-200 dark:bg-slate-800/70 dark:border-slate-700">
            <div className="col-span-3 text-center space-y-2">
              <div className="text-sm font-bold text-slate-900 dark:text-slate-100 line-clamp-2 min-h-[2.5rem] flex items-center justify-center">
                {homeTeamName}
              </div>
              <div className="flex items-center justify-center gap-1.5">
                <button
                  type="button"
                  disabled={isForfeit}
                  onClick={() => setHomeScore(Math.max(0, homeScore - 1))}
                  className="h-9 w-9 rounded-lg border border-slate-300 bg-white font-bold text-slate-700 hover:bg-slate-100 active:scale-95 disabled:opacity-40 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
                >
                  -
                </button>
                <input
                  type="number"
                  min={0}
                  disabled={isForfeit}
                  value={homeScore}
                  onChange={(e) =>
                    setHomeScore(Math.max(0, parseInt(e.target.value, 10) || 0))
                  }
                  className="h-12 w-16 rounded-lg border border-slate-300 bg-white text-center text-2xl font-extrabold text-slate-900 focus:border-blue-600 focus:outline-none disabled:bg-slate-100 dark:border-slate-700 dark:bg-slate-900 dark:text-white"
                />
                <button
                  type="button"
                  disabled={isForfeit}
                  onClick={() => setHomeScore(homeScore + 1)}
                  className="h-9 w-9 rounded-lg border border-slate-300 bg-white font-bold text-slate-700 hover:bg-slate-100 active:scale-95 disabled:opacity-40 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
                >
                  +
                </button>
              </div>
            </div>

            <div className="col-span-1 text-center font-black text-xl text-slate-400">
              VS
            </div>

            <div className="col-span-3 text-center space-y-2">
              <div className="text-sm font-bold text-slate-900 dark:text-slate-100 line-clamp-2 min-h-[2.5rem] flex items-center justify-center">
                {awayTeamName}
              </div>
              <div className="flex items-center justify-center gap-1.5">
                <button
                  type="button"
                  disabled={isForfeit}
                  onClick={() => setAwayScore(Math.max(0, awayScore - 1))}
                  className="h-9 w-9 rounded-lg border border-slate-300 bg-white font-bold text-slate-700 hover:bg-slate-100 active:scale-95 disabled:opacity-40 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
                >
                  -
                </button>
                <input
                  type="number"
                  min={0}
                  disabled={isForfeit}
                  value={awayScore}
                  onChange={(e) =>
                    setAwayScore(Math.max(0, parseInt(e.target.value, 10) || 0))
                  }
                  className="h-12 w-16 rounded-lg border border-slate-300 bg-white text-center text-2xl font-extrabold text-slate-900 focus:border-blue-600 focus:outline-none disabled:bg-slate-100 dark:border-slate-700 dark:bg-slate-900 dark:text-white"
                />
                <button
                  type="button"
                  disabled={isForfeit}
                  onClick={() => setAwayScore(awayScore + 1)}
                  className="h-9 w-9 rounded-lg border border-slate-300 bg-white font-bold text-slate-700 hover:bg-slate-100 active:scale-95 disabled:opacity-40 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
                >
                  +
                </button>
              </div>
            </div>
          </div>

          {/* Penalty Shootout Inputs (Futsal Group Draws & Final Four / Elimination Draws) */}
          {showPenaltySection && (
            <div className="rounded-xl border-2 border-indigo-200 bg-indigo-50/70 p-3.5 space-y-3 dark:border-indigo-800/70 dark:bg-indigo-950/30">
              <div className="flex items-center gap-2 text-xs font-extrabold text-indigo-900 dark:text-indigo-200">
                <Target className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
                <span>{t('modals.penalties_title')}</span>
              </div>
              <p className="text-[11px] font-medium text-indigo-800 dark:text-indigo-300">
                {effectiveElimination
                  ? t('modals.elimination_penalty_hint')
                  : t('modals.futsal_penalty_hint')}
              </p>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-[11px] font-bold text-indigo-900 dark:text-indigo-200 mb-1 truncate">
                    {t('modals.home_penalties')} ({homeTeamName})
                  </label>
                  <input
                    type="number"
                    min={0}
                    required={effectiveElimination}
                    placeholder="Örn: 5"
                    value={homePenalties}
                    onChange={(e) => setHomePenalties(e.target.value)}
                    className="w-full rounded-lg border border-indigo-300 bg-white px-3 py-2 text-center text-base font-extrabold text-indigo-950 focus:border-indigo-600 focus:outline-none dark:border-indigo-700 dark:bg-slate-900 dark:text-white"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-bold text-indigo-900 dark:text-indigo-200 mb-1 truncate">
                    {t('modals.away_penalties')} ({awayTeamName})
                  </label>
                  <input
                    type="number"
                    min={0}
                    required={effectiveElimination}
                    placeholder="Örn: 4"
                    value={awayPenalties}
                    onChange={(e) => setAwayPenalties(e.target.value)}
                    className="w-full rounded-lg border border-indigo-300 bg-white px-3 py-2 text-center text-base font-extrabold text-indigo-950 focus:border-indigo-600 focus:outline-none dark:border-indigo-700 dark:bg-slate-900 dark:text-white"
                  />
                </div>
              </div>
            </div>
          )}

          {/* Match Status (Completed or Scheduled) */}
          <div>
            <label className="block text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-400 mb-1.5">
              {t('modals.match_status')}
            </label>
            <div className="grid grid-cols-2 gap-2">
              {(['completed', 'scheduled'] as MatchStatus[]).map((st) => (
                <button
                  key={st}
                  type="button"
                  onClick={() => {
                    setStatus(st);
                    if (st === 'scheduled') {
                      setIsForfeit(false);
                      setForfeitWinnerId(null);
                      setHomePenalties('');
                      setAwayPenalties('');
                    }
                  }}
                  className={`rounded-lg border px-3 py-2 text-xs font-semibold transition-all ${
                    status === st
                      ? 'border-blue-600 bg-blue-600 text-white shadow-xs'
                      : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300'
                  }`}
                >
                  {t(`status.${st}`)}
                </button>
              ))}
            </div>
          </div>

          {/* Schedule & Venue Selector */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="flex items-center gap-1 text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1">
                <Calendar className="h-3.5 w-3.5 text-slate-400" />
                {t('fixtures.date_time')}
              </label>
              <input
                type="text"
                placeholder="2026-10-20 14:00"
                value={scheduledAt}
                onChange={(e) => setScheduledAt(e.target.value)}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm text-slate-900 focus:border-blue-600 focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              />
            </div>
            <div>
              <label className="flex items-center gap-1 text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1">
                <MapPin className="h-3.5 w-3.5 text-slate-400" />
                {t('venues.select_venue')}
              </label>
              <select
                value={venue}
                onChange={(e) => {
                  setVenue(e.target.value);
                  setCourt('');
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm text-slate-900 focus:border-blue-600 focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              >
                <option value="">{t('venues.no_venue_selected')}</option>
                {venue &&
                  !matchingVenues.some((v) => v.name === venue) && (
                    <option value={venue}>{venue}</option>
                  )}
                {matchingVenues.map((v) => (
                  <option key={v.id} value={v.name}>
                    {v.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-200 dark:border-slate-800">
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            >
              {t('modals.cancel')}
            </button>
            <button
              type="submit"
              disabled={saving}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-5 py-2 text-sm font-semibold text-white shadow-xs hover:bg-blue-700 disabled:opacity-50"
            >
              <CheckCircle2 className="h-4 w-4" />
              {t('modals.save')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

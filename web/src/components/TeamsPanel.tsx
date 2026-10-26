import React, { useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Users,
  Plus,
  Trash2,
  School,
  Upload,
  Download,
  Star,
  GitBranch,
  Sparkles,
  Inbox,
} from 'lucide-react';
import { Stage, Team } from '../types';

interface TeamsPanelProps {
  stages: Stage[];
  teams: Team[];
  currentLang: 'tr' | 'en';
  isKnockoutMode?: boolean;
  onAddStage?: (name: string) => Promise<void>;
  onDeleteStage: (stageId: number) => Promise<void>;
  onAddTeam: (schoolName: string) => Promise<void>;
  onUpdateTeamStage: (teamId: number, stageId: number | null) => Promise<void>;
  onToggleTeamSeed: (teamId: number, isSeeded: boolean) => Promise<void>;
  onOpenSeedingModal: () => void;
  onBuildKnockout?: () => Promise<void> | void;
  onImportExcel: (
    file: File
  ) => Promise<number | { imported_count: number; skipped_count?: number }>;
  onDeleteTeam: (teamId: number) => Promise<void>;
}

export const TeamsPanel: React.FC<TeamsPanelProps> = ({
  stages,
  teams,
  currentLang,
  isKnockoutMode = false,
  onDeleteStage,
  onAddTeam,
  onUpdateTeamStage,
  onToggleTeamSeed,
  onOpenSeedingModal,
  onBuildKnockout,
  onImportExcel,
  onDeleteTeam,
}) => {
  const { t } = useTranslation();
  const [schoolName, setSchoolName] = useState('');
  const [importing, setImporting] = useState(false);
  const [importBanner, setImportBanner] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const schoolInputRef = useRef<HTMLInputElement | null>(null);

  const handleTeamSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!schoolName.trim()) return;
    await onAddTeam(schoolName.trim());
    setSchoolName('');
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setImporting(true);
    setImportBanner(null);
    try {
      const result = await onImportExcel(file);
      const count = typeof result === 'number' ? result : result.imported_count;
      const skipped =
        typeof result === 'object' && result !== null
          ? result.skipped_count || 0
          : 0;
      if (skipped > 0) {
        setImportBanner(
          t('teams.import_success_with_skipped', { count, skipped })
        );
      } else {
        setImportBanner(t('teams.import_success', { count }));
      }
    } catch (err: any) {
      setImportBanner(err?.message || 'Import error');
    } finally {
      setImporting(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleDownloadTemplate = () => {
    window.location.href = `/api/teams/template-excel?lang=${currentLang}`;
  };

  const minTeamsRequired = isKnockoutMode ? 2 : 3;
  const canDraw = teams.length >= minTeamsRequired;
  const seededCount = teams.filter((tm) => tm.is_seeded).length;
  const activeStagesWithTeams = stages.filter((s) => (s.teams?.length || 0) > 0);

  return (
    <div className="space-y-6">
      {/* Top Row: Category Team Registration + Draw Readiness Card */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5 items-stretch">
        {/* Left Card: Kategoriye Okul / Takım Ekle */}
        <form
          onSubmit={handleTeamSubmit}
          className="lg:col-span-2 border border-slate-200 dark:border-zinc-800 rounded-xl bg-white dark:bg-zinc-900 shadow-sm p-5 sm:p-6 flex flex-col justify-between gap-4"
        >
          {/* Card Header aligned on exact vertical baseline */}
          <div className="flex flex-wrap items-center justify-between gap-2 min-h-[36px]">
            <div className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
              <School className="h-4 w-4 text-blue-600 dark:text-blue-400 shrink-0" />
              <span>{t('teams.add_team')}</span>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <button
                type="button"
                onClick={handleDownloadTemplate}
                className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-slate-300 dark:border-zinc-700 bg-slate-50 dark:bg-zinc-800 px-3 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-zinc-700 transition"
              >
                <Download className="h-3.5 w-3.5 text-slate-600 dark:text-slate-300" />
                {t('teams.download_template')}
              </button>

              <input
                ref={fileInputRef}
                type="file"
                id="team-excel-input"
                accept=".xlsx, .xls, application/vnd.ms-excel, application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                onChange={handleFileChange}
                className="hidden"
              />
              <button
                type="button"
                disabled={importing}
                onClick={() => fileInputRef.current?.click()}
                className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 px-3.5 py-1.5 text-xs font-medium text-white disabled:opacity-50 transition"
              >
                <Upload className="h-3.5 w-3.5" />
                {importing ? t('teams.importing') : t('teams.import_excel')}
              </button>
            </div>
          </div>

          {importBanner && (
            <div className="rounded-lg bg-emerald-50 border border-emerald-200 px-3.5 py-2 text-xs font-semibold text-emerald-800 dark:bg-emerald-950/40 dark:border-emerald-800 dark:text-emerald-300">
              {importBanner}
            </div>
          )}

          {/* Non-overlapping Flex Input + "+ Oluştur" Button */}
          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400 mb-1">
              {t('teams.school_name')}
            </label>
            <div className="flex items-center gap-2 mt-1">
              <input
                ref={schoolInputRef}
                type="text"
                required
                placeholder={t('teams.school_placeholder')}
                value={schoolName}
                onChange={(e) => setSchoolName(e.target.value)}
                className="flex-1 rounded-lg border border-slate-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-slate-900 dark:text-slate-100 px-3.5 py-2 text-sm focus:ring-2 focus:ring-primary focus:outline-none"
              />
              <button
                type="submit"
                className="shrink-0 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg flex items-center gap-1.5 transition"
              >
                <Plus size={16} />
                {t('modals.create')}
              </button>
            </div>
          </div>
        </form>

        {/* Right Card: Kura Çek ve Fikstür Oluştur (Aligned on same vertical baseline) */}
        <div className="border border-slate-200 dark:border-zinc-800 rounded-xl bg-white dark:bg-zinc-900 shadow-sm p-5 sm:p-6 flex flex-col justify-between gap-4">
          <div className="space-y-2.5">
            <div className="flex items-center justify-between gap-2 min-h-[36px]">
              <div className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
                {isKnockoutMode ? (
                  <GitBranch className="h-4 w-4 text-purple-600 dark:text-purple-400 shrink-0" />
                ) : (
                  <Sparkles className="h-4 w-4 text-blue-600 dark:text-blue-400 shrink-0" />
                )}
                <span>{t('fixtures.draw_btn')}</span>
              </div>
              <span className="rounded-full bg-slate-100 dark:bg-zinc-800 px-2.5 py-0.5 text-xs font-semibold text-slate-700 dark:text-slate-300">
                {teams.length} {t('teams.team_count')}
              </span>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
              {isKnockoutMode ? t('bracket.subtitle') : t('seeding.subtitle')}
            </p>
            <div className="flex flex-wrap items-center gap-2 pt-0.5">
              <span className="inline-flex items-center gap-1 rounded-lg bg-amber-50 border border-amber-200 px-2.5 py-1 text-xs font-semibold text-amber-800 dark:bg-amber-950/40 dark:border-amber-800 dark:text-amber-300">
                <Star className="h-3 w-3 fill-amber-500 text-amber-500" />
                Seri Başı: {seededCount}
              </span>
              {!isKnockoutMode && teams.length > 0 && (
                <span className="inline-flex items-center gap-1 rounded-lg bg-blue-50 border border-blue-200 px-2.5 py-1 text-xs font-semibold text-blue-800 dark:bg-blue-950/40 dark:border-blue-800 dark:text-blue-300">
                  Tahmini: {Math.max(1, Math.ceil(teams.length / 4))} Grup
                </span>
              )}
            </div>
          </div>

          <div
            title={
              !canDraw
                ? t('fixtures.tooltip_need_teams')
                : t('fixtures.draw_btn')
            }
          >
            <button
              type="button"
              disabled={!canDraw}
              title={
                !canDraw
                  ? t('fixtures.tooltip_need_teams')
                  : t('fixtures.draw_btn')
              }
              onClick={() =>
                isKnockoutMode && onBuildKnockout
                  ? onBuildKnockout()
                  : onOpenSeedingModal()
              }
              className="inline-flex w-full h-10 items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-xs hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50 transition"
            >
              <Sparkles className="h-4 w-4" />
              {t('fixtures.draw_btn')}
            </button>
          </div>
        </div>
      </div>

      {/* Category Team Pool & Group Assignment Table */}
      <div className="overflow-hidden border border-slate-200 dark:border-zinc-800 rounded-xl bg-white dark:bg-zinc-900 shadow-sm">
        <div className="flex items-center justify-between border-b border-slate-200 dark:border-zinc-800 bg-slate-900 dark:bg-zinc-950 px-5 py-3.5 text-white">
          <div className="flex items-center gap-2">
            <Users className="h-4 w-4 text-blue-400" />
            <span className="text-sm font-semibold">
              {isKnockoutMode ? t('nav.participating_teams') : t('teams.all_category_teams')}
            </span>
          </div>
          <span className="rounded-full bg-slate-800 px-2.5 py-0.5 text-xs font-medium text-slate-300">
            {teams.length} {t('teams.team_count')}
          </span>
        </div>

        {teams.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-10 text-center space-y-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-slate-100 text-slate-500 dark:bg-zinc-800 dark:text-slate-400">
              <Inbox className="h-6 w-6" />
            </div>
            <div className="space-y-1 max-w-sm">
              <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                {t('standings.empty')}
              </h4>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Yukarıdaki okul ekleme formundan veya Excel şablonundan bu kategoriye takım ekleyebilirsiniz.
              </p>
            </div>
            <button
              type="button"
              onClick={() => schoolInputRef.current?.focus()}
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-xs hover:bg-blue-700 transition"
            >
              <Plus className="h-4 w-4" />
              {t('standings.manage_teams_cta')}
            </button>
          </div>
        ) : (
          <div className="divide-y divide-slate-100 dark:divide-zinc-800">
            {teams.map((tm, idx) => (
              <div
                key={tm.id}
                className="flex flex-wrap items-center justify-between gap-3 px-5 py-3 hover:bg-slate-50 dark:hover:bg-zinc-800/50 transition-colors"
              >
                <div className="flex items-center gap-3">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-slate-100 text-xs font-bold text-slate-600 dark:bg-zinc-800 dark:text-slate-300">
                    {idx + 1}
                  </span>
                  <div className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
                    <span>{tm.school_name}</span>
                    {tm.is_seeded && (
                      <span className="inline-flex items-center gap-1 rounded-md bg-amber-100 px-2 py-0.5 text-[10px] font-bold text-amber-800 dark:bg-amber-500/20 dark:text-amber-300">
                        <Star className="h-2.5 w-2.5 fill-amber-500 text-amber-600 dark:text-amber-400" />
                        {t('standings.seeded_badge')}
                      </span>
                    )}
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => onToggleTeamSeed(tm.id, !tm.is_seeded)}
                    title={t('standings.seeded_badge')}
                    className={`inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs font-medium transition-colors ${
                      tm.is_seeded
                        ? 'border-amber-300 bg-amber-100 text-amber-800 dark:border-amber-600/50 dark:bg-amber-500/20 dark:text-amber-300'
                        : 'border-slate-200 bg-white text-slate-500 hover:text-amber-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-slate-400'
                    }`}
                  >
                    <Star
                      className={`h-3.5 w-3.5 ${
                        tm.is_seeded ? 'fill-amber-500 text-amber-600 dark:text-amber-400' : ''
                      }`}
                    />
                    <span className="hidden sm:inline">{t('standings.seeded_badge')}</span>
                  </button>

                  {!isKnockoutMode && activeStagesWithTeams.length > 0 && (
                    <select
                      value={tm.stage_id ?? ''}
                      onChange={(e) => {
                        const val = e.target.value;
                        onUpdateTeamStage(tm.id, val === '' ? null : Number(val));
                      }}
                      className={`rounded-lg border px-2.5 py-1.5 text-xs font-medium ${
                        tm.stage_id
                          ? 'border-blue-200 bg-blue-50 text-blue-800 dark:border-blue-800 dark:bg-blue-950/60 dark:text-blue-200'
                          : 'border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-300'
                      }`}
                    >
                      <option value="">{t('teams.unassigned')}</option>
                      {activeStagesWithTeams.map((s) => (
                        <option key={s.id} value={s.id}>
                          {s.name}
                        </option>
                      ))}
                    </select>
                  )}

                  <button
                    onClick={() => onDeleteTeam(tm.id)}
                    title={t('teams.delete_team')}
                    className="rounded-lg p-1.5 text-slate-400 hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/50 dark:hover:text-rose-400 transition-colors"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Stage / Group Cards (Only rendered AFTER draw has created groups with teams) */}
      {!isKnockoutMode && activeStagesWithTeams.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
          {activeStagesWithTeams.map((stage) => (
            <div
              key={stage.id}
              className="overflow-hidden border border-slate-200 dark:border-zinc-800 rounded-xl bg-white dark:bg-zinc-900 shadow-sm"
            >
              <div className="flex items-center justify-between border-b border-slate-200 dark:border-zinc-800 bg-slate-900 dark:bg-zinc-950 px-4 py-3 text-white">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-sm">{stage.name}</span>
                  <span className="rounded-full bg-slate-800 px-2 py-0.5 text-xs text-slate-200">
                    {stage.teams?.length || 0} {t('teams.team_count')}
                  </span>
                </div>
                {activeStagesWithTeams.length > 1 && (
                  <button
                    onClick={() => onDeleteStage(stage.id)}
                    title={t('teams.delete_stage')}
                    className="text-slate-400 hover:text-rose-400 p-1"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                )}
              </div>

              <ul className="divide-y divide-slate-100 dark:divide-zinc-800">
                {stage.teams.map((tm, idx) => (
                  <li
                    key={tm.id}
                    className="flex items-center justify-between px-4 py-2.5 text-sm"
                  >
                    <span className="font-medium text-slate-800 dark:text-slate-200">
                      {idx + 1}. {tm.school_name}
                    </span>
                    {tm.is_seeded && idx === 0 && (
                      <span className="inline-flex items-center gap-1 rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-bold text-amber-800 dark:bg-amber-500/20 dark:text-amber-300">
                        <Star className="h-2.5 w-2.5 fill-amber-500" />
                        {t('standings.seeded_badge')}
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

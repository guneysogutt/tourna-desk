import React, { useEffect, useState, useCallback, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Trophy,
  Calendar,
  Settings,
  FileSpreadsheet,
  Plus,
  Users,
  Globe,
  Trash2,
  LayoutGrid,
  Layers,
  GitBranch,
  Star,
  ChevronRight,
  ChevronLeft,
  Sun,
  Moon,
  RotateCcw,
  AlertTriangle,
  Edit3,
  Info,
  Sparkles,
  Inbox,
  CheckCircle2,
  Clock,
  Building2,
  X,
} from 'lucide-react';
import { setAppLanguage } from './i18n';
import { showConfirmModal } from './utils/confirmModal';
import {
  CategoryDetail,
  Match,
  MatchStatus,
  SportSlug,
  Tournament,
  Venue,
} from './types';
import {
  SportBadge,
  SportIcon,
  getSportBadgeClasses,
} from './components/SportBadge';
import { StandingsTable } from './components/StandingsTable';
import { FixturePanel } from './components/FixturePanel';
import { TeamsPanel } from './components/TeamsPanel';
import { ScoreModal } from './components/ScoreModal';
import { SeedingModal } from './components/SeedingModal';
import { PotManagementPanel } from './components/PotManagementPanel';
import { FinalFourBracket } from './components/FinalFourBracket';
import { BracketTree } from './components/BracketTree';
import { VenueManagerModal } from './components/VenueManagerModal';

type ActiveTab =
  | 'stage_1'
  | 'pots_draw'
  | 'stage_2'
  | 'final_four'
  | 'knockout_bracket'
  | 'teams';

export const App: React.FC = () => {
  const { t, i18n } = useTranslation();
  const currentLang = (i18n.language?.startsWith('en') ? 'en' : 'tr') as 'tr' | 'en';

  // Dark Mode State (System preference + LocalStorage persistence)
  const [isDark, setIsDark] = useState<boolean>(() => {
    try {
      const saved = localStorage.getItem('tourna_theme');
      if (saved === 'dark') return true;
      if (saved === 'light') return false;
      return window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? false;
    } catch {
      return false;
    }
  });

  useEffect(() => {
    const root = document.documentElement;
    if (isDark) {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }
    try {
      localStorage.setItem('tourna_theme', isDark ? 'dark' : 'light');
    } catch {
      // ignore storage errors
    }
  }, [isDark]);

  // Collapsible Left Sidebar State
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState<boolean>(() => {
    try {
      return localStorage.getItem('tourna_sidebar_collapsed') === 'true';
    } catch {
      return false;
    }
  });

  const toggleSidebar = () => {
    setIsSidebarCollapsed((prev) => {
      const next = !prev;
      try {
        localStorage.setItem('tourna_sidebar_collapsed', String(next));
      } catch {
        // ignore
      }
      return next;
    });
  };

  // Horizontal Resizable Sidebar State (220px..480px, persisted in localStorage under sidebar_width)
  const [sidebarWidth, setSidebarWidth] = useState<number>(() => {
    try {
      const raw = localStorage.getItem('sidebar_width');
      if (raw) {
        const parsed = parseInt(raw, 10);
        if (!Number.isNaN(parsed) && parsed >= 220 && parsed <= 480) {
          return parsed;
        }
      }
    } catch {
      // ignore
    }
    return 272;
  });
  const [isResizingSidebar, setIsResizingSidebar] = useState(false);

  const handleSidebarResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizingSidebar(true);
  }, []);

  useEffect(() => {
    if (!isResizingSidebar) return;

    const handleMouseMove = (e: MouseEvent) => {
      const clientX = e.clientX;
      if (clientX < 100) {
        setIsSidebarCollapsed(true);
        try {
          localStorage.setItem('tourna_sidebar_collapsed', 'true');
        } catch {
          // ignore
        }
        return;
      }
      const clamped = Math.max(220, Math.min(480, clientX));
      setIsSidebarCollapsed(false);
      setSidebarWidth(clamped);
      try {
        localStorage.setItem('tourna_sidebar_collapsed', 'false');
        localStorage.setItem('sidebar_width', String(clamped));
      } catch {
        // ignore
      }
    };

    const handleMouseUp = () => {
      setIsResizingSidebar(false);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isResizingSidebar]);

  const [tournaments, setTournaments] = useState<Tournament[]>([]);
  const [venues, setVenues] = useState<Venue[]>([]);
  const [showVenueManagerModal, setShowVenueManagerModal] = useState(false);
  const [activeSeasonId, setActiveSeasonId] = useState<number | null>(null);
  const [selectedCategoryId, setSelectedCategoryId] = useState<number | null>(null);
  const [categoryDetail, setCategoryDetail] = useState<CategoryDetail | null>(null);
  const [activeTab, setActiveTab] = useState<ActiveTab>('stage_1');
  const [selectedGroupFilter, setSelectedGroupFilter] = useState<number | 'all'>('all');
  const [editingMatch, setEditingMatch] = useState<Match | null>(null);
  const [showSeedingModal, setShowSeedingModal] = useState(false);
  const [showResetConfirmModal, setShowResetConfirmModal] = useState(false);
  const [resettingFixtures, setResettingFixtures] = useState(false);
  const [showRulesPopover, setShowRulesPopover] = useState(false);
  const rulesPopoverRef = useRef<HTMLDivElement | null>(null);
  const [showSeasonManagerModal, setShowSeasonManagerModal] = useState(false);

  // Close Scoring Rules Popover on outside click or Escape key
  useEffect(() => {
    if (!showRulesPopover) return;

    const handleOutsideClick = (e: MouseEvent) => {
      if (
        rulesPopoverRef.current &&
        !rulesPopoverRef.current.contains(e.target as Node)
      ) {
        setShowRulesPopover(false);
      }
    };

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setShowRulesPopover(false);
      }
    };

    document.addEventListener('mousedown', handleOutsideClick);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handleOutsideClick);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [showRulesPopover]);

  // Modals for Season Creation & Category Creation / Editing
  const [showNewTournament, setShowNewTournament] = useState(false);
  const [tName, setTName] = useState('');
  const [tStart, setTStart] = useState('2026');
  const [tEnd, setTEnd] = useState('2027');

  const [showNewCategoryForTournament, setShowNewCategoryForTournament] = useState<
    number | null
  >(null);
  const [cName, setCName] = useState('');
  const [cSport, setCSport] = useState<SportSlug>('football');
  const [cFormat, setCFormat] = useState<'groups' | 'knockout'>('groups');

  // Quick Edit Category Modal State
  const [showEditCategoryModal, setShowEditCategoryModal] = useState(false);
  const [editCatName, setEditCatName] = useState('');
  const [editCatSport, setEditCatSport] = useState<SportSlug>('football');
  const [editCatFormat, setEditCatFormat] = useState<'groups' | 'knockout'>('groups');

  const fetchTournaments = useCallback(
    async (preserveSelection = true) => {
      const res = await fetch('/api/tournaments');
      if (!res.ok) return;
      const data: Tournament[] = await res.json();
      setTournaments(data);

      if (data.length === 0) {
        setActiveSeasonId(null);
        setSelectedCategoryId(null);
        setCategoryDetail(null);
        return;
      }

      if (!preserveSelection || selectedCategoryId === null) {
        const firstSeason = data[0];
        setActiveSeasonId(firstSeason.id);
        for (const trn of data) {
          if (trn.categories && trn.categories.length > 0) {
            setActiveSeasonId(trn.id);
            setSelectedCategoryId(trn.categories[0].id);
            return;
          }
        }
        setSelectedCategoryId(null);
        setCategoryDetail(null);
      } else {
        const ownerSeason = data.find((trn) =>
          trn.categories?.some((c) => c.id === selectedCategoryId)
        );
        if (ownerSeason) {
          setActiveSeasonId(ownerSeason.id);
        } else if (activeSeasonId === null) {
          setActiveSeasonId(data[0].id);
        }
      }
    },
    [selectedCategoryId, activeSeasonId]
  );

  const fetchCategoryDetail = useCallback(async (catId: number) => {
    const res = await fetch(`/api/categories/${catId}`);
    if (!res.ok) return;
    const detail: CategoryDetail = await res.json();
    setCategoryDetail(detail);
    const hasKO = detail.stages?.some((s) => s.stage_type === 'knockout');
    const hasGrp = detail.stages?.some((s) => s.stage_type === 'group');
    const isKnockout =
      detail.category?.format === 'knockout' || (hasKO && !hasGrp);
    if (isKnockout) {
      setActiveTab((prev) =>
        prev === 'teams' || prev === 'knockout_bracket' ? prev : 'knockout_bracket'
      );
    } else {
      setActiveTab((prev) =>
        prev === 'knockout_bracket' && !hasKO ? 'stage_1' : prev
      );
    }
  }, []);

  const fetchVenues = useCallback(async () => {
    try {
      const res = await fetch('/api/venues');
      if (!res.ok) return;
      const data: Venue[] = await res.json();
      setVenues(data);
    } catch {
      // ignore network error
    }
  }, []);

  const handleCreateVenue = async (payload: {
    name: string;
    address: string;
    supported_sports: string[];
  }) => {
    const res = await fetch('/api/venues', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (res.ok) {
      await fetchVenues();
    }
  };

  const handleUpdateVenue = async (
    id: number,
    payload: {
      name: string;
      address: string;
      supported_sports: string[];
    }
  ) => {
    const res = await fetch(`/api/venues/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (res.ok) {
      await fetchVenues();
    }
  };

  const handleDeleteVenue = async (id: number) => {
    const confirmed = await showConfirmModal({
      title: t('nav.venues'),
      message: t('modals.confirm_delete'),
      confirmText: t('modals.delete'),
      cancelText: t('modals.cancel'),
      variant: 'danger',
    });
    if (!confirmed) return;
    const res = await fetch(`/api/venues/${id}`, { method: 'DELETE' });
    if (res.ok) {
      await fetchVenues();
    }
  };

  useEffect(() => {
    fetchTournaments(false);
    fetchVenues();
  }, []);

  useEffect(() => {
    if (selectedCategoryId !== null) {
      setSelectedGroupFilter('all');
      setShowRulesPopover(false);
      fetchCategoryDetail(selectedCategoryId);
    }
  }, [selectedCategoryId, fetchCategoryDetail]);

  const activeSeason = useMemo(() => {
    if (tournaments.length === 0) return null;
    return (
      tournaments.find((trn) => trn.id === activeSeasonId) || tournaments[0]
    );
  }, [tournaments, activeSeasonId]);

  const handleSeasonSelectChange = (seasonId: number) => {
    setActiveSeasonId(seasonId);
    const targetSeason = tournaments.find((trn) => trn.id === seasonId);
    if (targetSeason && targetSeason.categories?.length > 0) {
      setSelectedCategoryId(targetSeason.categories[0].id);
    } else {
      setSelectedCategoryId(null);
      setCategoryDetail(null);
    }
  };

  const handleCreateTournament = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tName.trim()) return;
    const res = await fetch('/api/tournaments', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: tName.trim(),
        description: '',
        location: '',
        start_date: tStart.trim(),
        end_date: tEnd.trim(),
      }),
    });
    if (res.ok) {
      const created: Tournament = await res.json();
      setShowNewTournament(false);
      setTName('');
      await fetchTournaments(true);
      setActiveSeasonId(created.id);
    }
  };

  const handleDeleteTournament = async (id: number) => {
    const confirmed = await showConfirmModal({
      title: t('nav.tournaments'),
      message: t('modals.confirm_delete'),
      confirmText: t('modals.delete'),
      cancelText: t('modals.cancel'),
      variant: 'danger',
    });
    if (!confirmed) return;
    await fetch(`/api/tournaments/${id}`, { method: 'DELETE' });
    await fetchTournaments(false);
  };

  const handleCreateCategory = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!showNewCategoryForTournament || !cName.trim()) return;
    const res = await fetch('/api/categories', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        tournament_id: showNewCategoryForTournament,
        sport_slug: cSport,
        name: cName.trim(),
        format: cFormat,
      }),
    });
    if (res.ok) {
      const created = await res.json();
      setShowNewCategoryForTournament(null);
      setCName('');
      await fetchTournaments(true);
      setActiveSeasonId(created.tournament_id);
      setSelectedCategoryId(created.id);
    }
  };

  const handleOpenEditCategory = () => {
    if (!categoryDetail) return;
    setEditCatName(categoryDetail.category.name);
    setEditCatSport(categoryDetail.category.sport_slug);
    setEditCatFormat(
      categoryDetail.category.format === 'knockout' ? 'knockout' : 'groups'
    );
    setShowEditCategoryModal(true);
  };

  const handleUpdateCategory = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedCategoryId || !editCatName.trim()) return;
    const res = await fetch(`/api/categories/${selectedCategoryId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: editCatName.trim(),
        sport_slug: editCatSport,
        format: editCatFormat,
      }),
    });
    if (res.ok) {
      const updatedDetail: CategoryDetail = await res.json();
      setCategoryDetail(updatedDetail);
      setShowEditCategoryModal(false);
      await fetchTournaments(true);
    }
  };

  const handleResetFixtures = async () => {
    if (!selectedCategoryId) return;
    setResettingFixtures(true);
    try {
      const res = await fetch(
        `/api/categories/${selectedCategoryId}/reset-fixtures`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({}),
        }
      );
      if (res.ok) {
        const updatedDetail: CategoryDetail = await res.json();
        setCategoryDetail(updatedDetail);
        setSelectedGroupFilter('all');
        setShowResetConfirmModal(false);
        await fetchTournaments(true);
      }
    } finally {
      setResettingFixtures(false);
    }
  };

  const handleTriggerResetFixtures = async () => {
    if (!selectedCategoryId) return;
    const confirmed = await showConfirmModal({
      title: t('fixtures.reset_confirm_title'),
      message: t('fixtures.reset_confirm_body'),
      confirmText: t('fixtures.reset_confirm_action'),
      cancelText: t('modals.cancel'),
      variant: 'danger',
    });
    if (!confirmed) return;
    await handleResetFixtures();
  };

  const handleDeleteCategory = async (catId: number, e: React.MouseEvent) => {
    e.stopPropagation();
    const confirmed = await showConfirmModal({
      title: t('nav.categories'),
      message: t('modals.confirm_delete'),
      confirmText: t('modals.delete'),
      cancelText: t('modals.cancel'),
      variant: 'danger',
    });
    if (!confirmed) return;
    await fetch(`/api/categories/${catId}`, { method: 'DELETE' });
    await fetchTournaments(false);
  };

  const handleSaveMatchScore = async (
    matchId: number,
    payload: {
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
      home_team_id?: number | null;
      away_team_id?: number | null;
      home_team_name?: string;
      away_team_name?: string;
      round?: number;
      match_number?: number;
    }
  ) => {
    const res = await fetch(`/api/matches/${matchId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || 'Failed to update match');
    }
    const updatedDetail: CategoryDetail = await res.json();
    setCategoryDetail(updatedDetail);
  };

  const handleSwapMatchPairings = async (payload: {
    match_a_id: number;
    slot_a: 'home' | 'away';
    match_b_id: number;
    slot_b: 'home' | 'away';
  }) => {
    if (!selectedCategoryId) return;
    const res = await fetch(`/api/categories/${selectedCategoryId}/matches/swap`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (res.ok) {
      const updatedDetail: CategoryDetail = await res.json();
      setCategoryDetail(updatedDetail);
    }
  };

  const handleRunInitialDraw = async (payload: {
    seeded_team_ids: number[];
    format?: 'group' | 'knockout';
    generate_fixtures: boolean;
    double_round_robin: boolean;
    default_venue: string;
    default_court: string;
    start_date: string;
  }) => {
    if (!selectedCategoryId) return;
    const res = await fetch(`/api/categories/${selectedCategoryId}/initial-draw`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || 'Initial draw failed');
    }
    const updatedDetail: CategoryDetail = await res.json();
    setCategoryDetail(updatedDetail);
    setSelectedGroupFilter('all');
    if (payload.format === 'knockout') {
      setActiveTab('knockout_bracket');
    } else {
      setActiveTab('stage_1');
    }
  };

  const handlePreparePots = async (stageIds?: number[]) => {
    if (!selectedCategoryId) return;
    const res = await fetch(`/api/categories/${selectedCategoryId}/pots/prepare`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ stage_ids: stageIds || [] }),
    });
    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || 'Failed to prepare pots');
    }
    const updatedDetail: CategoryDetail = await res.json();
    setCategoryDetail(updatedDetail);
  };

  const handleSavePots = async (
    pot1: { team_id: number; stage_id: number | null }[],
    pot2: { team_id: number; stage_id: number | null }[]
  ) => {
    if (!selectedCategoryId) return;
    const res = await fetch(`/api/categories/${selectedCategoryId}/pots`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pot1, pot2 }),
    });
    if (res.ok) {
      const updatedDetail: CategoryDetail = await res.json();
      setCategoryDetail(updatedDetail);
    }
  };

  const handleExecuteNextRoundDraw = async (payload: {
    pot1_team_ids: number[];
    pot2_team_ids: number[];
    format?: 'group' | 'knockout';
    generate_fixtures: boolean;
    default_venue: string;
    default_court: string;
    start_date: string;
  }) => {
    if (!selectedCategoryId) return;
    const res = await fetch(
      `/api/categories/${selectedCategoryId}/next-round-draw`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      }
    );
    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || 'Next round draw failed');
    }
    const updatedDetail: CategoryDetail = await res.json();
    setCategoryDetail(updatedDetail);
    setSelectedGroupFilter('all');
    if (payload.format === 'knockout') {
      setActiveTab('knockout_bracket');
    } else {
      setActiveTab('stage_2');
    }
  };

  const handleBuildFinalFour = async (payload: {
    group_a_stage_id: number;
    group_b_stage_id: number;
    default_venue: string;
    default_court: string;
    start_date: string;
  }) => {
    if (!selectedCategoryId) return;
    const res = await fetch(`/api/categories/${selectedCategoryId}/final-four`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || 'Final Four generation failed');
    }
    const updatedDetail: CategoryDetail = await res.json();
    setCategoryDetail(updatedDetail);
    setActiveTab('final_four');
  };

  const handleBuildKnockout = async (payload: {
    team_ids?: number[];
    seeded_team_ids?: number[];
    default_venue: string;
    default_court: string;
    start_date: string;
  }) => {
    if (!selectedCategoryId) return;
    const res = await fetch(`/api/categories/${selectedCategoryId}/knockout`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || 'Knockout generation failed');
    }
    const updatedDetail: CategoryDetail = await res.json();
    setCategoryDetail(updatedDetail);
    setActiveTab('knockout_bracket');
  };

  const handleDeleteStage = async (stageId: number) => {
    if (!selectedCategoryId) return;
    const confirmed = await showConfirmModal({
      title: t('teams.delete_stage'),
      message: t('modals.confirm_delete'),
      confirmText: t('modals.delete'),
      cancelText: t('modals.cancel'),
      variant: 'danger',
    });
    if (!confirmed) return;
    await fetch(`/api/stages/${stageId}`, { method: 'DELETE' });
    await fetchCategoryDetail(selectedCategoryId);
  };

  const handleAddTeam = async (schoolName: string) => {
    if (!selectedCategoryId) return;
    const res = await fetch(`/api/categories/${selectedCategoryId}/teams`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        school_name: schoolName,
      }),
    });
    if (res.ok) {
      await fetchCategoryDetail(selectedCategoryId);
    }
  };

  const handleUpdateTeamStage = async (teamId: number, stageId: number | null) => {
    if (!selectedCategoryId) return;
    const res = await fetch(`/api/teams/${teamId}/stage`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ stage_id: stageId }),
    });
    if (res.ok) {
      await fetchCategoryDetail(selectedCategoryId);
    }
  };

  const handleToggleTeamSeed = async (teamId: number, isSeeded: boolean) => {
    if (!selectedCategoryId) return;
    const res = await fetch(`/api/teams/${teamId}/seed`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ is_seeded: isSeeded }),
    });
    if (res.ok) {
      await fetchCategoryDetail(selectedCategoryId);
    }
  };

  const handleImportTeamsExcel = async (
    file: File
  ): Promise<{ imported_count: number; skipped_count: number }> => {
    if (!selectedCategoryId) return { imported_count: 0, skipped_count: 0 };
    const formData = new FormData();
    formData.append('file', file);
    const res = await fetch(
      `/api/categories/${selectedCategoryId}/teams/import-excel`,
      {
        method: 'POST',
        body: formData,
      }
    );
    if (!res.ok) {
      const errData = await res.json().catch(() => ({}));
      throw new Error(errData.error || 'Import failed');
    }
    const data = await res.json();
    if (data.detail) {
      setCategoryDetail(data.detail);
    } else {
      await fetchCategoryDetail(selectedCategoryId);
    }
    return {
      imported_count: data.imported_count || 0,
      skipped_count: data.skipped_count || 0,
    };
  };

  const handleDeleteTeam = async (teamId: number) => {
    if (!selectedCategoryId) return;
    const confirmed = await showConfirmModal({
      title: t('teams.delete_team'),
      message: t('modals.confirm_delete'),
      confirmText: t('modals.delete'),
      cancelText: t('modals.cancel'),
      variant: 'danger',
    });
    if (!confirmed) return;
    await fetch(`/api/teams/${teamId}`, { method: 'DELETE' });
    await fetchCategoryDetail(selectedCategoryId);
  };

  const handleExportExcel = () => {
    if (!selectedCategoryId) return;
    window.location.href = `/api/categories/${selectedCategoryId}/export-excel?lang=${currentLang}`;
  };

  const handleExportBracketExcel = () => {
    if (!selectedCategoryId) return;
    window.location.href = `/api/categories/${selectedCategoryId}/export-bracket-excel?lang=${currentLang}`;
  };

  // Compact Scoring Rule Badges per Sport
  const getSportRulePills = (slug: SportSlug): string[] => {
    switch (slug) {
      case 'basketball':
        return ['G: 2P', 'M: 1P', 'Hükmen: 0P'];
      case 'volleyball':
        return ['3-0/3-1: 3P', '3-2: 2P', '2-3: 1P', '0-3/1-3: 0P'];
      case 'handball':
        return ['G: 2P', 'B: 1P', 'M: 0P'];
      case 'futsal':
      case 'football':
      default:
        return ['G: 3P', 'B: 1P', 'M: 0P'];
    }
  };

  // Detailed Scoring Rule Breakdown for Click-Activated Popover
  const getSportRuleDetails = (slug: SportSlug) => {
    const isEN = currentLang === 'en';
    switch (slug) {
      case 'basketball':
        return {
          win: isEN ? '2 Points' : '2 Puan',
          draw: isEN ? 'None (Overtime played)' : 'Yok (Uzatma oynanır)',
          loss: isEN
            ? '1 Point (Played match) • 0 Points on Forfeit'
            : '1 Puan (Oynanan maç) • Hükmen mağlubiyette 0 Puan',
          tiebreak: isEN
            ? 'Points → Point Diff → Points For → Head-to-Head'
            : 'Puan → Genel Sayı Averajı → Atılan Sayı → İkili Averaj',
          forfeitScore: '20 - 0',
        };
      case 'volleyball':
        return {
          win: isEN
            ? '3-0 / 3-1: 3 Pts • 3-2: 2 Pts'
            : '3-0 / 3-1: 3 Puan • 3-2: 2 Puan',
          draw: isEN ? 'None (Set-based)' : 'Yok (Set usulü)',
          loss: isEN
            ? '2-3: 1 Pt • 0-3 / 1-3 & Forfeit: 0 Pts'
            : '2-3: 1 Puan • 0-3 / 1-3 ve Hükmen: 0 Puan',
          tiebreak: isEN
            ? 'Wins → Points → Set Ratio (SW/SL) → Set Diff'
            : 'Galibiyet Sayısı → Puan → Set Oranı (AS/YS) → Set Farkı',
          forfeitScore: '3 - 0',
        };
      case 'futsal':
        return {
          win: isEN ? '3 Points' : '3 Puan',
          draw: isEN
            ? '1 Point (Penalties used for tiebreak resolution)'
            : '1 Puan (Penaltı atışları eşitlik bozmada kullanılır)',
          loss: isEN ? '0 Points' : '0 Puan',
          tiebreak: isEN
            ? 'Points → Goal Diff → Goals For → H2H / Penalties'
            : 'Puan → Genel Averaj → Atılan Gol → İkili Averaj / Penaltı',
          forfeitScore: '5 - 0',
        };
      case 'handball':
        return {
          win: isEN ? '2 Points' : '2 Puan',
          draw: isEN ? '1 Point' : '1 Puan',
          loss: isEN ? '0 Points' : '0 Puan',
          tiebreak: isEN
            ? 'Points → Goal Diff → Goals For → Head-to-Head'
            : 'Puan → Genel Averaj → Atılan Gol → İkili Averaj',
          forfeitScore: '10 - 0',
        };
      case 'football':
      default:
        return {
          win: isEN ? '3 Points' : '3 Puan',
          draw: isEN ? '1 Point' : '1 Puan',
          loss: isEN ? '0 Points' : '0 Puan',
          tiebreak: isEN
            ? 'Points → Goal Diff → Goals For → Head-to-Head'
            : 'Puan → Genel Averaj → Atılan Gol → İkili Averaj',
          forfeitScore: '3 - 0',
        };
    }
  };

  // Partition stages into Kademe 1 (round_number === 1, group) and Kademe 2+ (round_number >= 2, group)
  // Prevent Premature "Grup A" Shells: only consider a stage as an active drawn group if it has teams or matches
  const stage1Groups = useMemo(
    () =>
      categoryDetail?.stages.filter(
        (s) =>
          s.stage_type === 'group' &&
          (s.round_number || 1) === 1 &&
          ((s.teams?.length || 0) > 0 || (s.matches?.length || 0) > 0)
      ) || [],
    [categoryDetail]
  );

  const stage2Groups = useMemo(
    () =>
      categoryDetail?.stages.filter(
        (s) =>
          s.stage_type === 'group' &&
          (s.round_number || 1) >= 2 &&
          ((s.teams?.length || 0) > 0 || (s.matches?.length || 0) > 0)
      ) || [],
    [categoryDetail]
  );

  const stage1TeamCount = useMemo(
    () => stage1Groups.reduce((sum, stg) => sum + (stg.teams?.length || 0), 0),
    [stage1Groups]
  );

  const stage2TeamCount = useMemo(
    () => stage2Groups.reduce((sum, stg) => sum + (stg.teams?.length || 0), 0),
    [stage2Groups]
  );

  const hasFinalFourStage =
    categoryDetail?.stages.some(
      (s) => s.stage_type === 'final_four' && (s.matches?.length || 0) > 0
    ) || false;

  const hasKnockoutStage =
    categoryDetail?.stages.some(
      (s) => s.stage_type === 'knockout' && (s.matches?.length || 0) > 0
    ) || false;

  const isKnockoutCategory =
    categoryDetail?.category.format === 'knockout' ||
    (hasKnockoutStage && stage1Groups.length === 0 && stage2Groups.length === 0);

  const hasActiveMatches = (categoryDetail?.matches.length || 0) > 0;
  const hasDrawnStages =
    hasActiveMatches || stage1Groups.length > 0 || stage2Groups.length > 0;

  const registeredTeamCount = categoryDetail?.teams.length || 0;
  const seededTeamCount =
    categoryDetail?.teams.filter((tm) => tm.is_seeded).length || 0;
  const minTeamsForDraw = isKnockoutCategory ? 2 : 3;
  const canExecuteDraw = registeredTeamCount >= minTeamsForDraw;

  const drawTooltipText = !canExecuteDraw
    ? t('fixtures.tooltip_need_teams')
    : t('fixtures.draw_btn');

  const resetTooltipText = !hasDrawnStages
    ? t('fixtures.tooltip_no_reset')
    : t('fixtures.reset_btn');

  const exportTooltipText = !hasDrawnStages
    ? t('fixtures.tooltip_no_export')
    : t('app.export_excel');

  // Clarified Ambiguous Counters for Tab Labels
  const stage1TabLabel =
    stage1Groups.length > 0
      ? `${t('nav.stage_1')} (${stage1Groups.length} Grup • ${stage1TeamCount} Takım)`
      : `${t('nav.stage_1')} (${t('nav.awaiting_draw')})`;

  const stage2TabLabel =
    stage2Groups.length > 0
      ? `${t('nav.stage_2')} (${stage2Groups.length} Grup • ${stage2TeamCount} Takım)`
      : `${t('nav.stage_2')} (${t('nav.not_started')})`;

  const renderKademeGroupView = (
    kademeStages: typeof stage1Groups,
    isSecondKademe = false
  ) => {
    // Prevent Premature "Grup A" Shells: Show high-level "Kura Bekleniyor" setup card before draw is finalized
    if (kademeStages.length === 0) {
      if (isSecondKademe) {
        return (
          <div className="rounded-2xl border border-dashed border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 p-10 text-center space-y-4">
            <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-indigo-50 text-indigo-600 dark:bg-indigo-950/60 dark:text-indigo-400">
              <Layers className="h-6 w-6" />
            </div>
            <div className="space-y-1 max-w-md mx-auto">
              <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
                2. Kademe ({t('nav.not_started')})
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                2. Kademe grupları henüz oluşturulmadı. Önce 1. Kademe maçlarını tamamlayıp{' '}
                <strong>{t('nav.pots_draw')}</strong> sekmesinden üst tur kurasını çekin.
              </p>
            </div>
            <div className="flex flex-wrap items-center justify-center gap-2">
              <button
                type="button"
                onClick={() => setActiveTab('pots_draw')}
                className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors"
              >
                <Layers className="h-4 w-4" />
                {t('nav.pots_draw')}
              </button>
              {hasFinalFourStage && (
                <button
                  type="button"
                  onClick={() => setActiveTab('final_four')}
                  className="inline-flex items-center gap-2 rounded-xl bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700 transition-colors"
                >
                  <GitBranch className="h-4 w-4" />
                  {t('nav.final_four')}
                </button>
              )}
            </div>
          </div>
        );
      }

      // Stage 1: If 0 teams registered, show Actionable Empty State Card
      if (registeredTeamCount === 0) {
        return (
          <div className="border border-slate-200 dark:border-zinc-800 rounded-xl bg-white dark:bg-zinc-900 shadow-sm p-6 sm:p-10 text-center space-y-4">
            <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-xl bg-slate-100 text-slate-500 dark:bg-zinc-800 dark:text-slate-400">
              <Inbox className="h-7 w-7" />
            </div>
            <div className="space-y-1.5 max-w-md mx-auto">
              <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
                {t('standings.empty')}
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
                Kura çekimi ve fikstür oluşturma adımına geçebilmek için önce bu kategoriye okul takımlarını ekleyin.
              </p>
            </div>
            <div className="flex items-center justify-center pt-1">
              <button
                type="button"
                onClick={() => setActiveTab('teams')}
                className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-xs hover:bg-blue-700 transition-colors"
              >
                <Users className="h-4 w-4" />
                {t('standings.manage_teams_cta')}
              </button>
            </div>
          </div>
        );
      }

      // Stage 1: High-Level "Kura Bekleniyor" Setup Card (when teams exist, before draw is finalized)
      const estimatedGroups = Math.max(1, Math.ceil(registeredTeamCount / 4));
      return (
        <div className="border border-slate-200 dark:border-zinc-800 rounded-xl bg-white dark:bg-zinc-900 p-5 sm:p-6 shadow-sm space-y-6">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-100 dark:border-zinc-800 pb-5">
            <div className="flex items-start gap-3.5">
              <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-amber-50 text-amber-600 ring-1 ring-amber-200 dark:bg-amber-950/50 dark:text-amber-400 dark:ring-amber-800">
                <Clock className="h-5 w-5" />
              </div>
              <div className="space-y-1">
                <div className="flex flex-wrap items-center gap-2">
                  <h3 className="text-lg font-bold text-slate-900 dark:text-slate-100">
                    Kura Bekleniyor
                  </h3>
                  <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-semibold text-amber-800 dark:bg-amber-950/60 dark:text-amber-300">
                    Hazırlık Aşaması
                  </span>
                </div>
                <p className="text-xs text-slate-500 dark:text-slate-400 max-w-2xl">
                  Takımlar kaydedildi. <strong>{t('fixtures.draw_btn')}</strong> butonuna tıklayarak seri başı takımları 1. sıraya yerleştirip resmi rotasyon anahtarına (1-4, 2-3) göre grupları ve maç fikstürünü oluşturabilirsiniz.
                </p>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-2.5 shrink-0">
              <button
                type="button"
                onClick={() => setActiveTab('teams')}
                className="inline-flex h-10 items-center justify-center gap-2 rounded-lg border border-slate-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-zinc-700 transition-colors"
              >
                <Users className="h-4 w-4" />
                {t('standings.manage_teams_cta')}
              </button>

              <span title={drawTooltipText} className="inline-flex">
                <button
                  type="button"
                  disabled={!canExecuteDraw}
                  title={drawTooltipText}
                  onClick={() => setShowSeedingModal(true)}
                  className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-xs hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50 transition-colors"
                >
                  <Sparkles className="h-4 w-4" />
                  {t('fixtures.draw_btn')}
                </button>
              </span>
            </div>
          </div>

          {/* Summary Metrics */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div className="rounded-xl border border-slate-200 dark:border-zinc-800 bg-slate-50/70 dark:bg-zinc-800/40 p-4">
              <div className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                Kayıtlı Takım Sayısı
              </div>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-2xl font-bold text-slate-900 dark:text-slate-100">
                  {registeredTeamCount}
                </span>
                <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
                  {registeredTeamCount < 3
                    ? `(En az 3 takım gerekli)`
                    : 'Kura için hazır'}
                </span>
              </div>
            </div>

            <div className="rounded-xl border border-slate-200 dark:border-zinc-800 bg-slate-50/70 dark:bg-zinc-800/40 p-4">
              <div className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                Seri Başı (Seeded) Takımlar
              </div>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-2xl font-bold text-amber-600 dark:text-amber-400">
                  {seededTeamCount}
                </span>
                <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
                  Grup 1. sıraya yerleşir
                </span>
              </div>
            </div>

            <div className="rounded-xl border border-slate-200 dark:border-zinc-800 bg-slate-50/70 dark:bg-zinc-800/40 p-4">
              <div className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                Tahmini Grup Yapısı
              </div>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-2xl font-bold text-blue-600 dark:text-blue-400">
                  {estimatedGroups} Grup
                </span>
                <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
                  4&apos;lü Lig Usulü
                </span>
              </div>
            </div>
          </div>

          {/* Registered Teams Preview Pills */}
          <div className="space-y-2.5">
            <div className="text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              Kura Havuzundaki Okul Takımları ({registeredTeamCount})
            </div>
            <div className="flex flex-wrap gap-2">
              {categoryDetail?.teams.map((tm, idx) => (
                <div
                  key={tm.id}
                  className={`inline-flex items-center gap-2 rounded-lg border px-3 py-1.5 text-xs font-semibold ${
                    tm.is_seeded
                      ? 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200'
                      : 'border-slate-200 bg-slate-50 text-slate-700 dark:border-zinc-800 dark:bg-zinc-800/70 dark:text-slate-300'
                  }`}
                >
                  <span className="text-slate-400 font-mono">{idx + 1}.</span>
                  <span>{tm.school_name}</span>
                  {tm.is_seeded && (
                    <Star className="h-3.5 w-3.5 fill-amber-500 text-amber-500" />
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      );
    }

    const visibleStages =
      selectedGroupFilter === 'all'
        ? kademeStages
        : kademeStages.filter((s) => s.id === selectedGroupFilter);

    const effectiveStages = visibleStages.length > 0 ? visibleStages : kademeStages;

    return (
      <div className="space-y-4">
        {/* Group Filter Pills */}
        {kademeStages.length > 1 && (
          <div className="flex flex-wrap items-center gap-2 rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-4 shadow-xs">
            <span className="text-xs font-extrabold uppercase tracking-wider text-slate-400 px-2">
              Grup Seçimi:
            </span>
            <button
              type="button"
              onClick={() => setSelectedGroupFilter('all')}
              className={`rounded-xl px-4 py-2 text-sm font-medium transition-all ${
                selectedGroupFilter === 'all'
                  ? 'bg-slate-900 dark:bg-blue-600 text-white shadow-xs'
                  : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700'
              }`}
            >
              {t('fixtures.all_stages')} ({kademeStages.length} Grup)
            </button>
            {kademeStages.map((stg) => (
              <button
                key={stg.id}
                type="button"
                onClick={() => setSelectedGroupFilter(stg.id)}
                className={`rounded-xl px-4 py-2 text-sm font-medium transition-all ${
                  selectedGroupFilter === stg.id
                    ? 'bg-blue-600 text-white shadow-xs'
                    : 'bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700'
                }`}
              >
                {stg.name} ({stg.teams?.length || 0} Takım)
              </button>
            ))}
          </div>
        )}

        {/* Standings Tables for Selected Group(s) */}
        <div
          className={`grid grid-cols-1 ${
            effectiveStages.length > 1 ? '2xl:grid-cols-2' : ''
          } gap-4`}
        >
          {effectiveStages.map((stg) => (
            <StandingsTable
              key={stg.id}
              stageName={stg.name}
              sportSlug={categoryDetail!.category.sport_slug}
              standings={categoryDetail!.standings[stg.id] || []}
              showSeededBadge={!isSecondKademe && (stg.round_number || 1) === 1}
              onManageTeams={() => setActiveTab('teams')}
            />
          ))}
        </div>

        {/* Round-Pill Filtered Fixtures for Selected Group(s) */}
        <FixturePanel
          stages={kademeStages}
          matches={categoryDetail!.matches}
          selectedStageFilter={
            effectiveStages.length === 1 && selectedGroupFilter !== 'all'
              ? effectiveStages[0].id
              : 'all'
          }
          onSelectMatch={(m) => setEditingMatch(m)}
          onSwapMatchPairings={handleSwapMatchPairings}
          onOpenSeedingModal={() => setShowSeedingModal(true)}
        />
      </div>
    );
  };

  return (
    <div className="h-screen overflow-hidden flex flex-col lg:flex-row bg-slate-50 dark:bg-zinc-950 text-slate-900 dark:text-slate-100 transition-colors">
      {/* Resizable, Contained Left Sidebar Navigation */}
      <aside
        style={{
          width: isSidebarCollapsed ? '64px' : `${sidebarWidth}px`,
        }}
        className={`relative shrink-0 border-b lg:border-b-0 lg:border-r border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 flex flex-col h-screen overflow-hidden ${
          isResizingSidebar ? '' : 'transition-[width] duration-150 ease-out'
        }`}
      >
        {/* Horizontal Drag-to-Resize Handle */}
        <div
          id="sidebar-resizer"
          onMouseDown={handleSidebarResizeStart}
          className="w-1 hover:w-1.5 cursor-col-resize bg-transparent hover:bg-blue-500/50 transition-all absolute right-0 top-0 bottom-0 z-20"
        />

        <div className="flex flex-col flex-1 min-h-0 w-full overflow-hidden">
          {/* 1. Brand Header & Collapse Toggle */}
          {isSidebarCollapsed ? (
            <div className="flex flex-col items-center gap-1.5 border-b border-slate-200 dark:border-zinc-800 py-3 px-2">
              <div
                title={t('app.title')}
                className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-600 text-white shadow-xs"
              >
                <Trophy className="h-5 w-5" />
              </div>
              <button
                type="button"
                onClick={toggleSidebar}
                title={t('app.expand_sidebar')}
                className="flex h-10 w-10 items-center justify-center rounded-lg text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-zinc-800 transition-colors"
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          ) : (
            <div className="flex items-center justify-between gap-2 border-b border-slate-200 dark:border-zinc-800 px-3.5 py-3">
              <div className="flex items-center gap-2.5 min-w-0">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-600 text-white shadow-xs">
                  <Trophy className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <h1 className="truncate text-sm font-bold tracking-tight text-slate-900 dark:text-white">
                    {t('app.title')}
                  </h1>
                  <p className="truncate text-[11px] font-medium text-slate-500 dark:text-slate-400">
                    {t('app.subtitle')}
                  </p>
                </div>
              </div>

              <button
                type="button"
                onClick={toggleSidebar}
                title={t('app.collapse_sidebar')}
                className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 dark:border-zinc-700 bg-slate-50 dark:bg-zinc-800 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-zinc-700 transition-colors"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
            </div>
          )}

          {/* 2. Active Season Selector Directly Beneath Brand Header */}
          {!isSidebarCollapsed ? (
            <div className="border-b border-slate-200 dark:border-zinc-800 p-3.5 space-y-2">
              <div className="flex items-center justify-between">
                <span className="flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  <Calendar className="h-3.5 w-3.5 text-blue-600 dark:text-blue-400" />
                  {t('nav.active_season')}
                </span>
                <button
                  type="button"
                  onClick={() => setShowNewTournament(true)}
                  className="inline-flex items-center gap-1 rounded-lg bg-blue-50 dark:bg-blue-950/50 px-2 py-1 text-[11px] font-semibold text-blue-600 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-900/50 transition-colors"
                >
                  <Plus className="h-3 w-3" />
                  {t('nav.new_tournament')}
                </button>
              </div>

              {tournaments.length > 0 ? (
                <select
                  value={activeSeason?.id || ''}
                  onChange={(e) => handleSeasonSelectChange(Number(e.target.value))}
                  className="w-full rounded-lg border border-slate-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 px-3 py-1.5 text-xs font-semibold text-slate-900 dark:text-slate-100 focus:border-blue-600 focus:outline-none"
                >
                  {tournaments.map((trn) => (
                    <option key={trn.id} value={trn.id}>
                      {trn.name}
                      {trn.start_date ? ` (${trn.start_date})` : ''}
                    </option>
                  ))}
                </select>
              ) : (
                <div className="text-xs text-slate-500 dark:text-slate-400">
                  Henüz aktif sezon bulunmuyor.
                </div>
              )}
            </div>
          ) : (
            <div className="flex flex-col items-center border-b border-slate-200 dark:border-zinc-800 py-2 px-2">
              <button
                type="button"
                onClick={() => setShowNewTournament(true)}
                title={t('nav.new_tournament')}
                className="flex h-10 w-10 items-center justify-center rounded-lg text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-zinc-800 transition-colors"
              >
                <Calendar className="h-4 w-4" />
              </button>
            </div>
          )}

          {/* 3. Categorized Menu Items with Dedicated Sport Badges & Icons */}
          <div className="flex-1 overflow-y-auto p-2.5 space-y-3">
            {!isSidebarCollapsed ? (
              <div className="space-y-2">
                <div className="flex items-center justify-between px-1.5">
                  <span className="text-[11px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                    {t('nav.categories')} ({activeSeason?.categories?.length || 0})
                  </span>
                  {activeSeason && (
                    <button
                      type="button"
                      onClick={() => setShowNewCategoryForTournament(activeSeason.id)}
                      className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-[11px] font-semibold text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-zinc-800 transition-colors"
                    >
                      <Plus className="h-3.5 w-3.5" />
                      {t('nav.new_category')}
                    </button>
                  )}
                </div>

                {activeSeason && activeSeason.categories?.length > 0 ? (
                  <nav className="space-y-1">
                    {activeSeason.categories.map((cat) => {
                      const isSelected = cat.id === selectedCategoryId;
                      const isCatKnockout = cat.format === 'knockout';
                      const sportSlug = cat.sport || cat.sport_slug;
                      return (
                        <div
                          key={cat.id}
                          onClick={() => setSelectedCategoryId(cat.id)}
                          className={`group flex cursor-pointer items-center justify-between gap-2.5 rounded-lg px-2.5 py-2 transition-all ${
                            isSelected
                              ? 'bg-blue-600 text-white shadow-xs font-semibold'
                              : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-zinc-800'
                          }`}
                        >
                          <div className="flex items-center gap-2.5 min-w-0 flex-1">
                            <div
                              className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border ${
                                isSelected
                                  ? 'border-white/20 bg-white/20 text-white'
                                  : getSportBadgeClasses(sportSlug)
                              }`}
                            >
                              <SportIcon sport={sportSlug} className="h-4 w-4" />
                            </div>
                            <div className="min-w-0 flex-1">
                              <div className="truncate text-xs font-semibold">
                                {cat.name}
                              </div>
                              <div className="mt-0.5 flex items-center gap-1.5">
                                <span
                                  className={`text-[10px] font-medium ${
                                    isSelected
                                      ? 'text-blue-100'
                                      : 'text-slate-500 dark:text-slate-400'
                                  }`}
                                >
                                  {t(`sports.${sportSlug}`)}
                                </span>
                                <span
                                  className={`text-[10px] ${
                                    isSelected
                                      ? 'text-blue-200'
                                      : 'text-slate-300 dark:text-zinc-600'
                                  }`}
                                >
                                  •
                                </span>
                                <span
                                  className={`text-[10px] font-semibold ${
                                    isSelected
                                      ? 'text-white'
                                      : isCatKnockout
                                      ? 'text-purple-600 dark:text-purple-400'
                                      : 'text-slate-500 dark:text-slate-400'
                                  }`}
                                >
                                  {isCatKnockout ? 'Eleme' : 'Grup'}
                                </span>
                              </div>
                            </div>
                          </div>

                          <button
                            onClick={(e) => handleDeleteCategory(cat.id, e)}
                            title={t('modals.delete')}
                            className={`opacity-0 group-hover:opacity-100 rounded-lg p-1 transition-opacity ${
                              isSelected
                                ? 'text-blue-100 hover:bg-white/20 hover:text-white'
                                : 'text-slate-400 hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/50'
                            }`}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      );
                    })}
                  </nav>
                ) : (
                  <div className="rounded-xl border border-dashed border-slate-300 dark:border-zinc-800 p-4 text-center space-y-2">
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                      Bu sezonda henüz kategori yok.
                    </p>
                    {activeSeason && (
                      <button
                        type="button"
                        onClick={() => setShowNewCategoryForTournament(activeSeason.id)}
                        className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700"
                      >
                        <Plus className="h-3.5 w-3.5" />
                        {t('nav.new_category')}
                      </button>
                    )}
                  </div>
                )}
              </div>
            ) : (
              /* Collapsed Icon Rail (w-10 h-10 centered buttons with rounded-lg hover states) */
              <div className="flex flex-col items-center gap-1.5">
                {activeSeason?.categories?.map((cat) => {
                  const isSelected = cat.id === selectedCategoryId;
                  const sportSlug = cat.sport || cat.sport_slug;
                  return (
                    <button
                      key={cat.id}
                      type="button"
                      onClick={() => setSelectedCategoryId(cat.id)}
                      title={`${cat.name} (${t(`sports.${sportSlug}`)})`}
                      className={`flex h-10 w-10 items-center justify-center rounded-lg border transition-colors ${
                        isSelected
                          ? 'border-blue-600 bg-blue-600 text-white shadow-xs'
                          : `${getSportBadgeClasses(sportSlug)} hover:opacity-90`
                      }`}
                    >
                      <SportIcon sport={sportSlug} className="h-4 w-4" />
                    </button>
                  );
                })}
                {activeSeason && (
                  <button
                    type="button"
                    onClick={() => setShowNewCategoryForTournament(activeSeason.id)}
                    title={t('nav.new_category')}
                    className="flex h-10 w-10 items-center justify-center rounded-lg text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-zinc-800 transition-colors"
                  >
                    <Plus className="h-4 w-4" />
                  </button>
                )}
              </div>
            )}
          </div>
        </div>

        {/* 4. Strictly Contained Sidebar Footer: Tesis Yönetimi, Sezon Yönetimi & Theme / Language Controls */}
        {isSidebarCollapsed ? (
          <div className="mt-auto p-2 border-t border-slate-200 dark:border-zinc-800 w-full overflow-hidden flex flex-col items-center gap-1">
            <button
              type="button"
              onClick={() => setShowVenueManagerModal(true)}
              title={t('nav.venues')}
              className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-zinc-800 transition-colors"
            >
              <Building2 className="h-4 w-4" />
            </button>

            <button
              type="button"
              onClick={() => setShowSeasonManagerModal(true)}
              title={t('nav.tournaments')}
              className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-zinc-800 transition-colors"
            >
              <Settings className="h-4 w-4" />
            </button>

            <button
              id="theme-toggle"
              type="button"
              onClick={() => setIsDark((prev) => !prev)}
              title={isDark ? t('app.light_mode') : t('app.dark_mode')}
              className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-zinc-800 transition-colors"
            >
              {isDark ? (
                <Sun className="h-4 w-4 text-amber-400" />
              ) : (
                <Moon className="h-4 w-4 text-slate-600" />
              )}
            </button>

            <button
              type="button"
              onClick={() => setAppLanguage(currentLang === 'tr' ? 'en' : 'tr')}
              title="TR / EN"
              className="flex h-9 w-9 items-center justify-center rounded-lg text-[11px] font-bold text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-zinc-800 transition-colors"
            >
              {currentLang.toUpperCase()}
            </button>
          </div>
        ) : (
          <div className="mt-auto p-3 border-t border-slate-200 dark:border-zinc-800 w-full overflow-hidden flex flex-col gap-2">
            {/* Global Venue Management (Tesis Yönetimi) Link/Button */}
            <button
              type="button"
              onClick={() => setShowVenueManagerModal(true)}
              title={t('nav.venues')}
              className="w-full flex items-center justify-between gap-2 px-2.5 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-zinc-800 truncate transition-colors"
            >
              <div className="flex items-center gap-2 min-w-0 truncate">
                <Building2 className="h-4 w-4 text-blue-600 dark:text-blue-400 shrink-0" />
                <span className="truncate">{t('nav.venues')}</span>
              </div>
              <span className="shrink-0 rounded bg-slate-100 dark:bg-zinc-800 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600 dark:text-slate-400">
                {venues.length}
              </span>
            </button>

            {/* Top Row: Sezon Yönetimi Link/Button with text truncation */}
            <button
              type="button"
              onClick={() => setShowSeasonManagerModal(true)}
              title={t('nav.tournaments')}
              className="w-full flex items-center justify-between gap-2 px-2.5 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-zinc-800 truncate transition-colors"
            >
              <div className="flex items-center gap-2 min-w-0 truncate">
                <Settings className="h-4 w-4 text-slate-500 dark:text-slate-400 shrink-0" />
                <span className="truncate">{t('nav.tournaments')}</span>
              </div>
              <span className="shrink-0 rounded bg-slate-100 dark:bg-zinc-800 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600 dark:text-slate-400">
                {tournaments.length}
              </span>
            </button>

            {/* Bottom Row: Dark mode + Lang Switcher neatly spaced */}
            <div className="flex items-center justify-between pt-1">
              <button
                id="theme-toggle"
                type="button"
                onClick={() => setIsDark((prev) => !prev)}
                title={isDark ? t('app.light_mode') : t('app.dark_mode')}
                className="p-1.5 text-slate-500 hover:text-slate-900 dark:text-zinc-400 dark:hover:text-zinc-100 rounded-md hover:bg-slate-100 dark:hover:bg-zinc-800 transition-colors"
              >
                {isDark ? (
                  <Sun className="h-4 w-4 text-amber-400" />
                ) : (
                  <Moon className="h-4 w-4" />
                )}
              </button>

              <div className="flex items-center rounded-md border border-slate-200 dark:border-zinc-700 p-0.5 text-[11px]">
                <button
                  type="button"
                  onClick={() => setAppLanguage('tr')}
                  className={`px-1.5 py-0.5 rounded font-semibold transition-colors ${
                    currentLang === 'tr'
                      ? 'bg-blue-600 text-white'
                      : 'text-slate-500 hover:text-slate-900 dark:text-zinc-400 dark:hover:text-zinc-100'
                  }`}
                >
                  TR
                </button>
                <button
                  type="button"
                  onClick={() => setAppLanguage('en')}
                  className={`px-1.5 py-0.5 rounded font-semibold transition-colors ${
                    currentLang === 'en'
                      ? 'bg-blue-600 text-white'
                      : 'text-slate-500 hover:text-slate-900 dark:text-zinc-400 dark:hover:text-zinc-100'
                  }`}
                >
                  EN
                </button>
              </div>
            </div>
          </div>
        )}
      </aside>

      {/* Right Main Content Canvas */}
      <main className="flex-1 min-w-0 h-screen overflow-auto p-4 sm:p-6 space-y-6">
        {categoryDetail ? (
          <>
            {/* Prominent Category Header Card + Simplified Rule Badges + Utility Toolbar */}
            <div className="border border-slate-200 dark:border-zinc-800 rounded-xl bg-white dark:bg-zinc-900 p-5 sm:p-6 shadow-sm space-y-4">
              <div className="flex flex-col xl:flex-row xl:items-center justify-between gap-4">
                <div className="space-y-2">
                  {/* Breadcrumb & Sport Metadata using reusable <SportBadge /> */}
                  <div className="flex flex-wrap items-center gap-2">
                    <SportBadge
                      sport={
                        categoryDetail.category.sport ||
                        categoryDetail.category.sport_slug
                      }
                    />
                    <span
                      className={`inline-flex items-center rounded-lg border px-2.5 py-1 text-xs font-semibold ${
                        isKnockoutCategory
                          ? 'border-purple-200 bg-purple-50 text-purple-700 dark:border-purple-800 dark:bg-purple-950/40 dark:text-purple-300'
                          : 'border-indigo-200 bg-indigo-50 text-indigo-700 dark:border-indigo-800 dark:bg-indigo-950/40 dark:text-indigo-300'
                      }`}
                    >
                      {isKnockoutCategory
                        ? t('nav.knockout_bracket')
                        : t('modals.format_groups')}
                    </span>
                    <span className="text-xs font-medium text-slate-400">
                      {categoryDetail.tournament_name}
                    </span>
                  </div>

                  {/* Prominent Title + Status Badge + Quick Edit Icon Button */}
                  <div className="flex flex-wrap items-center gap-3">
                    <h2 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
                      {categoryDetail.category.name}
                    </h2>

                    {hasDrawnStages ? (
                      <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-semibold text-emerald-800 dark:border-emerald-800/60 dark:bg-emerald-950/50 dark:text-emerald-300">
                        <CheckCircle2 className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-400" />
                        {t('status.in_progress')}
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-xs font-semibold text-amber-800 dark:border-amber-800/60 dark:bg-amber-950/50 dark:text-amber-300">
                        <Clock className="h-3.5 w-3.5 text-amber-600 dark:text-amber-400" />
                        {t('status.draft_awaiting_draw')}
                      </span>
                    )}

                    <button
                      type="button"
                      onClick={handleOpenEditCategory}
                      title={t('modals.category_edit_title')}
                      className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 dark:border-zinc-700 bg-slate-50 dark:bg-zinc-800 text-slate-600 dark:text-slate-300 hover:border-blue-500 hover:text-blue-600 transition-colors"
                    >
                      <Edit3 className="h-4 w-4" />
                    </button>
                  </div>

                  {/* Compact Rule Pill Badges + Click-Activated Info Trigger Popover */}
                  <div
                    ref={rulesPopoverRef}
                    className="relative flex flex-wrap items-center gap-2 pt-0.5"
                  >
                    {getSportRulePills(categoryDetail.category.sport_slug).map(
                      (pill) => (
                        <span
                          key={pill}
                          className="inline-flex items-center rounded-md border border-slate-200 dark:border-zinc-700 bg-slate-100/80 dark:bg-zinc-800 px-2 py-0.5 text-xs font-semibold text-slate-700 dark:text-slate-300"
                        >
                          {pill}
                        </span>
                      )
                    )}

                    <button
                      type="button"
                      onClick={() => setShowRulesPopover((prev) => !prev)}
                      aria-expanded={showRulesPopover}
                      className={`inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-0.5 text-xs font-semibold transition-colors ${
                        showRulesPopover
                          ? 'border-blue-500 bg-blue-50 text-blue-700 dark:border-blue-500 dark:bg-blue-950/60 dark:text-blue-300'
                          : 'border-slate-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-zinc-700'
                      }`}
                    >
                      <Info className="h-3.5 w-3.5" />
                      <span>{t('rules.trigger_label')}</span>
                    </button>

                    {showRulesPopover && (() => {
                      const ruleDetails = getSportRuleDetails(
                        categoryDetail.category.sport_slug
                      );
                      const isEN = currentLang === 'en';
                      return (
                        <div className="absolute left-0 top-full mt-2 z-40 w-[330px] sm:w-[420px] rounded-xl border border-slate-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 p-4 text-xs shadow-xl ring-1 ring-slate-900/5 space-y-3">
                          <div className="flex items-center justify-between border-b border-slate-100 dark:border-zinc-800 pb-2.5">
                            <div className="flex items-center gap-2">
                              <SportBadge
                                sport={categoryDetail.category.sport_slug}
                                size="sm"
                              />
                              <span className="font-bold text-slate-900 dark:text-slate-100">
                                {t('rules.trigger_label')}
                              </span>
                            </div>
                            <button
                              type="button"
                              onClick={() => setShowRulesPopover(false)}
                              className="rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-zinc-800 dark:hover:text-slate-200 transition-colors"
                            >
                              <X className="h-3.5 w-3.5" />
                            </button>
                          </div>

                          <dl className="space-y-2 text-slate-700 dark:text-slate-300">
                            <div className="flex items-start justify-between gap-3">
                              <dt className="font-semibold text-slate-500 dark:text-slate-400 shrink-0">
                                {isEN ? 'Win:' : 'Galibiyet:'}
                              </dt>
                              <dd className="font-bold text-emerald-600 dark:text-emerald-400 text-right">
                                {ruleDetails.win}
                              </dd>
                            </div>

                            <div className="flex items-start justify-between gap-3">
                              <dt className="font-semibold text-slate-500 dark:text-slate-400 shrink-0">
                                {isEN ? 'Draw:' : 'Beraberlik:'}
                              </dt>
                              <dd className="font-medium text-right">
                                {ruleDetails.draw}
                              </dd>
                            </div>

                            <div className="flex items-start justify-between gap-3">
                              <dt className="font-semibold text-slate-500 dark:text-slate-400 shrink-0">
                                {isEN ? 'Loss:' : 'Mağlubiyet:'}
                              </dt>
                              <dd className="font-medium text-right">
                                {ruleDetails.loss}
                              </dd>
                            </div>

                            <div className="flex items-start justify-between gap-3 pt-1.5 border-t border-slate-100 dark:border-zinc-800">
                              <dt className="font-semibold text-slate-500 dark:text-slate-400 shrink-0">
                                {isEN ? 'Forfeit Score:' : 'Hükmen Skoru:'}
                              </dt>
                              <dd className="font-bold text-amber-600 dark:text-amber-400 text-right">
                                {ruleDetails.forfeitScore}
                              </dd>
                            </div>

                            <div className="pt-1.5 border-t border-slate-100 dark:border-zinc-800 space-y-1">
                              <dt className="font-semibold text-slate-500 dark:text-slate-400">
                                {isEN
                                  ? 'Tiebreak Criteria:'
                                  : 'Averaj Sıralama Kriterleri:'}
                              </dt>
                              <dd className="rounded-lg bg-slate-50 dark:bg-zinc-800/80 px-2.5 py-1.5 font-medium text-slate-800 dark:text-slate-200 leading-relaxed">
                                {ruleDetails.tiebreak}
                              </dd>
                            </div>
                          </dl>
                        </div>
                      );
                    })()}
                  </div>
                </div>

                {/* Top Right Header Actions (Unified flex items-center gap-2.5 container with fixed h-10 baselines) */}
                <div className="flex flex-wrap items-center gap-2.5 shrink-0">
                  {/* 1. Primary Unified Draw & Fixture Button */}
                  {!isKnockoutCategory && (
                    <span title={drawTooltipText} className="inline-flex">
                      <button
                        type="button"
                        disabled={!canExecuteDraw}
                        title={drawTooltipText}
                        onClick={() => setShowSeedingModal(true)}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 text-sm font-medium text-white shadow-xs hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50 transition"
                      >
                        <Sparkles className="h-4 w-4" />
                        {t('fixtures.draw_btn')}
                      </button>
                    </span>
                  )}

                  {/* 2. Export Excel Button */}
                  <span title={exportTooltipText} className="inline-flex">
                    <button
                      type="button"
                      disabled={!hasDrawnStages}
                      title={exportTooltipText}
                      onClick={
                        isKnockoutCategory
                          ? handleExportBracketExcel
                          : handleExportExcel
                      }
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-emerald-600 px-4 text-sm font-medium text-white shadow-xs hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-50 transition"
                    >
                      <FileSpreadsheet className="h-4 w-4" />
                      {isKnockoutCategory
                        ? t('bracket.export_excel')
                        : t('app.export_excel')}
                    </button>
                  </span>

                  {/* 3. Danger-Styled "Fikstürü Sıfırla" Button */}
                  <span title={resetTooltipText} className="inline-flex">
                    <button
                      type="button"
                      disabled={!hasDrawnStages || resettingFixtures}
                      title={resetTooltipText}
                      onClick={handleTriggerResetFixtures}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-lg border border-rose-200 dark:border-rose-800/80 bg-rose-50/80 dark:bg-rose-950/40 px-4 text-sm font-medium text-rose-700 dark:text-rose-300 hover:bg-rose-100 dark:hover:bg-rose-900/60 disabled:cursor-not-allowed disabled:opacity-40 transition"
                    >
                      <RotateCcw className="h-4 w-4" />
                      {t('fixtures.reset_btn')}
                    </button>
                  </span>
                </div>
              </div>

              {/* Navigation Tabs Bar Aligned with Fixed h-9 Baselines */}
              <div className="flex flex-wrap items-center justify-between gap-2.5 border-t border-slate-100 dark:border-zinc-800 pt-4">
                {isKnockoutCategory ? (
                  /* Knockout Mode Tabs */
                  <>
                    <div className="flex flex-wrap items-center gap-2">
                      <button
                        onClick={() => setActiveTab('knockout_bracket')}
                        className={`inline-flex h-9 items-center justify-center gap-2 rounded-lg px-3.5 text-sm font-medium transition ${
                          activeTab !== 'teams'
                            ? 'bg-purple-600 text-white shadow-xs'
                            : 'bg-purple-50 dark:bg-purple-950/60 text-purple-800 dark:text-purple-300 hover:bg-purple-100'
                        }`}
                      >
                        <GitBranch className="h-4 w-4" />
                        {t('nav.knockout_bracket')}
                      </button>
                    </div>

                    <button
                      onClick={() => setActiveTab('teams')}
                      className={`inline-flex h-9 items-center justify-center gap-2 rounded-lg px-3.5 text-sm font-medium transition ${
                        activeTab === 'teams'
                          ? 'bg-slate-900 dark:bg-blue-600 text-white shadow-xs'
                          : 'border border-slate-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-zinc-700'
                      }`}
                    >
                      <Users className="h-4 w-4" />
                      {t('nav.participating_teams')} ({registeredTeamCount} Takım)
                    </button>
                  </>
                ) : (
                  /* Group Mode Multi-Stage Tabs with Aligned Right-Side Teams Button */
                  <>
                    <div className="flex flex-wrap items-center gap-2">
                      <button
                        onClick={() => {
                          setSelectedGroupFilter('all');
                          setActiveTab('stage_1');
                        }}
                        className={`inline-flex h-9 items-center justify-center gap-2 rounded-lg px-3.5 text-sm font-medium transition ${
                          activeTab === 'stage_1'
                            ? 'bg-slate-900 dark:bg-blue-600 text-white shadow-xs'
                            : 'bg-slate-100 dark:bg-zinc-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-zinc-700'
                        }`}
                      >
                        <LayoutGrid className="h-4 w-4" />
                        {stage1TabLabel}
                      </button>

                      <ChevronRight className="h-4 w-4 text-slate-300 dark:text-zinc-600 hidden sm:inline" />

                      <button
                        onClick={() => setActiveTab('pots_draw')}
                        className={`inline-flex h-9 items-center justify-center gap-2 rounded-lg px-3.5 text-sm font-medium transition ${
                          activeTab === 'pots_draw'
                            ? 'bg-indigo-600 text-white shadow-xs'
                            : 'bg-indigo-50 dark:bg-indigo-950/60 text-indigo-700 dark:text-indigo-300 hover:bg-indigo-100 dark:hover:bg-indigo-900/60'
                        }`}
                      >
                        <Layers className="h-4 w-4" />
                        <span>{t('nav.pots_draw')}</span>
                        {categoryDetail.pot_assignments?.length > 0 && (
                          <span className="rounded-full bg-indigo-900/20 dark:bg-indigo-400/20 px-2 py-0.5 text-xs font-bold">
                            {categoryDetail.pot_assignments.length} Takım
                          </span>
                        )}
                      </button>

                      <ChevronRight className="h-4 w-4 text-slate-300 dark:text-zinc-600 hidden sm:inline" />

                      <button
                        onClick={() => {
                          setSelectedGroupFilter('all');
                          setActiveTab('stage_2');
                        }}
                        className={`inline-flex h-9 items-center justify-center gap-2 rounded-lg px-3.5 text-sm font-medium transition ${
                          activeTab === 'stage_2'
                            ? 'bg-blue-600 text-white shadow-xs'
                            : 'bg-slate-100 dark:bg-zinc-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-zinc-700'
                        }`}
                      >
                        <LayoutGrid className="h-4 w-4" />
                        {stage2TabLabel}
                      </button>

                      <ChevronRight className="h-4 w-4 text-slate-300 dark:text-zinc-600 hidden sm:inline" />

                      <button
                        onClick={() => setActiveTab('final_four')}
                        className={`inline-flex h-9 items-center justify-center gap-2 rounded-lg px-3.5 text-sm font-medium transition ${
                          activeTab === 'final_four'
                            ? 'bg-amber-600 text-white shadow-xs'
                            : hasFinalFourStage
                            ? 'bg-amber-100 dark:bg-amber-950/60 text-amber-900 dark:text-amber-300 hover:bg-amber-200'
                            : 'bg-amber-50 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300 hover:bg-amber-100'
                        }`}
                      >
                        <Trophy className="h-4 w-4" />
                        {t('nav.final_four')}
                      </button>
                    </div>

                    {/* Teams Management Tab aligned on identical h-9 baseline */}
                    <button
                      onClick={() => setActiveTab('teams')}
                      className={`inline-flex h-9 items-center justify-center gap-2 rounded-lg px-3.5 text-sm font-medium transition ${
                        activeTab === 'teams'
                          ? 'bg-slate-900 dark:bg-blue-600 text-white shadow-xs'
                          : 'border border-slate-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-zinc-700'
                      }`}
                    >
                      <Users className="h-4 w-4" />
                      {t('nav.teams')} ({registeredTeamCount} Takım)
                    </button>
                  </>
                )}
              </div>
            </div>

            {/* Workspace Views */}
            {isKnockoutCategory ? (
              activeTab === 'teams' ? (
                <TeamsPanel
                  stages={categoryDetail.stages}
                  teams={categoryDetail.teams}
                  currentLang={currentLang}
                  isKnockoutMode={true}
                  onDeleteStage={handleDeleteStage}
                  onAddTeam={handleAddTeam}
                  onUpdateTeamStage={handleUpdateTeamStage}
                  onToggleTeamSeed={handleToggleTeamSeed}
                  onOpenSeedingModal={() => setShowSeedingModal(true)}
                  onBuildKnockout={() => setActiveTab('knockout_bracket')}
                  onImportExcel={handleImportTeamsExcel}
                  onDeleteTeam={handleDeleteTeam}
                />
              ) : (
                <BracketTree
                  stages={categoryDetail.stages}
                  matches={categoryDetail.matches}
                  teams={categoryDetail.teams}
                  sportSlug={categoryDetail.category.sport_slug}
                  venues={venues}
                  onSelectMatch={(m) => setEditingMatch(m)}
                  onBuildKnockout={handleBuildKnockout}
                  onSwapMatchPairings={handleSwapMatchPairings}
                  onManageTeams={() => setActiveTab('teams')}
                />
              )
            ) : (
              <>
                {activeTab === 'stage_1' &&
                  renderKademeGroupView(stage1Groups, false)}

                {activeTab === 'pots_draw' && (
                  <PotManagementPanel
                    stages={categoryDetail.stages}
                    teams={categoryDetail.teams}
                    potAssignments={categoryDetail.pot_assignments || []}
                    currentRound={categoryDetail.current_round || 1}
                    sportSlug={categoryDetail.category.sport_slug}
                    venues={venues}
                    onPreparePots={handlePreparePots}
                    onSavePots={handleSavePots}
                    onExecuteNextRoundDraw={handleExecuteNextRoundDraw}
                    onBuildFinalFour={handleBuildFinalFour}
                  />
                )}

                {activeTab === 'stage_2' &&
                  renderKademeGroupView(stage2Groups, true)}

                {activeTab === 'final_four' && (
                  <FinalFourBracket
                    stages={categoryDetail.stages}
                    matches={categoryDetail.matches}
                    teams={categoryDetail.teams}
                    sportSlug={categoryDetail.category.sport_slug}
                    venues={venues}
                    onSelectMatch={(m) => setEditingMatch(m)}
                    onBuildFinalFour={handleBuildFinalFour}
                    onSwapMatchPairings={handleSwapMatchPairings}
                  />
                )}

                {activeTab === 'knockout_bracket' && (
                  <BracketTree
                    stages={categoryDetail.stages}
                    matches={categoryDetail.matches}
                    teams={categoryDetail.teams}
                    sportSlug={categoryDetail.category.sport_slug}
                    venues={venues}
                    onSelectMatch={(m) => setEditingMatch(m)}
                    onBuildKnockout={handleBuildKnockout}
                    onSwapMatchPairings={handleSwapMatchPairings}
                    onManageTeams={() => setActiveTab('teams')}
                  />
                )}

                {activeTab === 'teams' && (
                  <TeamsPanel
                    stages={categoryDetail.stages}
                    teams={categoryDetail.teams}
                    currentLang={currentLang}
                    isKnockoutMode={false}
                    onDeleteStage={handleDeleteStage}
                    onAddTeam={handleAddTeam}
                    onUpdateTeamStage={handleUpdateTeamStage}
                    onToggleTeamSeed={handleToggleTeamSeed}
                    onOpenSeedingModal={() => setShowSeedingModal(true)}
                    onImportExcel={handleImportTeamsExcel}
                    onDeleteTeam={handleDeleteTeam}
                  />
                )}
              </>
            )}
          </>
        ) : (
          <div className="rounded-2xl border border-dashed border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 p-12 text-center space-y-4">
            <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400">
              <Trophy className="h-7 w-7" />
            </div>
            <div className="space-y-1">
              <h3 className="text-base font-bold text-slate-900 dark:text-slate-100">
                Kategori Seçilmedi
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Sol menüden bir kategori seçin veya aktif sezona yeni bir kategori ekleyin.
              </p>
            </div>
            {activeSeason && (
              <button
                type="button"
                onClick={() => setShowNewCategoryForTournament(activeSeason.id)}
                className="inline-flex items-center gap-2 rounded-xl bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 transition-colors"
              >
                <Plus className="h-4 w-4" />
                {t('nav.new_category')}
              </button>
            )}
          </div>
        )}
      </main>

      {/* Seeding & Stage 1 Draw Modal */}
      <SeedingModal
        open={showSeedingModal}
        teams={categoryDetail?.teams || []}
        sportSlug={categoryDetail?.category.sport_slug}
        venues={venues}
        onClose={() => setShowSeedingModal(false)}
        onRunInitialDraw={handleRunInitialDraw}
      />

      {/* Match Score, Penalty Shootout & Editable Pairing Modal */}
      <ScoreModal
        match={editingMatch}
        sportSlug={categoryDetail?.category.sport_slug || 'football'}
        teams={categoryDetail?.teams || []}
        venues={venues}
        onClose={() => setEditingMatch(null)}
        onSave={handleSaveMatchScore}
      />

      {/* Global Venue (Tesis) Management Modal */}
      <VenueManagerModal
        open={showVenueManagerModal}
        venues={venues}
        onClose={() => setShowVenueManagerModal(false)}
        onCreateVenue={handleCreateVenue}
        onUpdateVenue={handleUpdateVenue}
        onDeleteVenue={handleDeleteVenue}
      />

      {/* Fixture Reset Confirmation Modal */}
      {showResetConfirmModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-xs">
          <div className="w-full max-w-md overflow-hidden rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xl">
            <div className="flex items-center gap-2.5 border-b border-rose-200 dark:border-rose-900/60 bg-rose-600 px-5 py-4 text-white">
              <AlertTriangle className="h-5 w-5 shrink-0" />
              <h3 className="text-base font-extrabold">
                {t('fixtures.reset_confirm_title')}
              </h3>
            </div>
            <div className="p-5 space-y-4">
              <p className="text-sm font-medium text-slate-700 dark:text-slate-300 leading-relaxed">
                {t('fixtures.reset_confirm_body')}
              </p>
              <div className="flex justify-end gap-2 pt-2 border-t border-slate-100 dark:border-slate-800">
                <button
                  type="button"
                  disabled={resettingFixtures}
                  onClick={() => setShowResetConfirmModal(false)}
                  className="inline-flex items-center gap-2 rounded-xl border border-slate-300 dark:border-slate-700 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  {t('modals.cancel')}
                </button>
                <button
                  type="button"
                  disabled={resettingFixtures}
                  onClick={handleResetFixtures}
                  className="inline-flex items-center gap-2 rounded-xl bg-rose-600 px-4 py-2 text-sm font-medium text-white hover:bg-rose-700 disabled:opacity-50"
                >
                  <RotateCcw className="h-4 w-4" />
                  {t('fixtures.reset_confirm_action')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Season Management Modal */}
      {showSeasonManagerModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-xs">
          <div className="w-full max-w-lg overflow-hidden rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xl">
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 bg-slate-900 px-5 py-4 text-white">
              <div className="flex items-center gap-2">
                <Calendar className="h-4 w-4 text-blue-400" />
                <h3 className="text-base font-bold">{t('nav.tournaments')}</h3>
              </div>
              <button
                type="button"
                onClick={() => {
                  setShowSeasonManagerModal(false);
                  setShowNewTournament(true);
                }}
                className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-bold text-white hover:bg-blue-700"
              >
                <Plus className="h-3.5 w-3.5" />
                {t('nav.new_tournament')}
              </button>
            </div>
            <div className="p-5 space-y-3 max-h-96 overflow-y-auto">
              {tournaments.map((trn) => (
                <div
                  key={trn.id}
                  className="flex items-center justify-between gap-3 rounded-xl border border-slate-200 dark:border-slate-800 p-3.5"
                >
                  <div>
                    <div className="text-sm font-bold text-slate-900 dark:text-slate-100">
                      {trn.name}
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-400">
                      {trn.categories?.length || 0} Kategori
                      {trn.start_date ? ` • ${trn.start_date}` : ''}
                      {trn.end_date ? ` – ${trn.end_date}` : ''}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => {
                        handleSeasonSelectChange(trn.id);
                        setShowSeasonManagerModal(false);
                      }}
                      className="rounded-lg border border-slate-200 dark:border-slate-700 px-3 py-1.5 text-xs font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800"
                    >
                      Seç
                    </button>
                    <button
                      type="button"
                      onClick={() => handleDeleteTournament(trn.id)}
                      className="rounded-lg p-1.5 text-slate-400 hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-950/50"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
            <div className="flex justify-end border-t border-slate-200 dark:border-slate-800 px-5 py-3">
              <button
                type="button"
                onClick={() => setShowSeasonManagerModal(false)}
                className="rounded-xl border border-slate-300 dark:border-slate-700 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300"
              >
                Kapat
              </button>
            </div>
          </div>
        </div>
      )}

      {/* New Season Modal */}
      {showNewTournament && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-xs">
          <div className="w-full max-w-md overflow-hidden rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xl">
            <div className="border-b border-slate-200 dark:border-slate-800 bg-slate-900 px-5 py-4 text-white">
              <h3 className="text-base font-bold">{t('modals.tournament_title')}</h3>
            </div>
            <form onSubmit={handleCreateTournament} className="p-5 space-y-4">
              <div>
                <label className="block text-xs font-bold uppercase text-slate-600 dark:text-slate-400 mb-1">
                  {t('modals.tournament_name')}
                </label>
                <input
                  type="text"
                  required
                  placeholder="2026-2027 Okul Sporları Sezonu"
                  value={tName}
                  onChange={(e) => setTName(e.target.value)}
                  className="w-full rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-3 py-2 text-sm"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-bold uppercase text-slate-600 dark:text-slate-400 mb-1">
                    {t('modals.start_date')}
                  </label>
                  <input
                    type="text"
                    placeholder="2026"
                    value={tStart}
                    onChange={(e) => setTStart(e.target.value)}
                    className="w-full rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs font-bold uppercase text-slate-600 dark:text-slate-400 mb-1">
                    {t('modals.end_date')}
                  </label>
                  <input
                    type="text"
                    placeholder="2027"
                    value={tEnd}
                    onChange={(e) => setTEnd(e.target.value)}
                    className="w-full rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-3 py-2 text-sm"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-3 border-t border-slate-200 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setShowNewTournament(false)}
                  className="rounded-xl border border-slate-300 dark:border-slate-700 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  {t('modals.cancel')}
                </button>
                <button
                  type="submit"
                  className="rounded-xl bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
                >
                  {t('modals.create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* New Category Modal */}
      {showNewCategoryForTournament !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-xs">
          <div className="w-full max-w-md overflow-hidden rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xl">
            <div className="border-b border-slate-200 dark:border-slate-800 bg-slate-900 px-5 py-4 text-white">
              <h3 className="text-base font-bold">{t('modals.category_title')}</h3>
            </div>
            <form onSubmit={handleCreateCategory} className="p-5 space-y-4">
              <div>
                <label className="block text-xs font-bold uppercase text-slate-600 dark:text-slate-400 mb-1">
                  {t('modals.sport_branch')}
                </label>
                <select
                  value={cSport}
                  onChange={(e) => setCSport(e.target.value as SportSlug)}
                  className="w-full rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-3 py-2 text-sm font-semibold"
                >
                  <option value="football">{t('sports.football')}</option>
                  <option value="futsal">{t('sports.futsal')}</option>
                  <option value="basketball">{t('sports.basketball')}</option>
                  <option value="volleyball">{t('sports.volleyball')}</option>
                  <option value="handball">{t('sports.handball')}</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-bold uppercase text-slate-600 dark:text-slate-400 mb-1">
                  {t('modals.category_name')}
                </label>
                <input
                  type="text"
                  required
                  placeholder="Genç Erkek A / Yıldız Kızlar"
                  value={cName}
                  onChange={(e) => setCName(e.target.value)}
                  className="w-full rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="block text-xs font-bold uppercase text-slate-600 dark:text-slate-400 mb-1.5">
                  {t('modals.category_format')}
                </label>
                <div className="grid grid-cols-1 gap-2">
                  <button
                    type="button"
                    onClick={() => setCFormat('groups')}
                    className={`flex items-center gap-2.5 rounded-xl border px-3.5 py-2.5 text-left text-xs font-extrabold transition-all ${
                      cFormat === 'groups'
                        ? 'border-blue-600 bg-blue-50 text-blue-900 dark:border-blue-500 dark:bg-blue-950/60 dark:text-blue-200'
                        : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300'
                    }`}
                  >
                    <LayoutGrid className="h-4 w-4 shrink-0 text-blue-600 dark:text-blue-400" />
                    <span>{t('modals.format_groups')}</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setCFormat('knockout')}
                    className={`flex items-center gap-2.5 rounded-xl border px-3.5 py-2.5 text-left text-xs font-extrabold transition-all ${
                      cFormat === 'knockout'
                        ? 'border-purple-600 bg-purple-50 text-purple-900 dark:border-purple-500 dark:bg-purple-950/60 dark:text-purple-200'
                        : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300'
                    }`}
                  >
                    <GitBranch className="h-4 w-4 shrink-0 text-purple-600 dark:text-purple-400" />
                    <span>{t('modals.format_knockout')}</span>
                  </button>
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-3 border-t border-slate-200 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setShowNewCategoryForTournament(null)}
                  className="rounded-xl border border-slate-300 dark:border-slate-700 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  {t('modals.cancel')}
                </button>
                <button
                  type="submit"
                  className="rounded-xl bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
                >
                  {t('modals.create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Quick Edit Category Modal */}
      {showEditCategoryModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-xs">
          <div className="w-full max-w-md overflow-hidden rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xl">
            <div className="border-b border-slate-200 dark:border-slate-800 bg-slate-900 px-5 py-4 text-white">
              <h3 className="text-base font-bold">{t('modals.category_edit_title')}</h3>
            </div>
            <form onSubmit={handleUpdateCategory} className="p-5 space-y-4">
              <div>
                <label className="block text-xs font-bold uppercase text-slate-600 dark:text-slate-400 mb-1">
                  {t('modals.sport_branch')}
                </label>
                <select
                  value={editCatSport}
                  onChange={(e) => setEditCatSport(e.target.value as SportSlug)}
                  className="w-full rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-3 py-2 text-sm font-semibold"
                >
                  <option value="football">{t('sports.football')}</option>
                  <option value="futsal">{t('sports.futsal')}</option>
                  <option value="basketball">{t('sports.basketball')}</option>
                  <option value="volleyball">{t('sports.volleyball')}</option>
                  <option value="handball">{t('sports.handball')}</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-bold uppercase text-slate-600 dark:text-slate-400 mb-1">
                  {t('modals.category_name')}
                </label>
                <input
                  type="text"
                  required
                  value={editCatName}
                  onChange={(e) => setEditCatName(e.target.value)}
                  className="w-full rounded-xl border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 dark:text-slate-100 px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="block text-xs font-bold uppercase text-slate-600 dark:text-slate-400 mb-1.5">
                  {t('modals.category_format')}
                </label>
                <div className="grid grid-cols-1 gap-2">
                  <button
                    type="button"
                    onClick={() => setEditCatFormat('groups')}
                    className={`flex items-center gap-2.5 rounded-xl border px-3.5 py-2.5 text-left text-xs font-extrabold transition-all ${
                      editCatFormat === 'groups'
                        ? 'border-blue-600 bg-blue-50 text-blue-900 dark:border-blue-500 dark:bg-blue-950/60 dark:text-blue-200'
                        : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300'
                    }`}
                  >
                    <LayoutGrid className="h-4 w-4 shrink-0 text-blue-600 dark:text-blue-400" />
                    <span>{t('modals.format_groups')}</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setEditCatFormat('knockout')}
                    className={`flex items-center gap-2.5 rounded-xl border px-3.5 py-2.5 text-left text-xs font-extrabold transition-all ${
                      editCatFormat === 'knockout'
                        ? 'border-purple-600 bg-purple-50 text-purple-900 dark:border-purple-500 dark:bg-purple-950/60 dark:text-purple-200'
                        : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300'
                    }`}
                  >
                    <GitBranch className="h-4 w-4 shrink-0 text-purple-600 dark:text-purple-400" />
                    <span>{t('modals.format_knockout')}</span>
                  </button>
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-3 border-t border-slate-200 dark:border-slate-800">
                <button
                  type="button"
                  onClick={() => setShowEditCategoryModal(false)}
                  className="rounded-xl border border-slate-300 dark:border-slate-700 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  {t('modals.cancel')}
                </button>
                <button
                  type="submit"
                  className="rounded-xl bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
                >
                  {t('modals.save')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default App;

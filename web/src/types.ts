export type SportSlug = 'football' | 'futsal' | 'basketball' | 'volleyball' | 'handball';

export type MatchStatus = 'scheduled' | 'completed' | 'bye_advanced';

export type BracketSlot = '' | 'semi_final_1' | 'semi_final_2' | 'third_place' | 'final' | string;

export interface Category {
  id: number;
  tournament_id: number;
  sport_slug: SportSlug;
  sport?: SportSlug;
  name: string;
  format?: 'groups' | 'knockout' | string;
  status?: string;
  created_at: string;
}

export interface Tournament {
  id: number;
  name: string;
  description: string;
  location: string;
  start_date: string;
  end_date: string;
  created_at: string;
  categories: Category[];
}

export interface Team {
  id: number;
  category_id: number;
  stage_id: number | null;
  slot_index?: number;
  school_name: string;
  team_tag: string;
  display_name: string;
  is_seeded?: boolean;
}

export interface Match {
  id: number;
  category_id: number;
  stage_id: number;
  stage_name?: string;
  round: number;
  match_number: number;
  bracket_slot?: BracketSlot;
  source_home_stage_id?: number | null;
  source_home_rank?: number;
  source_away_stage_id?: number | null;
  source_away_rank?: number;
  home_team_id: number | null;
  away_team_id: number | null;
  home_team_name: string;
  away_team_name: string;
  is_bye: boolean;
  is_forfeit?: boolean;
  forfeit_winner_team_id?: number | null;
  home_score: number;
  away_score: number;
  home_penalties?: number | null;
  away_penalties?: number | null;
  next_match_id?: number | null;
  next_match_slot?: string;
  winner_team_id?: number | null;
  status: MatchStatus;
  venue: string;
  court: string;
  scheduled_at: string;
  notes: string;
  updated_at: string;
}

export interface PotAssignment {
  id: number;
  category_id: number;
  round_number: number;
  stage_id: number | null;
  stage_name?: string;
  team_id: number;
  team_name: string;
  school_name: string;
  team_tag: string;
  pot_number: 1 | 2;
}

export interface Standing {
  rank: number;
  stage_id: number;
  stage_name: string;
  team_id: number;
  team_name: string;
  school_name: string;
  team_tag: string;
  is_seeded?: boolean;
  played: number;
  won: number;
  drawn: number;
  lost: number;
  points: number;
  score_for: number;
  score_against: number;
  score_diff: number;
  sets_won: number;
  sets_lost: number;
  set_ratio: number;
}

export interface Stage {
  id: number;
  category_id: number;
  name: string;
  stage_type: 'group' | 'final_group' | 'final_four' | 'knockout' | string;
  round_number: number;
  parent_stage_id?: number | null;
  double_round_robin: boolean;
  status?: string;
  teams: Team[];
  matches: Match[];
}

export interface CategoryDetail {
  category: Category;
  tournament_name: string;
  stages: Stage[];
  teams: Team[];
  matches: Match[];
  standings: Record<number, Standing[]>;
  all_standings: Standing[];
  pot_assignments: PotAssignment[];
  current_round: number;
}

export interface ServerInfo {
  local_ip: string;
  port: string;
  network_url: string;
}

export interface Venue {
  id: number;
  name: string;
  address: string;
  supported_sports: string[];
}

export function sportSlugToCode(slug: SportSlug): string {
  switch (slug) {
    case 'basketball':
      return 'BASKETBOL';
    case 'volleyball':
      return 'VOLEYBOL';
    case 'football':
      return 'FUTBOL';
    case 'futsal':
      return 'FUTSAL';
    case 'handball':
      return 'HENTBOL';
    default:
      return String(slug).toUpperCase();
  }
}

export function sportCodeToSlug(code: string): SportSlug {
  const upper = (code || '').trim().toUpperCase();
  switch (upper) {
    case 'BASKETBOL':
    case 'BASKETBALL':
      return 'basketball';
    case 'VOLEYBOL':
    case 'VOLLEYBALL':
      return 'volleyball';
    case 'FUTBOL':
    case 'FOOTBALL':
      return 'football';
    case 'FUTSAL':
      return 'futsal';
    case 'HENTBOL':
    case 'HANDBALL':
      return 'handball';
    default:
      return 'football';
  }
}

export function venueSupportsSport(venue: Venue, sportSlug?: SportSlug): boolean {
  if (!sportSlug) return true;
  if (!venue.supported_sports || venue.supported_sports.length === 0) return true;
  const target = sportSlugToCode(sportSlug);
  return venue.supported_sports.some(
    (s) => sportSlugToCode(sportCodeToSlug(s)) === target
  );
}

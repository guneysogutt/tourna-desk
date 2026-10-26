import React, { useMemo } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  createColumnHelper,
  ColumnDef,
} from '@tanstack/react-table';
import { useTranslation } from 'react-i18next';
import { Trophy, Star, Inbox, Users } from 'lucide-react';
import { SportSlug, Standing } from '../types';

interface StandingsTableProps {
  stageName: string;
  sportSlug: SportSlug;
  standings: Standing[];
  showSeededBadge?: boolean;
  onManageTeams?: () => void;
}

const columnHelper = createColumnHelper<Standing>();

export const StandingsTable: React.FC<StandingsTableProps> = ({
  stageName,
  sportSlug,
  standings,
  showSeededBadge = true,
  onManageTeams,
}) => {
  const { t } = useTranslation();
  const isVolleyball = sportSlug === 'volleyball';
  const isBasketball = sportSlug === 'basketball';

  const columns = useMemo(() => {
    const baseCols: ColumnDef<Standing, any>[] = [
      columnHelper.accessor('rank', {
        header: () => t('standings.rank'),
        cell: (info) => {
          const rank = info.getValue();
          return (
            <div className="flex items-center justify-center">
              <span
                className={`inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold ${
                  rank === 1
                    ? 'bg-amber-100 text-amber-800 ring-1 ring-amber-300 dark:bg-amber-500/20 dark:text-amber-300 dark:ring-amber-500/40'
                    : rank === 2
                    ? 'bg-slate-200 text-slate-800 dark:bg-slate-700 dark:text-slate-200'
                    : rank === 3
                    ? 'bg-orange-100 text-orange-800 dark:bg-orange-500/20 dark:text-orange-300'
                    : 'text-slate-600 dark:text-slate-400'
                }`}
              >
                {rank}
              </span>
            </div>
          );
        },
      }),
      columnHelper.accessor('team_name', {
        header: () => t('standings.team'),
        cell: (info) => (
          <div className="flex items-center gap-2 text-left font-semibold text-slate-900 dark:text-slate-100">
            <span>{info.getValue()}</span>
            {showSeededBadge && info.row.original.is_seeded && (
              <span
                title={t('standings.seeded_badge')}
                className="inline-flex items-center gap-1 rounded-md bg-amber-100 px-2 py-0.5 text-[10px] font-extrabold text-amber-800 dark:bg-amber-500/20 dark:text-amber-300"
              >
                <Star className="h-2.5 w-2.5 fill-amber-500 text-amber-600 dark:text-amber-400" />
                {t('standings.seeded_badge')}
              </span>
            )}
          </div>
        ),
      }),
      columnHelper.accessor('played', {
        header: () => t('standings.played'),
        cell: (info) => (
          <span className="font-medium text-slate-700 dark:text-slate-300">
            {info.getValue()}
          </span>
        ),
      }),
      columnHelper.accessor('won', {
        header: () => t('standings.won'),
        cell: (info) => (
          <span className="font-semibold text-emerald-700 dark:text-emerald-400">
            {info.getValue()}
          </span>
        ),
      }),
    ];

    if (!isVolleyball && !isBasketball) {
      baseCols.push(
        columnHelper.accessor('drawn', {
          header: () => t('standings.drawn'),
          cell: (info) => (
            <span className="text-slate-600 dark:text-slate-400">{info.getValue()}</span>
          ),
        })
      );
    }

    baseCols.push(
      columnHelper.accessor('lost', {
        header: () => t('standings.lost'),
        cell: (info) => (
          <span className="text-rose-600 dark:text-rose-400">{info.getValue()}</span>
        ),
      })
    );

    if (isVolleyball) {
      baseCols.push(
        columnHelper.accessor('sets_won', {
          header: () => t('standings.sets_won'),
          cell: (info) => (
            <span className="text-slate-700 dark:text-slate-300">{info.getValue()}</span>
          ),
        }),
        columnHelper.accessor('sets_lost', {
          header: () => t('standings.sets_lost'),
          cell: (info) => (
            <span className="text-slate-700 dark:text-slate-300">{info.getValue()}</span>
          ),
        }),
        columnHelper.accessor('set_ratio', {
          header: () => t('standings.set_ratio'),
          cell: (info) => (
            <span className="font-mono text-xs text-slate-600 dark:text-slate-400">
              {info.getValue().toFixed(2)}
            </span>
          ),
        })
      );
    } else {
      baseCols.push(
        columnHelper.accessor('score_for', {
          header: () => t('standings.for'),
          cell: (info) => (
            <span className="text-slate-600 dark:text-slate-400">{info.getValue()}</span>
          ),
        }),
        columnHelper.accessor('score_against', {
          header: () => t('standings.against'),
          cell: (info) => (
            <span className="text-slate-600 dark:text-slate-400">{info.getValue()}</span>
          ),
        }),
        columnHelper.accessor('score_diff', {
          header: () => t('standings.diff'),
          cell: (info) => {
            const val = info.getValue();
            return (
              <span
                className={`font-medium ${
                  val > 0
                    ? 'text-emerald-600 dark:text-emerald-400'
                    : val < 0
                    ? 'text-rose-600 dark:text-rose-400'
                    : 'text-slate-500 dark:text-slate-400'
                }`}
              >
                {val > 0 ? `+${val}` : val}
              </span>
            );
          },
        })
      );
    }

    baseCols.push(
      columnHelper.accessor('points', {
        header: () => t('standings.points'),
        cell: (info) => (
          <span className="inline-flex min-w-[2rem] items-center justify-center rounded-md bg-blue-50 px-2 py-0.5 font-bold text-blue-700 ring-1 ring-inset ring-blue-600/20 dark:bg-blue-500/20 dark:text-blue-300 dark:ring-blue-400/30">
            {info.getValue()}
          </span>
        ),
      })
    );

    return baseCols;
  }, [t, isVolleyball, isBasketball, showSeededBadge]);

  const table = useReactTable({
    data: standings,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xs dark:border-slate-800 dark:bg-slate-900">
      <div className="flex items-center justify-between border-b border-slate-200 bg-slate-900 px-4 py-3 text-white dark:border-slate-800 dark:bg-slate-950">
        <div className="flex items-center gap-2">
          <Trophy className="h-4 w-4 text-amber-400" />
          <h3 className="text-sm font-bold tracking-wide">{stageName}</h3>
        </div>
        <span className="rounded-full bg-slate-800 px-2.5 py-0.5 text-xs font-medium text-slate-300">
          {standings.length} {t('teams.team_count')}
        </span>
      </div>

      {standings.length === 0 ? (
        <div className="flex flex-col items-center justify-center p-8 text-center space-y-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400">
            <Inbox className="h-6 w-6" />
          </div>
          <div className="space-y-1 max-w-sm">
            <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100">
              {t('standings.empty')}
            </h4>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {t('standings.empty_hint')}
            </p>
          </div>
          {onManageTeams && (
            <button
              type="button"
              onClick={onManageTeams}
              className="inline-flex items-center justify-center gap-2 rounded-xl bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-xs hover:bg-blue-700 transition-colors"
            >
              <Users className="h-4 w-4" />
              {t('standings.manage_teams_cta')}
            </button>
          )}
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              {table.getHeaderGroups().map((headerGroup) => (
                <tr
                  key={headerGroup.id}
                  className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wider text-slate-600 dark:border-slate-800 dark:bg-slate-800/60 dark:text-slate-300"
                >
                  {headerGroup.headers.map((header, idx) => (
                    <th
                      key={header.id}
                      className={`px-3 py-2.5 ${
                        idx === 1 ? 'text-left' : 'text-center'
                      }`}
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
              {table.getRowModel().rows.map((row, rIdx) => (
                <tr
                  key={row.id}
                  className={`transition-colors hover:bg-blue-50/40 dark:hover:bg-slate-800/70 ${
                    rIdx < 2
                      ? 'bg-emerald-50/25 dark:bg-emerald-950/20'
                      : rIdx % 2 === 1
                      ? 'bg-slate-50/50 dark:bg-slate-900/60'
                      : 'bg-white dark:bg-slate-900'
                  }`}
                >
                  {row.getVisibleCells().map((cell, cIdx) => (
                    <td
                      key={cell.id}
                      className={`px-3 py-2.5 ${
                        cIdx === 1 ? 'text-left' : 'text-center'
                      }`}
                    >
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

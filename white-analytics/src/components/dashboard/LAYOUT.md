# Layout rules

One spec for every page in the app shell, the portals and the public pages. The shell and the
shared components already follow it. A page aligns by using these components and adding no
spacing of its own.

## Surfaces

- The main panel is plain white (`bg-background`, `#ffffff`). Dark mode: deep navy `#151026`.
- Cards are solid `bg-card` with the hairline and soft shadow from `shadow-(--card-shadow)`.
  The hairline is a 1px shadow ring, so it never changes layout. Do not add `border` to a
  `Card`, and never use `backdrop-blur`, `bg-glass` or translucent `bg-card/NN` inside the panel.
- Popovers, menus, dialogs and sheets: `bg-popover` with `shadow-(--popover-shadow)` (built in).
- Gradients only on banners: `surface-banner` (or `PageHeader banner`) and `stage-tint`.
- The blush canvas appears only as the frame around the floating sidebar and panel on md+.

## Tokens (globals.css)

| Token | Phone | md (768+) | lg (1024+) | Use |
| --- | --- | --- | --- | --- |
| `--page-max` | 1440px | 1440px | 1440px | content max width |
| `--page-x` | 16px | 24px | 32px | page side padding |
| `--gap` | 16px | 24px | 24px | between page blocks, between cards in any grid |
| `--gap-inner` | 12px | 12px | 12px | section title to content, toolbar items, tabs to their content |
| `--card-pad` | 16px | 20px | 20px | card, KPI tile, dialog and sheet padding, first/last table cell |
| `--control-h` | 40px (44px on touch) | same | same | buttons, inputs, selects, tab lists in toolbars |

Use them through Tailwind: `gap-(--gap)`, `p-(--card-pad)`, `h-(--control-h)`.

## Page container

`src/app/(app)/layout.tsx` wraps every page in `page-container page-stack`: max width 1440,
side padding `--page-x`, and `--gap` between direct children. The top bar uses the same
container, so the sidebar trigger, page title, filters, tabs, cards and tables share one left
and one right edge.

Pages must NOT add:
- an outer wrapper with `space-y-*`, `gap-*`, `px-*`, `mx-auto`, `max-w-*` or `container`;
- negative margins (`-mx-*`, `-mt-*`, `-mb-3` under back links);
- `mt-*` / `mb-*` between top-level blocks.

Return a fragment (`<>...</>`) whose children are the blocks. When a page needs a narrower
reading width (a form, a settings page), put `max-w-3xl` on the block itself, left-aligned
(never `mx-auto`), so its left edge still matches the header.

Portals, share, careers, forms and chat layouts use the same utilities:
`page-container page-stack` with their own narrower `max-w-*` on the container if needed.

## Vertical rhythm

| Between | Spacing |
| --- | --- |
| Page header, filter bar, tabs, sections (page blocks) | `--gap` (automatic) |
| Cards in a grid, KPI tiles | `--gap` (`CardGrid`, `KpiGrid`) |
| Section title row and its content | `--gap-inner` (automatic in `Section`) |
| Items inside a toolbar | `--gap-inner` (`Toolbar`) |
| Form fields | 16px (`FormGrid`) |

## Components to use

From `@/components/dashboard/page-header`:
- `PageHeader` with `eyebrow`, `title`, `description`, `actions`, and `back={{ href, label }}`
  for detail pages. Do not render your own back link above the header.
- `Section` with `title`, `description` (info hint), `actions`. Its header row is 32px tall, so
  sections with and without actions line up.

From `@/components/dashboard/layout`:
- `Toolbar` (+ `ToolbarActions`): filter bars and table toolbars. Search first, filters next,
  `ToolbarActions` pushes buttons to the right. Pass `label` for an accessible name.
- `CardGrid cols={1|2|3|4}`: grids of cards with the page gap.
- `FormGrid cols={1|2}` and `FormActions`: label above input, two-column grids align label tops,
  footer right-aligned with the secondary button first and the primary last.
- `DetailLayout aside={...}`: main column plus a 340px aside from lg up.
- `KeyValueList items={[{ label, value }]}`: label column plus value column.

Existing shared pieces (already aligned):
- `Card`, `CardHeader`, `CardTitle`, `CardAction`, `CardContent`, `CardFooter`: padding is
  `--card-pad`. Do not override with `p-6`, `px-6`, `py-5`, `pt-5` and similar.
  `Card size="sm"` is the compact variant (12px phone, 16px md+).
- `ChartCard`: header row is at least 32px, card padding from the token.
- `KpiTile` / `KpiGrid`: tiles stretch to equal height; label and value sit on top, delta and
  caption are pinned to the bottom, so values and deltas line up across a row. Use `cols` on `KpiGrid` instead of your own grid.
- `StatStrip`, `DataTable` (+ ERP `ErpDataTable`, `SearchFilter`, `SelectFilter`,
  `SearchSelectFilter`, `ResetFilters`, `UrlTabs`, `DashboardFilterBar`, `FormField`).
- `EmptyState` / `ErpEmpty` / `ErpError`: fixed box heights (`min-h-64`, compact `min-h-40`).

## Controls

- Toolbars and forms use the default size (`--control-h`): `Button`, `Input`, `SelectTrigger`,
  `SearchableSelect`, `TabsList`. Do not pass `size="sm"` or `h-8`/`h-9` in a filter bar or
  form.
- `size="sm"` (32px) is for actions inside a card header, table rows and pagers only.
- Icons: 16px inline (`size-4`, default in buttons), 14px in `sm` buttons, 18-20px in nav.

## Tabs

- One style: `Tabs` / `TabsList` / `TabsTrigger` (or `UrlTabs`). The list starts on the content
  edge; no wrapper padding or margin around it. Do not restyle triggers.

## Cards and headers

- Card header text, chart card titles, KPI labels, table first columns and stat-strip labels all
  sit `--card-pad` from the card edge.
- Put card actions in `CardAction` (or `ChartCard actions`), sized `sm`.

## Tables

- A standalone `DataTable` is its own card (hairline frame, first column at `--card-pad`).
- A `DataTable` placed inside a `Card` drops its frame automatically and bleeds to the card
  edges, so its first column lines up with the card title. Its toolbar stays inside the card.
- In a ChartCard table view use `<DataTable bare dense />` (no toolbar, same bleed).
- A raw `<Table>` inside `CardContent`: wrap it in `<div className="card-bleed">` (or use
  `CardContent className="px-0"`). The first and last cells already carry `--card-pad`.
- Numeric columns: `meta: { align: "right" }` (right-aligned, tabular figures).
- Empty and loading states use the same box height as `EmptyState`.

## Dialogs and sheets

- Padding equals card padding (built in). Footer: `DialogFooter` / `FormActions`, secondary
  then primary. Fields inside use `FormGrid` / `FormField`.

## Checking a page

At 1440x900, 1280x800, 768x1024 and 375x812, light and dark, compare
`getBoundingClientRect().left` of the page title, filter bar, first card and table: they must
match (and `.right` of the last item in each row). No horizontal scroll at 375.

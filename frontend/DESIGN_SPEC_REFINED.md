# NavXchange Design System v2
**Zerodha Precision × Groww Delight × Game Economy**

> Based on the supplied engineering specification (market structure, NAV5 index, trading workflow, real-time architecture, quests, leaderboard, and WebSocket model), this guide assumes a professional trading dashboard where gameplay enhances—not distracts from—the trading experience.

---

# 1. Executive Summary

## Design Philosophy

The UI should feel like:

> **95% Trading Platform**  
> **5% Rewarding Game**

The interface should **never** resemble Duolingo, Coin Master, or a mobile idle game.

Instead, combine:

- **Zerodha** → precision, minimalism, trust
- **Bloomberg Terminal** → dense information hierarchy
- **Groww** → smooth interactions and delightful polish
- **Steam Achievements** → progression and rewards

The trading experience must always feel trustworthy.

The "game" only appears when:

- Quest progress updates
- Achievements unlock
- Rank changes
- Login rewards
- Streaks increase
- Milestones are reached

Never during order placement.

---

# 2. Refined Color Palette

## Backgrounds

| Purpose | Color |
|----------|---------|
| App Background | `#0B1220` |
| Secondary Surface | `#131C2E` |
| Elevated Surface | `#1A2438` |
| Modal Background | `#1E293B` |
| Hover Surface | `#24324A` |

These darker blues provide more depth than flat navy while preserving a professional appearance.

---

## Brand Colors

### Primary Blue

```
#2563EB
```

Hover

```
#1D4ED8
```

Pressed

```
#1E40AF
```

Use only for:

- Logo
- Active navigation
- Primary CTA
- Links
- Focus rings

---

## Success

```
#22C55E
```

Hover

```
#16A34A
```

Background tint

```css
rgba(34,197,94,.12)
```

Use for:

- Buy actions
- Positive P/L
- Price increase
- Success state

---

## Danger

```
#EF4444
```

Hover

```
#DC2626
```

Background tint

```css
rgba(239,68,68,.12)
```

Use for:

- Sell actions
- Losses
- Error state
- Negative movement

---

## Warning

```
#F59E0B
```

Use for:

- Pending orders
- Circuit breakers
- High volatility
- Quote expiration

---

## Achievement Gold

```
Primary
#FBBF24

Dark
#D97706

Glow
rgba(251,191,36,.25)
```

Reserved only for:

- Rank #1
- Milestones
- Legendary achievements
- Long streaks

---

## Information

```
#38BDF8
```

Use for:

- Live updates
- Notifications
- News
- WebSocket status

---

## Neutral Colors

| Purpose | Color |
|----------|---------|
| Primary Text | `#F8FAFC` |
| Secondary Text | `#CBD5E1` |
| Muted Text | `#94A3B8` |
| Disabled | `#64748B` |

---

## Card Type Indicators

### System Company

Left Accent

```
#2563EB
```

---

### NAV5 Index

Left Accent

```
#FBBF24
```

Subtle background

```
#1A2438
→
#162033
```

---

### User Created

Accent

```
#8B5CF6
```

No gradients.

---

## Opacity Guidelines

| State | Value |
|--------|-------|
| Hover | +6% brightness |
| Pressed | +10% |
| Selected | +14% |
| Disabled | 40% opacity |
| Loading | 65% opacity |

---

# 3. Typography

## Primary Font

```
Inter
```

---

## Monospace Font

Recommended:

```
IBM Plex Mono
```

Reasons:

- Better financial appearance
- Excellent digit alignment
- Cleaner than JetBrains Mono
- Highly readable

---

## Type Scale

### Hero Price

```
IBM Plex Mono
36px
600
Line-height: 40px
```

---

### Current Position

```
24px
600
```

---

### Card Price

```
18px
500
```

---

### Table Price

```
14px
500
```

---

### Daily %

```
13px
500
```

Always visually secondary.

---

## Headings

### Page Title

```
24px
600
```

---

### Section Title

```
18px
600
```

---

### Card Title

```
16px
600
```

---

### Body

```
14px
400
Line-height: 22px
```

---

### Caption

```
12px
400
Line-height: 18px
```

---

## ALL CAPS Usage

Allowed only for:

- BUY
- SELL
- LIVE
- NAV5
- P/L
- ROI

Never use ALL CAPS for headings.

---

# 4. Component Design

## Cards

```css
background: #131C2E;
border: 1px solid #2C3B55;
border-radius: 10px;
padding: 20px;
```

Hover

```css
border-color:#3B82F6;
transform:translateY(-2px);
transition:160ms;
```

Active

```css
outline:2px solid #3B82F6;
```

Shadow

Default

```
None
```

Hover

```css
0 8px 24px rgba(0,0,0,.18)
```

---

## Buttons

### Primary

Blue Filled

Height

```
48px
```

Padding

```
0 20px
```

---

### Buy

Filled Green

---

### Sell

Outlined Red

Avoid filled red buttons to reduce accidental selling.

---

### Secondary

Gray Outline

---

### Danger

Filled Red

Only for destructive actions.

---

### Loading

Replace label

```
Executing...
```

Display spinner.

Reduce opacity to 70%.

---

## Inputs

Height

```
48px
```

Radius

```
8px
```

Border

```
#334155
```

Focus

```
2px Blue Ring
```

---

## Order Panel

Quote preview inside highlighted container.

Background

```
#162033
```

Left border

```
Green
```

Show:

- Current Price
- Estimated Cost
- Fees
- Slippage
- Quote expires in 8 seconds

---

## Tables

Header

- Sticky
- Uppercase
- 12px
- Muted text

Rows

```
48px
```

Hover

```
#1A2438
```

Sorting

Chevron icon.

Inactive opacity

```
40%
```

---

## Progress Bars

Background

```
#2D3748
```

Fill

```
#22C55E
```

Animation

```
300ms ease-out
```

---

## Badges

Bronze

```
#B45309
```

Silver

```
#94A3B8
```

Gold

```
#FBBF24
```

Diamond

```
#60A5FA
```

---

# 5. Layout & Spacing

## Base Grid

```
8px
```

---

## Spacing Scale

| Token | Size |
|---------|------|
| xs | 8px |
| sm | 16px |
| md | 24px |
| lg | 32px |
| xl | 48px |

---

## Header

```
68px
```

---

## Sidebar

Desktop

```
296px
```

---

## Content Width

```
1600px max
```

Centered.

---

## Dashboard Grid

Desktop

```
4 Columns
Gap:20px
```

Laptop

```
3 Columns
```

Tablet

```
2 Columns
```

Mobile

```
1 Column
```

---

# 6. Page-by-Page Refinements

## Dashboard

### NAV5

Should dominate the page.

Design

- Double width card
- Gold border
- Sparkline
- Sector composition
- Live percentage

Place at the top.

---

### Company Cards

Each card shows

- Logo
- Symbol
- Price
- Daily %
- Mini Sparkline
- Volume

No descriptions.

---

### Price Movement

Instead of only color

Use

```
▲

▼
```

Improves accessibility.

---

### Sidebar

Desktop

Visible

Tablet

Collapsible

Mobile

Bottom drawer.

---

### News

Top-right section.

Newest item receives

```
NEW
```

Blue badge.

Expandable.

---

## Card Detail

Layout

```
Chart
65%

Order Panel
35%
```

Chart height

```
520px
```

---

### Your Position

Separate elevated card.

Display

- Shares
- Average Cost
- Current Value
- Profit/Loss

Never mix with market statistics.

---

### Executing Trade

Replace Buy button with

```
Executing...

Progress Bar

Estimated completion
```

Disable all inputs.

---

### Related News

Timeline below chart.

Each entry contains

- Headline
- Bullish/Bearish
- Sector
- Timestamp

---

## Leaderboard

Top 3

Left accent strip

- Gold
- Silver
- Bronze

Avoid oversized crowns.

---

### Current User

Blue border

Light blue background

Pinned if outside viewport.

---

### Rank Change

Animation

```
↑

↓

Slide
200ms
```

No bounce animation.

---

### Achievement Chips

Examples

- Top 1%
- Momentum Master
- Diamond Hands
- Sector Specialist
- Early Investor
- Market Maker

---

## Quests

Three Sections

- Active
- Completed
- Expired

---

### Progress

Animated bar.

Show

```
60%

3 / 5
```

---

### Completion

Animation

- Scale 250ms
- Gold pulse
- Checkmark
- Currency flies to balance

No confetti.

---

### Time Remaining

Top-right

```
Resets in

05:12:31
```

---

# 7. Interaction Guide

## Price Updates

Never animate the number itself.

Instead

```
Green flash
250ms
```

or

```
Red flash
250ms
```

Return to normal.

---

## News

Toast

Top-right

Display

- Headline
- Sector
- View button

---

## Leaderboard

Animate only rows that changed.

---

## Loading States

Cards

Skeleton

Tables

Skeleton

Buttons

Spinner

Charts

Gray Placeholder

---

## Connection Status

Header indicator

```
🟢 LIVE

⚪ CONNECTING

🔴 DISCONNECTED
```

---

# 8. Visual Flourishes

Animate only

- Quest completion
- Achievement unlock
- Login reward
- Rank increase
- Card creation

---

Never animate

- Prices
- Balance every tick
- Entire table refresh
- Dashboard refresh

---

## Micro-interactions

Hover

```
160ms
```

Modal

```
220ms
```

Card Lift

```
2px
```

Button

```
Brightness +6%
```

---

## Easter Egg

If player reaches Top 10

The logo gains a subtle gold shimmer for 24 hours.

Only once.

---

# 9. Tailwind + Shadcn Strategy

Customize

- Button
- Card
- Badge
- Dialog
- Table
- Progress
- Tabs
- Tooltip
- Toast
- ScrollArea
- HoverCard
- Skeleton
- Command

---

## Tailwind

```js
darkMode: "class"
```

---

## Plugins

```
tailwindcss-animate

@tailwindcss/forms

tailwindcss-radix
```

---

## Transition Defaults

```css
transition-all;
duration-150;
ease-out;
```

Maximum duration

```
300ms
```

---

## Charts

Recommended

- TradingView Lightweight Charts
- Recharts

Avoid Chart.js.

---

# 10. Design System Variables

```css
:root {

--bg:#0B1220;
--surface:#131C2E;
--surface-elevated:#1A2438;

--primary:#2563EB;
--primary-hover:#1D4ED8;

--success:#22C55E;
--danger:#EF4444;
--warning:#F59E0B;
--info:#38BDF8;

--gold:#FBBF24;
--silver:#94A3B8;
--bronze:#B45309;

--text:#F8FAFC;
--text-muted:#94A3B8;
--text-disabled:#64748B;

--border:#2C3B55;

--radius-card:10px;
--radius-button:8px;
--radius-input:8px;
--radius-modal:14px;

--shadow-hover:
0 8px 24px rgba(0,0,0,.18);

--transition-fast:150ms;
--transition-normal:220ms;
--transition-slow:300ms;

--header-height:68px;
--sidebar-width:296px;

--grid-gap:20px;
--section-gap:32px;

--font-sans:"Inter",sans-serif;
--font-mono:"IBM Plex Mono",monospace;

}
```

---

# Final Design Principle

Every screen should first answer:

> **"Can I make a better trading decision?"**

Only after that should it reward the player.

The trading interface must remain dense, precise, and trustworthy. The game layer should appear only through progression systems—quests, ranks, achievements, streaks, and rewards—ensuring NavXchange feels like a serious trading platform that happens to be fun rather than a game pretending to be a trading platform.
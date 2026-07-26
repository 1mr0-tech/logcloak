# logcloak.github.io source

Source for the logcloak marketing/docs site (Astro), deployed to the `gh-pages` branch
alongside the Helm chart artifacts. Built with Astro islands: Vue components from
[Inspira UI](https://inspira-ui.com) and React components from [Animate UI](https://animate-ui.com),
plus [Lenis](https://lenis.darkroom.engineering) for smooth scrolling.

## Structure

```
src/
├── pages/            index.astro, guide.astro, docs.astro, tool/index.astro
├── layouts/          Layout.astro — shared shell, nav, footer, Lenis
├── components/
│   ├── site/         page-specific components (Nav, Footer, Terminal, Code, PatternTool)
│   ├── inspira-ui/   Vue components pulled from the Inspira UI registry
│   ├── animate-ui/   React components pulled from the Animate UI registry
│   └── ui/           shadcn base primitives
public/
├── logo.png          full lockup (icon + wordmark), used for OG image
└── logo-icon.png      icon only, used for nav + favicon
```

## Commands

| Command | Action |
|---|---|
| `npm install` | Install dependencies |
| `npm run dev` | Dev server at `localhost:4321/logcloak/` (note the `base` path) |
| `npm run build` | Build to `./dist/` |
| `npm run preview` | Preview the production build locally |

## Deployment

Pushes to `main` that touch `website/**` trigger `.github/workflows/deploy-website.yaml`,
which builds the site and publishes only the site files to `gh-pages` — it never touches
`index.yaml` or the `logcloak-*.tgz` chart archives already there.

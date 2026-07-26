export function href(path: string): string {
  const base = import.meta.env.BASE_URL.replace(/\/$/, '');
  const clean = path.replace(/^\//, '');
  return clean ? `${base}/${clean}` : `${base}/`;
}

export const NAV_LINKS = [
  { label: 'Home', path: '/' },
  { label: 'Guide', path: '/guide' },
  { label: 'Docs', path: '/docs' },
] as const;

export const GITHUB_URL = 'https://github.com/1mr0-tech/logcloak';

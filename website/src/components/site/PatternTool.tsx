import * as React from 'react';
import { CopyButton } from '@/components/animate-ui/components/buttons/copy';

interface Builtin {
  name: string;
  desc: string;
  pattern: string;
}

const BUILTINS: Builtin[] = [
  { name: 'email', desc: 'RFC 5321 email addresses', pattern: '[a-zA-Z0-9._%+\\-]+@[a-zA-Z0-9.\\-]+\\.[a-zA-Z]{2,}' },
  { name: 'phone-e164', desc: 'E.164 international (+12025550104)', pattern: '\\+[1-9]\\d{6,14}' },
  { name: 'phone-us', desc: 'US phone formats', pattern: '(\\+1[\\-\\s]?)?\\(?\\d{3}\\)?[\\-\\s]?\\d{3}[\\-\\s]?\\d{4}' },
  { name: 'otp-6digit', desc: 'Standalone 6-digit OTP codes', pattern: '\\b[0-9]{6}\\b' },
  { name: 'credit-card', desc: '13–19 digit card numbers', pattern: '\\b(?:\\d[ \\-]?){13,19}\\b' },
  { name: 'jwt', desc: 'eyJ… JWT tokens', pattern: 'eyJ[a-zA-Z0-9\\-_]+\\.eyJ[a-zA-Z0-9\\-_]+\\.[a-zA-Z0-9\\-_]+' },
  { name: 'ipv4-private', desc: 'RFC 1918 private IPs', pattern: '(10\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}|172\\.(1[6-9]|2\\d|3[01])\\.\\d{1,3}\\.\\d{1,3}|192\\.168\\.\\d{1,3}\\.\\d{1,3})' },
  { name: 'uuid', desc: 'UUID v4', pattern: '[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}' },
  { name: 'iban', desc: 'International Bank Account Numbers', pattern: '\\b[A-Z]{2}[0-9]{2}[A-Z0-9]{11,30}\\b' },
  { name: 'ssn', desc: 'US Social Security Numbers', pattern: '\\b[0-9]{3}[- ][0-9]{2}[- ][0-9]{4}\\b' },
];

interface Custom {
  id: number;
  tab: 'write' | 'suggest';
  name: string;
  regex: string;
  sample: string;
}

function re2check(pattern: string): string {
  if (!pattern) return '';
  for (const u of ['(?=', '(?!', '(?<=', '(?<!', '(?P<']) {
    if (pattern.includes(u)) return `lookahead/lookbehind "${u}" not allowed`;
  }
  if (/\\[1-9]/.test(pattern)) return 'backreferences not allowed';
  try {
    new RegExp(pattern);
    return 'ok';
  } catch (e) {
    return (e as Error).message;
  }
}

function suggestRegex(sample: string): string {
  const s = sample.trim();
  if (!s) return '';
  const prefixNum = s.match(/^([A-Z0-9]+[_-])(\d+)$/i);
  if (prefixNum) return prefixNum[1].replace(/-/, '\\-') + '\\d+';
  const prefixAlnum = s.match(/^([A-Z0-9]+[_-])([a-zA-Z0-9]+)$/i);
  if (prefixAlnum) return prefixAlnum[1].slice(0, -1) + (prefixAlnum[1].slice(-1) === '-' ? '\\-' : '_') + '[a-zA-Z0-9]+';
  if (/^\d+$/.test(s)) return `\\d{${s.length}}`;
  if (/^[a-zA-Z0-9]+$/.test(s)) return `[a-zA-Z0-9]{${s.length}}`;
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&').replace(/[a-zA-Z0-9]+/g, '[a-zA-Z0-9]+');
}

type ActivePattern = { name: string; pattern: string; builtin?: string };

function previewMasked(log: string, active: ActivePattern[]): { text: string; masked: boolean }[] {
  if (!log) return [];
  let segs: { text: string; masked: boolean }[] = [{ text: log, masked: false }];
  for (const p of active) {
    try {
      const re = new RegExp(p.pattern, 'g');
      const next: typeof segs = [];
      for (const s of segs) {
        if (s.masked) {
          next.push(s);
          continue;
        }
        let last = 0;
        let m: RegExpExecArray | null;
        re.lastIndex = 0;
        while ((m = re.exec(s.text)) !== null) {
          if (m.index > last) next.push({ text: s.text.slice(last, m.index), masked: false });
          next.push({ text: m[0], masked: true });
          last = m.index + m[0].length;
          if (m[0].length === 0) break;
        }
        if (last < s.text.length) next.push({ text: s.text.slice(last), masked: false });
      }
      segs = next;
    } catch {
      // invalid regex mid-typing — skip
    }
  }
  return segs;
}

function CodeBlock({ text }: { text: string }) {
  return (
    <div className="group relative overflow-hidden rounded-lg border border-border bg-[#0d0d0f]">
      <div className="absolute right-2 top-2 opacity-0 transition-opacity group-hover:opacity-100">
        <CopyButton content={text} variant="ghost" size="xs" />
      </div>
      <pre className="overflow-x-auto p-4 font-mono text-xs leading-relaxed text-foreground/90">{text}</pre>
    </div>
  );
}

export default function PatternTool() {
  const [builtinChecked, setBuiltinChecked] = React.useState<Record<string, boolean>>({});
  const [customs, setCustoms] = React.useState<Custom[]>([]);
  const [nextId, setNextId] = React.useState(1);
  const [target, setTarget] = React.useState<'policy' | 'annotation'>('policy');
  const [namespace, setNamespace] = React.useState('');
  const [sampleLog, setSampleLog] = React.useState('');

  const ns = namespace.trim() || '<namespace>';

  const activePatterns: ActivePattern[] = React.useMemo(() => {
    const result: ActivePattern[] = [];
    for (const b of BUILTINS) {
      if (builtinChecked[b.name]) result.push({ name: b.name, pattern: b.pattern, builtin: b.name });
    }
    for (const c of customs) {
      if (c.name && c.regex && re2check(c.regex) === 'ok') {
        result.push({ name: c.name, pattern: c.regex });
      }
    }
    return result;
  }, [builtinChecked, customs]);

  const segments = React.useMemo(() => previewMasked(sampleLog, activePatterns), [sampleLog, activePatterns]);

  const regexOutput = activePatterns.length
    ? activePatterns.map((p) => `# ${p.name}\n${p.pattern}`).join('\n\n')
    : '# select patterns on the left';

  const yamlOutput = React.useMemo(() => {
    if (!activePatterns.length) return '# select patterns on the left';
    if (target === 'policy') {
      const lines = activePatterns
        .map((p) =>
          p.builtin
            ? `    - name: ${p.name}\n      builtin: ${p.builtin}`
            : `    - name: ${p.name}\n      regex: "${p.pattern.replace(/"/g, '\\"')}"`,
        )
        .join('\n');
      return `apiVersion: logcloak.io/v1alpha1\nkind: MaskingPolicy\nmetadata:\n  name: pii-baseline\n  namespace: ${ns}\nspec:\n  patterns:\n${lines}\n  redactWith: "[REDACTED]"`;
    }
    const builtinNames = activePatterns.filter((p) => p.builtin).map((p) => p.name);
    const customPats = activePatterns.filter((p) => !p.builtin);
    const lines: string[] = [];
    if (builtinNames.length) lines.push(`    logcloak.io/patterns: "${builtinNames.join(',')}"`);
    for (const p of customPats) lines.push(`    logcloak.io/regex-${p.name}: "${p.pattern.replace(/"/g, '\\"')}"`);
    return `# pod metadata.annotations:\nmetadata:\n  annotations:\n${lines.join('\n')}`;
  }, [activePatterns, target, ns]);

  const labelCmd = `kubectl label namespace ${ns} logcloak.io/inject=true`;

  const updateCustom = (id: number, patch: Partial<Custom>) => {
    setCustoms((cs) => cs.map((c) => (c.id === id ? { ...c, ...patch } : c)));
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[380px_1fr]">
      <div className="space-y-6">
        <div>
          <p className="mb-2 font-mono text-xs font-semibold uppercase tracking-wider text-muted-foreground">Namespace</p>
          <input
            type="text"
            placeholder="production"
            value={namespace}
            onChange={(e) => setNamespace(e.target.value)}
            className="w-full rounded-md border border-border bg-card px-3 py-2 font-mono text-sm text-foreground placeholder:text-muted-foreground/50 focus:border-primary focus:outline-none"
          />
        </div>

        <div>
          <p className="mb-2 font-mono text-xs font-semibold uppercase tracking-wider text-muted-foreground">Sample log line</p>
          <textarea
            placeholder="user jane.doe@example.com placed order ORD-20260423"
            value={sampleLog}
            onChange={(e) => setSampleLog(e.target.value)}
            rows={3}
            className="w-full resize-y rounded-md border border-border bg-card px-3 py-2 font-mono text-sm text-foreground placeholder:text-muted-foreground/50 focus:border-primary focus:outline-none"
          />
        </div>

        <div>
          <p className="mb-2 font-mono text-xs font-semibold uppercase tracking-wider text-muted-foreground">Built-in patterns</p>
          <div className="grid grid-cols-2 gap-1.5">
            {BUILTINS.map((b) => (
              <label
                key={b.name}
                className="flex cursor-pointer items-start gap-2 rounded-md border border-border bg-card p-2 text-xs hover:border-primary/40"
              >
                <input
                  type="checkbox"
                  checked={!!builtinChecked[b.name]}
                  onChange={(e) => setBuiltinChecked((prev) => ({ ...prev, [b.name]: e.target.checked }))}
                  className="mt-0.5 accent-primary"
                />
                <span>
                  <span className="block font-mono font-bold text-foreground">{b.name}</span>
                  <span className="block text-muted-foreground">{b.desc}</span>
                </span>
              </label>
            ))}
          </div>
        </div>

        <div>
          <div className="mb-2 flex items-center justify-between">
            <p className="font-mono text-xs font-semibold uppercase tracking-wider text-muted-foreground">Custom patterns</p>
            <button
              type="button"
              onClick={() => {
                setCustoms((cs) => [...cs, { id: nextId, tab: 'write', name: '', regex: '', sample: '' }]);
                setNextId((n) => n + 1);
              }}
              className="rounded-md border border-border px-2 py-1 text-xs text-muted-foreground hover:border-primary/40 hover:text-primary"
            >
              + add
            </button>
          </div>
          <div className="space-y-3">
            {customs.map((c) => {
              const validity = c.regex ? re2check(c.regex) : '';
              return (
                <div key={c.id} className="rounded-md border border-border bg-card p-3">
                  <div className="mb-2 flex items-center justify-between">
                    <span className="font-mono text-xs text-muted-foreground">custom #{c.id}</span>
                    <button
                      type="button"
                      onClick={() => setCustoms((cs) => cs.filter((x) => x.id !== c.id))}
                      className="text-muted-foreground hover:text-foreground"
                      aria-label="remove"
                    >
                      ✕
                    </button>
                  </div>
                  <input
                    type="text"
                    placeholder="order-id"
                    value={c.name}
                    onChange={(e) => updateCustom(c.id, { name: e.target.value })}
                    className="mb-2 w-full rounded border border-border bg-background px-2 py-1.5 font-mono text-xs text-foreground focus:border-primary focus:outline-none"
                  />
                  <div className="mb-2 flex gap-1">
                    <button
                      type="button"
                      onClick={() => updateCustom(c.id, { tab: 'write' })}
                      className={`rounded px-2 py-1 text-xs ${c.tab === 'write' ? 'bg-primary/15 text-primary' : 'text-muted-foreground'}`}
                    >
                      write regex
                    </button>
                    <button
                      type="button"
                      onClick={() => updateCustom(c.id, { tab: 'suggest' })}
                      className={`rounded px-2 py-1 text-xs ${c.tab === 'suggest' ? 'bg-primary/15 text-primary' : 'text-muted-foreground'}`}
                    >
                      from sample
                    </button>
                  </div>
                  {c.tab === 'write' ? (
                    <input
                      type="text"
                      placeholder="ORD-[0-9]{8}"
                      value={c.regex}
                      onChange={(e) => updateCustom(c.id, { regex: e.target.value })}
                      className="w-full rounded border border-border bg-background px-2 py-1.5 font-mono text-xs text-foreground focus:border-primary focus:outline-none"
                    />
                  ) : (
                    <div className="space-y-2">
                      <div className="flex gap-1.5">
                        <input
                          type="text"
                          placeholder="ORD-20260423"
                          value={c.sample}
                          onChange={(e) => updateCustom(c.id, { sample: e.target.value })}
                          className="flex-1 rounded border border-border bg-background px-2 py-1.5 font-mono text-xs text-foreground focus:border-primary focus:outline-none"
                        />
                        <button
                          type="button"
                          onClick={() => updateCustom(c.id, { regex: suggestRegex(c.sample) })}
                          className="rounded border border-border px-2 py-1 text-xs text-muted-foreground hover:border-primary/40 hover:text-primary"
                        >
                          suggest
                        </button>
                      </div>
                      {c.regex && (
                        <input
                          type="text"
                          value={c.regex}
                          onChange={(e) => updateCustom(c.id, { regex: e.target.value })}
                          className="w-full rounded border border-border bg-background px-2 py-1.5 font-mono text-xs text-foreground focus:border-primary focus:outline-none"
                        />
                      )}
                    </div>
                  )}
                  {c.regex && (
                    <p className={`mt-1.5 text-xs ${validity === 'ok' ? 'text-brand-good' : 'text-brand-bad'}`}>
                      {validity === 'ok' ? '✓ valid RE2' : `✗ ${validity}`}
                    </p>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <div className="space-y-6">
        <div>
          <p className="mb-2 font-mono text-xs font-semibold uppercase tracking-wider text-muted-foreground">Live preview</p>
          <div className="rounded-lg border border-border bg-[#0d0d0f] p-4 font-mono text-sm leading-relaxed">
            {sampleLog ? (
              segments.map((s, i) =>
                s.masked ? (
                  <span key={i} className="text-brand-bad">[REDACTED]</span>
                ) : (
                  <span key={i} className="text-foreground/80">{s.text}</span>
                ),
              )
            ) : (
              <span className="text-muted-foreground">paste a log line on the left →</span>
            )}
          </div>
        </div>

        <div>
          <div className="mb-2 flex items-center gap-1">
            <button
              type="button"
              onClick={() => setTarget('policy')}
              className={`rounded-md px-3 py-1.5 text-xs font-semibold ${target === 'policy' ? 'bg-primary text-primary-foreground' : 'border border-border text-muted-foreground'}`}
            >
              MaskingPolicy YAML
            </button>
            <button
              type="button"
              onClick={() => setTarget('annotation')}
              className={`rounded-md px-3 py-1.5 text-xs font-semibold ${target === 'annotation' ? 'bg-primary text-primary-foreground' : 'border border-border text-muted-foreground'}`}
            >
              Pod Annotation
            </button>
          </div>
          <CodeBlock text={yamlOutput} />
        </div>

        <div>
          <p className="mb-2 font-mono text-xs font-semibold uppercase tracking-wider text-muted-foreground">Regex reference</p>
          <CodeBlock text={regexOutput} />
        </div>

        <div>
          <p className="mb-2 font-mono text-xs font-semibold uppercase tracking-wider text-muted-foreground">Opt the namespace in</p>
          <CodeBlock text={labelCmd} />
        </div>
      </div>
    </div>
  );
}

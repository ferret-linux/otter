export interface RunResult {
  stdout: string;
  stderr: string;
}

export interface LogEntry {
  id: number;
  ts: number;
  command: string;
  args: string[];
  input?: string;
  status: 'running' | 'ok' | 'error';
  exitCode?: number;
  durationMs?: number;
  stdout?: string;
  stderr?: string;
}

export type ImageVariant = 'generic' | 'free' | 'nonfree' | 'negativo17';

export interface RegistryEntry {
  name: string;
  architecture: string[];
  built_at: string;
  pulled: boolean;
  staleness: 'not_pulled' | 'current' | 'behind' | 'ahead' | 'unknown';
  behind_count?: number;
  size?: number;
  image: string;
  family: string;
  image_variant: ImageVariant;
  is_reliable_fallback: boolean;
}

const IMAGE_VARIANTS = new Set<ImageVariant>(['generic', 'free', 'nonfree', 'negativo17']);
const STALENESS_VALUES = new Set<RegistryEntry['staleness']>([
  'not_pulled',
  'current',
  'behind',
  'ahead',
  'unknown',
]);

// Older CLI builds don't emit the classification fields. Fill them in so the
// UI keeps working without treating every entry as an odd one-off.
export function normalizeRegistryEntry(value: unknown): RegistryEntry {
  const raw = (value && typeof value === 'object' ? value : {}) as Record<string, unknown>;
  const staleness = raw.staleness;
  const imageVariant = raw.image_variant;
  const family = raw.family;

  return {
    name: typeof raw.name === 'string' ? raw.name : '',
    architecture: Array.isArray(raw.architecture)
      ? raw.architecture.filter((arch): arch is string => typeof arch === 'string')
      : [],
    built_at: typeof raw.built_at === 'string' ? raw.built_at : '',
    pulled: raw.pulled === true,
    staleness:
      typeof staleness === 'string' && STALENESS_VALUES.has(staleness as RegistryEntry['staleness'])
        ? (staleness as RegistryEntry['staleness'])
        : 'unknown',
    behind_count:
      typeof raw.behind_count === 'number' && Number.isFinite(raw.behind_count)
        ? raw.behind_count
        : undefined,
    size: typeof raw.size === 'number' && Number.isFinite(raw.size) ? raw.size : undefined,
    image: typeof raw.image === 'string' ? raw.image : '',
    family: typeof family === 'string' && family.trim() ? family : 'other',
    image_variant:
      typeof imageVariant === 'string' && IMAGE_VARIANTS.has(imageVariant as ImageVariant)
        ? (imageVariant as ImageVariant)
        : 'generic',
    is_reliable_fallback:
      typeof raw.is_reliable_fallback === 'boolean' ? raw.is_reliable_fallback : true,
  };
}

export function normalizeRegistryEntries(value: unknown): RegistryEntry[] {
  return Array.isArray(value) ? value.map(normalizeRegistryEntry) : [];
}

export interface DataContainer {
  id: string;
  name: string;
  status: string;
  image: string;
}

export interface SettingsEntry {
  section: string;
  field: string;
  kind: 'text' | 'toggle';
  value: string | boolean;
  description: string;
  options?: string[];
  numeric?: boolean;
}

export interface OtterApi {
  run: (command: string, args: string[], input?: string) => Promise<RunResult>;
  runStream: (command: string, args: string[], input?: string) => Promise<RunResult>;
  onRunStep: (callback: (message: string | null) => void) => () => void;
  getLog: () => Promise<LogEntry[]>;

  clearLog: () => Promise<void>;
  onLog: (callback: (entry: LogEntry) => void) => () => void;
  expandEnv: (value: string) => Promise<string>;
}

declare global {
  interface Window {
    otter: OtterApi;
  }
}
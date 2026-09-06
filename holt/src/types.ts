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

export interface RegistryEntry {
  name: string;
  architecture: string[];
  built_at: string;
  pulled: boolean;
  staleness: 'not_pulled' | 'current' | 'behind' | 'ahead' | 'unknown';
  behind_count?: number;
  size?: number;
  image: string;
}

export interface ContainerInfo {
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
  getLog: () => Promise<LogEntry[]>;
  clearLog: () => Promise<void>;
  onLog: (callback: (entry: LogEntry) => void) => () => void;
}

declare global {
  interface Window {
    otter: OtterApi;
  }
}
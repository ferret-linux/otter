export interface RunResult {
  stdout: string;
  stderr: string;
}

export interface RegistryEntry {
  name: string;
  architecture: string[];
  enabled: boolean;
  built_at: string;
  pulled: boolean;
  staleness: string;
  size: number;
  image: string;
}

export interface OtterApi {
  run: (command: string, args: string[]) => Promise<RunResult>;
}

declare global {
  interface Window {
    otter: OtterApi;
  }
}

export type {}
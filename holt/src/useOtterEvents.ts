import { useEffect } from 'react';

const CHANGE_ACTIONS = new Set(['create', 'start', 'stop', 'restart', 'pause', 'remove', 'rm']);

export default function useOtterEvents(onChange: () => void): void {
  useEffect(() => {
    return window.otter.onLog((entry) => {
      if (entry.status === 'running') return;
      if (entry.args[0] === 'reg') {
        const sub = entry.args[1];
        if (sub === 'pull' || sub === 'remove') onChange();
        return;
      }
      if (CHANGE_ACTIONS.has(entry.args[0])) onChange();
    });
  }, [onChange]);
}
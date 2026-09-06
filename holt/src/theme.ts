import { createTheme } from '@mui/material/styles';

// Material Design 3 dark theme, seeded from the otter teal.
// MUI derives the tonal palette (light/dark/contrastText and all colored
// surfaces) automatically from primary.main; only the base surfaces are
// pinned to the M3 dark-scheme values so the app feels like a dark
// Android surface.
export const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#00BCD4' },
    background: {
      default: '#191C1C',
      paper: '#1E2121',
    },
    divider: '#414947',
    error: { main: '#FFB4AB' },
  },
  shape: { borderRadius: 12 },
  typography: {
    button: { textTransform: 'none' },
  },
});
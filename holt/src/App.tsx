import { lazy, Suspense } from 'react';
import { styled } from '@mui/material/styles';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import AppBar from '@mui/material/AppBar';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import Drawer from '@mui/material/Drawer';
import Divider from '@mui/material/Divider';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import AddCircleOutlinedIcon from '@mui/icons-material/AddCircleOutlined';
import HomeIcon from '@mui/icons-material/Home';
import LayersIcon from '@mui/icons-material/Layers';
import SettingsIcon from '@mui/icons-material/Settings';
import TerminalOutlinedIcon from '@mui/icons-material/TerminalOutlined';
import { NavLink, Navigate, useLocation } from 'react-router-dom';

const Home = lazy(() => import('./pages/Home'));
const Create = lazy(() => import('./pages/Create'));
const Registry = lazy(() => import('./pages/Registry'));
const Settings = lazy(() => import('./pages/Settings'));
const Verbose = lazy(() => import('./pages/Verbose'));

const DRAWER_WIDTH = 240;

const StyledToolbar = styled(Toolbar)(({ theme }) => ({
  fontWeight: 700,
  letterSpacing: 0.5,
  color: theme.palette.text.primary,
}));

const StyledNavLink = styled(NavLink)(({ theme }) => ({
  textDecoration: 'none',
  color: theme.palette.text.primary,
  '&.active .MuiListItemButton-root': {
    backgroundColor: theme.palette.primary.main + '22',
    color: theme.palette.primary.main,
  },
}));

export default function App() {
  const location = useLocation();

  const pages = [
    { path: '/', key: 'home', element: <Home /> },
    { path: '/create', key: 'create', element: <Create /> },
    { path: '/registry', key: 'registry', element: <Registry /> },
    { path: '/settings', key: 'settings', element: <Settings /> },
    { path: '/verbose', key: 'verbose', element: <Verbose /> },
  ];
  const known = pages.some((p) => p.path === location.pathname);

  const nav = (
    <List>
      <StyledNavLink to="/" end>
        <ListItem disablePadding>
          <ListItemButton>
            <ListItemIcon>
              <HomeIcon />
            </ListItemIcon>
            <ListItemText primary="Home" />
          </ListItemButton>
        </ListItem>
      </StyledNavLink>
      <StyledNavLink to="/create">
        <ListItem disablePadding>
          <ListItemButton>
            <ListItemIcon>
              <AddCircleOutlinedIcon />
            </ListItemIcon>
            <ListItemText primary="Create" />
          </ListItemButton>
        </ListItem>
      </StyledNavLink>
      <StyledNavLink to="/registry">
        <ListItem disablePadding>
          <ListItemButton>
            <ListItemIcon>
              <LayersIcon />
            </ListItemIcon>
            <ListItemText primary="Registry" />
          </ListItemButton>
        </ListItem>
      </StyledNavLink>
    </List>
  );

  const bottomNav = (
    <Box sx={{ mt: 'auto', pb: 1 }}>
      <Divider sx={{ mx: 2, mb: 1 }} />
      <StyledNavLink to="/verbose">
        <ListItem disablePadding>
          <ListItemButton>
            <ListItemIcon>
              <TerminalOutlinedIcon />
            </ListItemIcon>
            <ListItemText primary="Verbose" />
          </ListItemButton>
        </ListItem>
      </StyledNavLink>
      <StyledNavLink to="/settings">
        <ListItem disablePadding>
          <ListItemButton>
            <ListItemIcon>
              <SettingsIcon />
            </ListItemIcon>
            <ListItemText primary="Settings" />
          </ListItemButton>
        </ListItem>
      </StyledNavLink>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <AppBar position="fixed" elevation={0} color="default">
        <StyledToolbar sx={{ pl: 2 }}>
          <Typography variant="h6" component="div" sx={{ fontWeight: 700 }}>
            Holt
          </Typography>
        </StyledToolbar>
      </AppBar>

      <Drawer
        variant="permanent"
        sx={{
          width: DRAWER_WIDTH,
          flexShrink: 0,
          [`& .MuiDrawer-paper`]: {
            width: DRAWER_WIDTH,
            boxSizing: 'border-box',
            mt: '64px',
            borderRight: 0,
          },
        }}
      >
        <Box sx={{ display: 'flex', flexDirection: 'column', height: 'calc(100% - 64px)', overflow: 'auto' }}>
          {nav}
          {bottomNav}
        </Box>
      </Drawer>

      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 3,
          mt: '64px',
          width: { sm: `calc(100% - ${DRAWER_WIDTH}px)` },
        }}
      >
        <Suspense
          fallback={
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 12 }}>
              <CircularProgress />
            </Box>
          }
        >
          {known
            ? pages.map((p) => (
                <Box key={p.key} sx={{ display: location.pathname === p.path ? 'block' : 'none' }}>
                  {p.element}
                </Box>
              ))
            : <Navigate to="/" replace />}
        </Suspense>
      </Box>
    </Box>
  );
}
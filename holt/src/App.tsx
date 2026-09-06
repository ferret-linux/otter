import { styled } from '@mui/material/styles';
import Box from '@mui/material/Box';
import AppBar from '@mui/material/AppBar';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import Drawer from '@mui/material/Drawer';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import HomeIcon from '@mui/icons-material/Home';
import LayersIcon from '@mui/icons-material/Layers';
import { NavLink, Navigate, Route, Routes } from 'react-router-dom';
import Home from './pages/Home';
import Registry from './pages/Registry';

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
        <Box sx={{ overflow: 'auto' }}>{nav}</Box>
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
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/registry" element={<Registry />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Box>
    </Box>
  );
}
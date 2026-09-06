import { styled } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Chip from '@mui/material/Chip';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import LayersIcon from '@mui/icons-material/Layers';
import DownloadIcon from '@mui/icons-material/Download';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import { Link } from 'react-router-dom';

const Hero = styled(Box)(({ theme }) => ({
  display: 'flex',
  alignItems: 'center',
  gap: theme.spacing(2),
  mb: 4,
}));

const FeatureIcon = styled(Box)(({ theme }) => ({
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 40,
  height: 40,
  borderRadius: 12,
  backgroundColor: theme.palette.primary.main + '22',
  color: theme.palette.primary.main,
}));

const features = [
  {
    icon: <LayersIcon />,
    title: 'Browse images',
    body: 'See every otter image in the registry, its architecture, size, and pull state at a glance.',
  },
  {
    icon: <DownloadIcon />,
    title: 'Pull images',
    body: 'Fetch the latest build of an otter image straight from the registry with a single click.',
  },
  {
    icon: <DeleteOutlinedIcon />,
    title: 'Manage storage',
    body: 'Remove images you no longer need and keep your container storage tidy.',
  },
];

export default function Home() {
  return (
    <Box>
      <Hero>
        <Box>
          <Chip label="osc" sx={{ mb: 1 }} color="primary" variant="outlined" size="small" />
          <Typography variant="h4" component="h1" sx={{ fontWeight: 700 }}>
            Welcome to Holt
          </Typography>
          <Typography color="text.secondary" sx={{ mt: 1 }}>
            A Material Design 3 desktop UI for managing otter containers.
          </Typography>
        </Box>
      </Hero>

      <Grid container spacing={3}>
        {features.map((f) => (
          <Grid size={{ xs: 12, md: 4 }} key={f.title}>
            <Card elevation={1}>
              <CardContent>
                <FeatureIcon>{f.icon}</FeatureIcon>
                <Typography variant="h6" sx={{ mt: 2, fontWeight: 600 }}>
                  {f.title}
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                  {f.body}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      <Box sx={{ mt: 4 }}>
        <Button
          component={Link}
          to="/registry"
          variant="contained"
          color="primary"
          endIcon={<ArrowForwardIcon />}
        >
          Browse the registry
        </Button>
      </Box>
    </Box>
  );
}
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import ConstructionIcon from '@mui/icons-material/Construction';

export default function Registry() {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 2,
        py: 12,
      }}
    >
      <ConstructionIcon sx={{ fontSize: 48, color: 'primary.main' }} />
      <Typography variant="h5" component="h2" sx={{ fontWeight: 600 }}>
        Registry list coming soon
      </Typography>
      <Typography color="text.secondary" align="center">
        The full image list with pull and remove actions is next on the build list.
      </Typography>
    </Box>
  );
}
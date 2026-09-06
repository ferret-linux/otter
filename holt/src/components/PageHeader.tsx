import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Tooltip from '@mui/material/Tooltip';
import IconButton from '@mui/material/IconButton';
import CircularProgress from '@mui/material/CircularProgress';
import RefreshIcon from '@mui/icons-material/Refresh';

interface PageHeaderProps {
  title: string;
  subtitle: string;
  loading?: boolean;
  onRefresh?: () => void;
}

export default function PageHeader({
  title,
  subtitle,
  loading = false,
  onRefresh,
}: PageHeaderProps) {
  return (
    <Stack
      sx={{
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        mb: 2,
      }}
    >
      <Box>
        <Typography variant="h5" component="h1" sx={{ fontWeight: 700 }}>
          {title}
        </Typography>
        <Typography color="text.secondary" variant="body2">
          {subtitle}
        </Typography>
      </Box>
      {onRefresh && (
        <Tooltip title="Refresh">
          <span>
            <IconButton onClick={onRefresh} disabled={loading} aria-label="Refresh list">
              {loading ? <CircularProgress size={20} /> : <RefreshIcon />}
            </IconButton>
          </span>
        </Tooltip>
      )}
    </Stack>
  );
}
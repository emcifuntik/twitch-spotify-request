// API Response Types
export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// User Types
export interface User {
  id: string;
  authenticated: boolean;
}

export interface UserProfile {
  id: number;
  channel_id: string;
  name: string;
  has_spotify_linked: boolean;
  has_twitch_linked: boolean;
  rewards_configured: boolean;
}

// Track Types
export interface Track {
  name: string;
  artists: string[];
  duration: number;
  uri?: string;
  progress?: number;
  image?: string;
}

// Queue Types
export interface QueueData {
  current_song: string;
  current_song_image?: string;
  current_song_artists?: string[];
  progress: number;
  duration: number;
  queue: Track[];
  timestamp: number;
}

// Streamer Types
export interface Streamer {
  id: number;
  name: string;
}

// Component Props Types
export interface QueueItemProps {
  track: Track;
  isCurrentlyPlaying?: boolean;
  showDuration?: boolean;
  compact?: boolean;
}

export interface StatusIndicatorProps {
  status: 'connected' | 'disconnected' | 'warning';
  label: string;
}

export interface LoadingSpinnerProps {
  message?: string;
}

// Auth Types
export interface AuthContextType {
  user: User | null;
  login: () => void;
  logout: () => void;
  loading: boolean;
}

// Form Types
export interface UserSettings {
  [key: string]: any;
}

// Settings Types
export interface StreamerSettings {
  max_song_length: number;
  cooldown_same_song: number;
  web_ui_enabled: boolean;
}

// Block Types
export interface Block {
  id: number;
  spotify_id: string;
  name: string;
  type: 'artist' | 'track';
}

export interface BlockRequest {
  spotify_id: string;
  name: string;
  type: 'artist' | 'track';
}

// Spotify Search Types
export interface SpotifySearchResult {
  id: string;
  name: string;
  type: 'artist' | 'track';
  image?: string;
  artists?: string[]; // For tracks
}

export interface SpotifySearchResponse {
  results: SpotifySearchResult[];
}

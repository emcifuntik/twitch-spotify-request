# Twitch Spotify Request Bot v2.0

A modern Twitch bot that allows viewers to request songs on Spotify using channel points or chat commands. Built with React frontend and Go backend.

## Features

- 🎵 **Song Requests**: Viewers can request songs using Twitch channel points
- ⏭️ **Skip Control**: Allow trusted users to skip songs
- 💬 **Chat Commands**: Support for chat-based commands (!sc, !sq, !sr, !volume)
- 🌐 **Web Interface**: Modern React-based web interface for streamers and viewers
- 📋 **Public Queue**: Public page showing current queue for OBS/stream overlay
- ⚙️ **User Cabinet**: Settings panel for streamers to manage their bot
- 🔄 **Auto Refresh**: Real-time updates using Spotify's native queue API

## Architecture

- **Frontend**: React with Vite, custom Twitch-like UI
- **Backend**: Go with Gorilla Mux, Helix (Twitch API), zmb3/spotify
- **Database**: MariaDB for user settings and rewards
- **Deployment**: Docker with multi-stage builds

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Twitch application (Client ID & Secret)
- Spotify application (Client ID & Secret)

### Development Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd twitch-spotify-request
   ```

2. **Install dependencies**
   ```bash
   npm run install
   ```

3. **Set up environment**
   ```bash
   # Copy and edit environment file
   cp default.env .env
   # Edit .env with your Twitch/Spotify credentials
   ```

4. **Start development servers**
   ```bash
   # Frontend (React + Vite)
   npm run dev:frontend

   # Backend (Go)
   npm run dev:backend
   ```

### Production Deployment

1. **Build and run with Docker**
   ```bash
   npm run docker:build
   npm run docker:up
   ```

2. **Check logs**
   ```bash
   npm run docker:logs
   ```

## Environment Variables

Create a `.env` file or set these environment variables:

```env
# Database
DB_HOST=localhost
DB_PORT=3306
DB_NAME=twspoty
DB_USER=twitch_user
DB_PASSWORD=twitch_pass

# Bot Configuration
BOT_HOST=https://your-domain.com/

# Twitch API
TWITCH_CLIENT_ID=your_twitch_client_id
TWITCH_CLIENT_SECRET=your_twitch_client_secret
EVENTSUB_SECRET=your_random_secret

# Spotify API
SPOTIFY_CLIENT_ID=your_spotify_client_id
SPOTIFY_CLIENT_SECRET=your_spotify_client_secret
```

## API Endpoints

### Public Endpoints
- `GET /api/streamers` - List active streamers
- `GET /api/streamer/{id}/queue` - Get public queue for a streamer

### User Endpoints
- `GET /api/user/{id}/profile` - Get user profile
- `GET /api/user/{id}/queue` - Get user's queue
- `POST /api/user/{id}/settings` - Update user settings

### Auth Endpoints
- `GET /auth` - Start authentication flow
- `GET /oauth/twitch` - Twitch OAuth callback
- `GET /oauth/spotify` - Spotify OAuth callback

## Frontend Routes

- `/` - Landing page with active streamers
- `/dashboard` - User dashboard and settings
- `/queue/{streamerId}` - Public queue page for viewers
- `/queue-compact/{streamerId}` - Compact queue overlay for OBS/streaming software

## Chat Commands

- `!sc` - Show current song
- `!sq` - Show queue
- `!sr` - Show recent songs  
- `!volume <0-100>` - Change volume (mods/broadcaster only)
- `!songhelp` - Show available commands

## Channel Point Rewards

The bot automatically creates two channel point rewards:
- **Request Song** - Add a song to the queue
- **Skip Song** - Skip the current song

## Database Schema

See `db-model/` directory for the database schema and model files.

## Development

### Project Structure

```
├── frontend/          # React frontend
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   └── ...
│   └── package.json
├── backend/           # Go backend
│   ├── cmd/server/
│   ├── internal/
│   │   ├── api/
│   │   ├── db/
│   │   ├── handlers/
│   │   ├── spotify/
│   │   └── twitch/
│   └── go.mod
├── docker-compose.yml
├── Dockerfile
└── package.json       # Root package.json for scripts
```

## Migration from v1

This v2 rewrite includes:
- ✅ Complete port from TypeScript/Node.js to Go
- ✅ Modern React frontend replacing basic HTML/JS
- ✅ REST API for frontend communication
- ✅ Docker multi-stage builds
- ✅ EventSub instead of IRC for chat reading
- ✅ Native Spotify queue API instead of custom implementation
- ✅ Improved error handling and logging

## Usage

### Compact Queue Overlay

The compact queue route (`/queue-compact/{streamerId}`) provides a minimal, overlay-friendly view perfect for OBS or streaming software:

**Features:**
- **Minimal Design**: Shows only currently playing track and progress
- **Fixed Position**: Positioned at bottom of screen
- **No Navigation**: No navbar or other UI elements
- **Real-time Updates**: Updates every 30 seconds with smooth progress animation
- **Responsive**: Works well at different resolutions

**Usage in OBS:**
1. Add a new Browser Source
2. Set URL to: `http://localhost:5173/queue-compact/{your-streamer-id}`
3. Set width/height as needed (recommended: 800x100)
4. The overlay will automatically show the current track

**URL Format:**
```
http://localhost:5173/queue-compact/{streamerId}
```

Where `{streamerId}` can be:
- Numeric streamer ID
- Twitch channel ID  
- Twitch username

### Regular Queue Page

The regular queue route (`/queue/{streamerId}`) provides a full-featured queue view:

**Features:**
- **Full Queue Display**: Shows current track and upcoming songs
- **Progress Tracking**: Real-time progress bar and time remaining
- **Auto-refresh**: Updates every 30 seconds
- **Responsive Design**: Works on desktop and mobile

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
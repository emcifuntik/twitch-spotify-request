import React from 'react'
import { QueueItemProps } from '../types'

const QueueItem: React.FC<QueueItemProps> = ({ 
  track, 
  isCurrentlyPlaying = false, 
  showDuration = true, 
  compact = false 
}) => {
  const formatDuration = (ms: number): string => {
    if (!ms || isNaN(ms)) return '0:00'
    const minutes = Math.floor(ms / 60000)
    const seconds = Math.floor((ms % 60000) / 1000)
    return `${minutes}:${seconds.toString().padStart(2, '0')}`
  }

  // Ensure track is valid
  if (!track) {
    return (
      <div className="queue-item">
        <div className="track-title">Unknown Track</div>
        <div className="track-artist">Unknown Artist</div>
      </div>
    )
  }

  const trackName = track.name || 'Unknown Track'
  const trackArtists = track.artists 
    ? (Array.isArray(track.artists) ? track.artists.join(', ') : track.artists)
    : 'Unknown Artist'

  // Compact mode - minimal display with only title, artist, and progress bar
  if (compact) {
    return (
      <div className="queue-compact">
        <div className="compact-current-track">
          {track.image && (
            <div className="compact-track-image">
              <img src={track.image} alt={trackName} />
            </div>
          )}
          <div className="compact-track-info">
            <div className="track-name">{trackName}</div>
            <div className="track-artist">{trackArtists}</div>
          </div>
        </div>
        
        {isCurrentlyPlaying && track.progress && track.duration && (
          <div className="compact-progress-bar">
            <div 
              className="compact-progress-fill"
              style={{ 
                width: `${Math.max(0, Math.min(100, (track.progress / track.duration) * 100))}%` 
              }}
            ></div>
          </div>
        )}
      </div>
    )
  }

  return (
    <div className={`queue-item ${isCurrentlyPlaying ? 'current' : ''}`}>
      <div className="queue-item-content">
        {track.image && (
          <div className="track-image">
            <img src={track.image} alt={trackName} />
          </div>
        )}
        <div className="track-info">
          <div className="track-title">{trackName}</div>
          <div className="track-artist">{trackArtists}</div>
        </div>
        {showDuration && track.duration && (
          <div className="track-duration">
            {formatDuration(track.duration)}
          </div>
        )}
      </div>
      
      {isCurrentlyPlaying && track.progress && track.duration && (
        <div className="progress-bar">
          <div 
            className="progress-fill"
            style={{ 
              width: `${Math.max(0, Math.min(100, (track.progress / track.duration) * 100))}%` 
            }}
          ></div>
        </div>
      )}
    </div>
  )
}

export default QueueItem

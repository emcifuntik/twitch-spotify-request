import React from 'react'
import { useState, useEffect, useRef } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import LoadingSpinner from '../components/LoadingSpinner'
import QueueItem from '../components/QueueItem'
import { QueueData } from '../types'
import axios from 'axios'

const Queue: React.FC = () => {
  const { streamerId } = useParams<{ streamerId: string }>()
  const [searchParams] = useSearchParams()
  const [queueData, setQueueData] = useState<QueueData | null>(null)
  const [loading, setLoading] = useState<boolean>(true)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [autoRefresh, setAutoRefresh] = useState<boolean>(true)
  const [currentProgress, setCurrentProgress] = useState<number>(0)
  const progressInterval = useRef<number | null>(null)
  const refreshInterval = useRef<number | null>(null)
  
  // Check if this is compact mode for OBS overlay
  const isCompact = searchParams.get('compact') === 'true'

  useEffect(() => {
    if (streamerId) {
      loadQueue()
    }
  }, [streamerId])

  useEffect(() => {
    // Start progress animation
    if (queueData?.current_song && queueData.duration > 0) {
      setCurrentProgress(queueData.progress)
      startProgressAnimation()
    } else {
      stopProgressAnimation()
    }

    return () => stopProgressAnimation()
  }, [queueData])

  useEffect(() => {
    // Auto refresh queue
    if (autoRefresh && streamerId) {
      refreshInterval.current = window.setInterval(() => {
        loadQueue(false) // Silent refresh
      }, 30000) // Refresh every 30 seconds
    }
    return () => {
      if (refreshInterval.current) clearInterval(refreshInterval.current)
    }
  }, [autoRefresh, streamerId])

  const startProgressAnimation = () => {
    stopProgressAnimation()
    
    if (!queueData || queueData.duration <= 0) return

    progressInterval.current = window.setInterval(() => {
      setCurrentProgress(prev => {
        const newProgress = prev + 1000 // Add 1 second (1000ms)
        
        // If song is finished, refresh queue
        if (newProgress >= queueData.duration) {
          loadQueue(false)
          return queueData.duration
        }
        
        return newProgress
      })
    }, 1000)
  }

  const stopProgressAnimation = () => {
    if (progressInterval.current) {
      clearInterval(progressInterval.current)
      progressInterval.current = null
    }
  }

  const loadQueue = async (showLoading: boolean = true): Promise<void> => {
    try {
      if (showLoading) {
        setLoading(true)
        setError(null)
      }
      
      // Try multiple endpoints to handle different ID formats
      let response
      try {
        // First try as numeric ID
        response = await axios.get(`/api/streamer/${streamerId}/queue`)
      } catch (err: any) {
        if (err.response?.status === 400 || err.response?.status === 404) {
          // If that fails, try to find streamer by channel ID
          const streamersResponse = await axios.get('/api/streamers')
          const streamers = streamersResponse.data?.data || []
          const streamer = streamers.find((s: any) => s.channel_id === streamerId || s.name === streamerId)
          
          if (streamer) {
            response = await axios.get(`/api/streamer/${streamer.id}/queue`)
          } else {
            throw new Error('Streamer not found')
          }
        } else {
          throw err
        }
      }
      
      console.log('Queue response:', response.data) // Debug log
      
      if (response.data && response.data.success) {
        const newQueueData = response.data.data
        setQueueData(newQueueData)
        setLastUpdated(new Date())
        setError(null)
        
        // Update current progress if track changed
        if (queueData?.current_song !== newQueueData?.current_song) {
          setCurrentProgress(newQueueData?.progress || 0)
        }
      } else {
        const errorMsg = response.data?.error || 'Failed to load queue'
        setError(errorMsg)
        console.warn('Queue request failed:', response.data)
      }
    } catch (err: any) {
      console.error('Error loading queue:', err)
      const errorMsg = err.response?.data?.error || err.message || 'Failed to load queue'
      setError(errorMsg)
    } finally {
      setLoading(false)
    }
  }

  const formatTime = (ms: number): string => {
    const totalSeconds = Math.floor(ms / 1000)
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = totalSeconds % 60
    return `${minutes}:${seconds.toString().padStart(2, '0')}`
  }

  const formatLastUpdated = (date: Date): string => {
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffSeconds = Math.floor(diffMs / 1000)
    const diffMinutes = Math.floor(diffSeconds / 60)
    const diffHours = Math.floor(diffMinutes / 60)

    if (diffSeconds < 60) {
      return `${diffSeconds} seconds ago`
    } else if (diffMinutes < 60) {
      return `${diffMinutes} minute${diffMinutes !== 1 ? 's' : ''} ago`
    } else if (diffHours < 24) {
      return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`
    } else {
      return date.toLocaleString()
    }
  }

  const getProgressPercentage = (): number => {
    if (!queueData || queueData.duration <= 0) return 0
    return Math.min(100, (currentProgress / queueData.duration) * 100)
  }

  if (loading && !queueData) {
    return (
      <div className={`queue-container ${isCompact ? 'compact' : ''}`}>
        <LoadingSpinner message="Loading queue..." />
      </div>
    )
  }

  if (error && !queueData) {
    return (
      <div className={`queue-container ${isCompact ? 'compact' : ''}`}>
        <div className="alert alert-error">
          {error}
          <button onClick={() => loadQueue()} className="btn btn-primary" style={{ marginLeft: '1rem' }}>
            Try Again
          </button>
        </div>
      </div>
    )
  }

  if (isCompact) {
    // Compact view for OBS overlay
    return (
      <div className="queue-compact">
        {queueData?.current_song ? (
          <>
            <div className="compact-current-track">
              {queueData.current_song_image && (
                <div className="compact-track-image">
                  <img src={queueData.current_song_image} alt={queueData.current_song} />
                </div>
              )}
              <div className="compact-track-info">
                {queueData.current_song_artists && queueData.current_song_artists.length > 0 ? (
                  <>
                    <div className="track-name">
                      {queueData.current_song.includes(' - ') 
                        ? queueData.current_song.split(' - ').slice(1).join(' - ')
                        : queueData.current_song}
                    </div>
                    <div className="track-artist">
                      {Array.isArray(queueData.current_song_artists) 
                        ? queueData.current_song_artists.join(', ') 
                        : queueData.current_song_artists}
                    </div>
                  </>
                ) : (
                  <div className="track-name">{queueData.current_song}</div>
                )}
                <div className="track-time">
                  {formatTime(currentProgress)} / {formatTime(queueData.duration)}
                </div>
              </div>
            </div>
            <div className="compact-progress-bar">
              <div 
                className="compact-progress-fill"
                style={{ 
                  width: `${getProgressPercentage()}%`,
                  transition: 'width 1s linear'
                }}
              />
            </div>
          </>
        ) : (
          <div className="compact-no-track">
            <div className="no-track-message">No track playing</div>
          </div>
        )}
      </div>
    )
  }

  // Regular view
  return (
    <div className="queue-container">
      {/* Header */}
      <div className="queue-header">
        <h1>🎵 Song Queue</h1>
        <div className="queue-controls">
          <button 
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`btn btn-small ${autoRefresh ? 'btn-success' : 'btn-secondary'}`}
          >
            {autoRefresh ? '⏸️' : '▶️'} Auto-refresh
          </button>
          <button 
            onClick={() => loadQueue()} 
            className="btn btn-secondary btn-small"
            disabled={loading}
          >
            🔄 Refresh
          </button>
        </div>
      </div>

      {lastUpdated && (
        <div className="queue-last-updated">
          Last updated: {formatLastUpdated(lastUpdated)}
        </div>
      )}

      {error && (
        <div className="alert alert-warning queue-error">
          Failed to refresh: {error}
        </div>
      )}

      {queueData ? (
        <div className="queue-content">
          {/* Now Playing */}
          <div className="queue-now-playing">
            <h2>Now Playing</h2>
            {queueData.current_song ? (
              <div className="current-track-card">
                <div className="current-track-content">
                  {queueData.current_song_image && (
                    <div className="current-track-image">
                      <img src={queueData.current_song_image} alt={queueData.current_song} />
                    </div>
                  )}
                  <div className="current-track-info">
                    {queueData.current_song_artists && queueData.current_song_artists.length > 0 ? (
                      <>
                        <div className="track-name">
                          {queueData.current_song.includes(' - ') 
                            ? queueData.current_song.split(' - ').slice(1).join(' - ')
                            : queueData.current_song}
                        </div>
                        <div className="track-artist">
                          {Array.isArray(queueData.current_song_artists) 
                            ? queueData.current_song_artists.join(', ') 
                            : queueData.current_song_artists}
                        </div>
                      </>
                    ) : (
                      <div className="track-name">{queueData.current_song}</div>
                    )}
                    <div className="track-meta">
                      {formatTime(currentProgress)} / {formatTime(queueData.duration)}
                    </div>
                  </div>
                </div>
                <div className="progress-bar">
                  <div 
                    className="progress-fill"
                    style={{ 
                      width: `${getProgressPercentage()}%`,
                      transition: 'width 1s linear'
                    }}
                  />
                </div>
              </div>
            ) : (
              <div className="no-track">
                <p>No track currently playing</p>
              </div>
            )}
          </div>

          {/* Queue */}
          <div className="queue-list">
            <h2>Up Next ({queueData.queue?.length || 0})</h2>
            {queueData.queue && queueData.queue.length > 0 ? (
              <div className="queue-items">
                {queueData.queue.map((track, index) => (
                  <QueueItem 
                    key={index}
                    track={track}
                    showDuration={true}
                  />
                ))}
              </div>
            ) : (
              <div className="empty-queue">
                <p>Queue is empty</p>
                <small className="text-muted">Songs will appear here when requested by viewers</small>
              </div>
            )}
          </div>

          {/* Instructions */}
          <div className="queue-instructions">
            <h3>How to Request Songs</h3>
            <div className="instructions-grid">
              <div>
                <h4>Channel Points</h4>
                <p>Use the "Request Song" reward to add songs to the queue.</p>
              </div>
              <div>
                <h4>Chat Commands</h4>
                <ul>
                  <li><strong>!sc</strong> - Current song</li>
                  <li><strong>!sq</strong> - View queue</li>
                  <li><strong>!sr</strong> - Recent songs</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className="no-data">
          <p>No queue data available</p>
        </div>
      )}
    </div>
  )
}

export default Queue

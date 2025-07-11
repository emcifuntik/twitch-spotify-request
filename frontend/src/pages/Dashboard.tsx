import React from 'react'
import { useState, useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { Button, Spinner } from '../components/ui'
import StatusIndicator from '../components/StatusIndicator'
import QueueItem from '../components/QueueItem'
import Settings from '../components/Settings'
import Blocks from '../components/Blocks'
import Moderators from '../components/Moderators'
import { QueueData } from '../types'
import axios from 'axios'

const Dashboard: React.FC = () => {
  const location = useLocation()
  const { user: profile, loading: authLoading } = useAuth()
  const [queue, setQueue] = useState<QueueData | null>(null)
  const [loading, setLoading] = useState<boolean>(true)
  const [error, setError] = useState<string | null>(null)
  const [authSuccess, setAuthSuccess] = useState<boolean>(false)
  const [activeTab, setActiveTab] = useState<string>('overview')
  const [fixingRewards, setFixingRewards] = useState<boolean>(false)
  
  const currentUserId = profile?.channel_id

  useEffect(() => {
    // Check if we were redirected from auth
    const urlParams = new URLSearchParams(location.search)
    if (urlParams.get('auth') === 'success') {
      setAuthSuccess(true)
      
      // Remove the auth parameters from URL
      window.history.replaceState({}, '', '/dashboard')
      
      // Refresh data after successful auth
      setTimeout(() => {
        if (currentUserId) {
          loadUserData()
        }
      }, 1000)
    } else {
      loadUserData()
    }
  }, [location, currentUserId])

  const loadUserData = async (): Promise<void> => {
    if (!currentUserId) {
      setError('No user ID available. Please authenticate first.')
      setLoading(false)
      return
    }

    try {
      setLoading(true)
      setError(null)
      
      // Load queue data
      const queueResponse = await axios.get(`/api/user/${currentUserId}/queue`)
      
      if (queueResponse.data?.success) {
        setQueue(queueResponse.data.data)
      }
    } catch (err: any) {
      console.error('Error loading user data:', err)
      setError(err.response?.data?.error || err.message || 'Failed to load user data')
    } finally {
      setLoading(false)
    }
  }

  const fixRewards = async (): Promise<void> => {
    if (!currentUserId) {
      setError('No user ID available.')
      return
    }

    try {
      setFixingRewards(true)
      setError(null)
      
      const response = await axios.post(`/api/user/${currentUserId}/fix-rewards`)
      
      if (response.data?.success) {
        // Show success message
        setAuthSuccess(true)
        // Refresh the page data to update the rewards status
        window.location.reload()
      }
    } catch (err: any) {
      console.error('Error fixing rewards:', err)
      setError(err.response?.data?.error || err.message || 'Failed to fix rewards')
    } finally {
      setFixingRewards(false)
    }
  }

  // If we're loading
  if (loading && !profile && !queue) {
    return (
      <div className="container">
        <Spinner size="large" />
      </div>
    )
  }

  // If we have an error and no data
  if (error && !profile && !queue) {
    return (
      <div className="container text-center">
        <h1 className="mb-4">Dashboard</h1>
        <div className="alert alert-error">
          {error}
        </div>
        <button onClick={loadUserData} className="btn btn-primary">
          Try Again
        </button>
      </div>
    )
  }

  return (
    <div className="container">
      <h1 className="mb-4">Dashboard</h1>
      
      {authSuccess && (
        <div className="alert alert-success mb-4">
          <div className="flex justify-between align-center">
            <div>
              <strong>🎉 Authentication Successful!</strong> Your Twitch and Spotify accounts have been connected.
            </div>
            <button 
              onClick={() => setAuthSuccess(false)}
              className="btn btn-outline-secondary btn-small"
            >
              ✕
            </button>
          </div>
        </div>
      )}

      {error && (
        <div className="alert alert-error">
          {error}
          <button onClick={loadUserData} className="btn btn-secondary btn-small" style={{ marginLeft: '1rem' }}>
            Retry
          </button>
        </div>
      )}

      {loading && <Spinner size="large" />}

      {!loading && profile && (
        <>
          {/* Profile Overview */}
          <div className="card mb-4">
            <h2 className="mb-3">Account Overview</h2>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="form-group">
                <label className="form-label">Channel Name</label>
                <div className="text-secondary">{profile.name}</div>
              </div>
              
              <div className="form-group">
                <label className="form-label">Channel ID</label>
                <div className="text-secondary">{profile.channel_id}</div>
              </div>
              
              <div className="form-group">
                <label className="form-label">Connections</label>
                <div className="flex gap-2 flex-column">
                  <StatusIndicator 
                    status={profile.has_spotify_linked ? 'connected' : 'disconnected'}
                    label="Spotify"
                  />
                  <StatusIndicator 
                    status={profile.has_twitch_linked ? 'connected' : 'disconnected'}
                    label="Twitch"
                  />
                  <StatusIndicator 
                    status={profile.rewards_configured ? 'connected' : 'warning'}
                    label="Rewards Configured"
                  />
                </div>
              </div>

              {(!profile.has_spotify_linked || !profile.has_twitch_linked) && (
                <div className="alert alert-warning">
                  <strong>Setup Required:</strong> Connect both Spotify and Twitch to enable song requests.
                </div>
              )}
              
              {!profile.rewards_configured && profile.has_spotify_linked && profile.has_twitch_linked && (
                <div className="alert alert-warning">
                  <div className="flex justify-between align-center">
                    <div>
                      <strong>Rewards Setup Issue:</strong> Channel point rewards are not properly configured.
                    </div>
                    <Button 
                      onClick={fixRewards}
                      disabled={fixingRewards}
                      variant="secondary"
                      size="small"
                    >
                      {fixingRewards ? '🔄 Fixing...' : '🔧 Fix Rewards'}
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Tab Navigation */}
          <div className="tabs mb-4">
            <button 
              className={`tab ${activeTab === 'overview' ? 'active' : ''}`}
              onClick={() => setActiveTab('overview')}
            >
              📊 Queue Overview
            </button>
            <button 
              className={`tab ${activeTab === 'settings' ? 'active' : ''}`}
              onClick={() => setActiveTab('settings')}
            >
              ⚙️ Bot Settings
            </button>
            <button 
              className={`tab ${activeTab === 'moderators' ? 'active' : ''}`}
              onClick={() => setActiveTab('moderators')}
            >
              👥 Moderators
            </button>
            <button 
              className={`tab ${activeTab === 'blocks' ? 'active' : ''}`}
              onClick={() => setActiveTab('blocks')}
            >
              🚫 Block Tracks
            </button>
          </div>

          {/* Tab Content */}
          {activeTab === 'overview' && (
            <div className="card">
              <h2 className="mb-3">Current Queue</h2>
              {queue ? (
                <div>
                  {queue.current_song && (
                    <div className="mb-3">
                      <h3 className="text-green mb-2">Now Playing</h3>
                      <QueueItem 
                        track={{
                          name: queue.current_song,
                          artists: ['Current Track'],
                          duration: queue.duration,
                          progress: queue.progress
                        }}
                        isCurrentlyPlaying={true}
                      />
                    </div>
                  )}
                  
                  {queue.queue && queue.queue.length > 0 ? (
                    <div>
                      <h3 className="mb-2">Up Next ({queue.queue.length})</h3>
                      <div style={{ maxHeight: '300px', overflowY: 'auto' }}>
                        {queue.queue.slice(0, 5).map((track, index) => (
                          <QueueItem key={index} track={track} />
                        ))}
                        {queue.queue.length > 5 && (
                          <div className="text-center text-muted mt-2">
                            And {queue.queue.length - 5} more...
                          </div>
                        )}
                      </div>
                    </div>
                  ) : (
                    <div className="text-muted">Queue is empty</div>
                  )}
                </div>
              ) : (
                <div className="text-muted">Failed to load queue</div>
              )}
            </div>
          )}

          {activeTab === 'settings' && currentUserId && (
            <Settings userId={currentUserId} />
          )}

          {activeTab === 'moderators' && currentUserId && (
            <Moderators userId={currentUserId} />
          )}

          {activeTab === 'blocks' && currentUserId && (
            <Blocks userId={currentUserId} />
          )}
        </>
      )}

      {/* Refresh Button */}
      <div className="text-center mt-4">
        <button onClick={loadUserData} className="btn btn-secondary">
          🔄 Refresh Data
        </button>
      </div>
    </div>
  )
}

export default Dashboard

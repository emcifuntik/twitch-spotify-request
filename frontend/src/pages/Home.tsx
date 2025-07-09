import React from 'react'
import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import LoadingSpinner from '../components/LoadingSpinner'
import { Streamer, APIResponse } from '../types'
import axios from 'axios'

const Home: React.FC = () => {
  const [streamers, setStreamers] = useState<Streamer[]>([])
  const [loading, setLoading] = useState<boolean>(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadStreamers()
  }, [])

  const loadStreamers = async (): Promise<void> => {
    try {
      setLoading(true)
      setError(null)
      const response = await axios.get<APIResponse<Streamer[]>>('/api/streamers')
      
      console.log('Streamers response:', response.data) // Debug log
      
      if (response.data && response.data.success && Array.isArray(response.data.data)) {
        setStreamers(response.data.data)
      } else {
        console.warn('Invalid response format:', response.data)
        setStreamers([])
        if (!response.data || !response.data.success) {
          setError(response.data?.error || 'Invalid response from server')
        }
      }
    } catch (err: any) {
      console.error('Error loading streamers:', err)
      setError(err.response?.data?.error || err.message || 'Failed to load streamers')
      setStreamers([])
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <LoadingSpinner message="Loading streamers..." />
  }

  if (error) {
    return (
      <div className="container text-center">
        <div className="alert alert-error">
          {error}
        </div>
        <button onClick={loadStreamers} className="btn btn-primary">
          Try Again
        </button>
      </div>
    )
  }

  return (
    <div className="container">
      {/* Hero Section */}
      <div className="text-center mb-4">
        <h1 className="mb-2">🎵 Twitch Spotify Request Bot</h1>
        <p className="text-secondary" style={{ fontSize: '1.25rem' }}>
          Let your viewers request songs and control your Spotify queue!
        </p>
      </div>

      {/* Features */}
      <div className="grid md:grid-cols-3 gap-3 mb-4">
        <div className="card text-center">
          <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>🎵</div>
          <h3 className="mb-2">Song Requests</h3>
          <p className="text-secondary">
            Let viewers request songs using Twitch channel points or chat commands.
          </p>
        </div>
        
        <div className="card text-center">
          <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>⏭️</div>
          <h3 className="mb-2">Skip Control</h3>
          <p className="text-secondary">
            Allow trusted users to skip songs with special rewards or commands.
          </p>
        </div>
        
        <div className="card text-center">
          <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>📋</div>
          <h3 className="mb-2">Queue Display</h3>
          <p className="text-secondary">
            Show your current queue to viewers with a public page.
          </p>
        </div>
      </div>

      {/* Active Streamers */}
      <div className="text-center mb-3">
        <h2 className="mb-3">Active Streamers</h2>
        
        {streamers && streamers.length > 0 ? (
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-2">
            {streamers.map(streamer => (
              <div key={streamer.id || streamer.name} className="card text-center">
                <h3 className="mb-2">{streamer.name || 'Unknown Streamer'}</h3>
                <Link 
                  to={`/queue/${streamer.id}`} 
                  className="btn btn-primary"
                >
                  View Queue
                </Link>
              </div>
            ))}
          </div>
        ) : (
          <div className="card text-center">
            <p className="text-muted">No active streamers found.</p>
            <p className="text-secondary">
              Streamers need to connect both Twitch and Spotify to appear here.
            </p>
          </div>
        )}
      </div>

      {/* Getting Started */}
      <div className="card text-center">
        <h2 className="mb-2">Get Started</h2>
        <p className="text-secondary mb-3">
          Ready to set up song requests for your stream?
        </p>
        <Link to="/dashboard" className="btn btn-primary btn-large">
          Go to Dashboard
        </Link>
      </div>
    </div>
  )
}

export default Home

import React, { useState, useEffect, useRef } from 'react'
import { useParams } from 'react-router-dom'
import QueueItem from '../components/QueueItem'
import LoadingSpinner from '../components/LoadingSpinner'
import { QueueData, Track } from '../types'
import axios from 'axios'

const QueueCompact: React.FC = () => {
  const { streamerId } = useParams<{ streamerId: string }>()
  const [queueData, setQueueData] = useState<QueueData | null>(null)
  const [currentTrack, setCurrentTrack] = useState<Track | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [currentProgress, setCurrentProgress] = useState<number>(0)
  const progressInterval = useRef<number | null>(null)

  const startProgressAnimation = () => {
    stopProgressAnimation()
    
    if (!queueData || queueData.duration <= 0) return

    progressInterval.current = window.setInterval(() => {
      setCurrentProgress(prev => {
        const newProgress = prev + 1000 // Add 1 second (1000ms)
        
        // If song is finished, refresh queue
        if (newProgress >= queueData.duration) {
          fetchQueueData(false)
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

  const fetchQueueData = async (showLoading: boolean = true) => {
    if (!streamerId) return
    
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
      
      if (response.data && response.data.success) {
        const newQueueData = response.data.data
        setQueueData(newQueueData)
        
        // Set current track with progress
        if (newQueueData.current_song) {
          const track: Track = {
            name: newQueueData.current_song,
            artists: newQueueData.current_song_artists || [],
            duration: newQueueData.duration,
            progress: newQueueData.progress,
            image: newQueueData.current_song_image
          }
          setCurrentTrack(track)
          
          // Update current progress if track changed
          if (queueData?.current_song !== newQueueData?.current_song) {
            setCurrentProgress(newQueueData?.progress || 0)
          }
        } else {
          setCurrentTrack(null)
        }
        
        setError(null)
      } else {
        const errorMsg = response.data?.error || 'Failed to load queue'
        setError(errorMsg)
      }
    } catch (err: any) {
      console.error('Error loading queue:', err)
      const errorMsg = err.response?.data?.error || err.message || 'Failed to load queue'
      setError(errorMsg)
    } finally {
      if (showLoading) {
        setLoading(false)
      }
    }
  }

  useEffect(() => {
    fetchQueueData()
    // Refresh every 30 seconds
    const interval = setInterval(() => fetchQueueData(false), 30000)
    return () => clearInterval(interval)
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

  if (loading) {
    return (
      <div className="queue-compact-page">
        <div className="compact-loading">
          <LoadingSpinner message="Loading queue..." />
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="queue-compact-page">
        <div className="compact-error">
          <p>Error: {error}</p>
        </div>
      </div>
    )
  }

  if (!currentTrack) {
    return (
      <div className="queue-compact-page">
        <div className="compact-no-track">
          <div className="no-track-message">No track currently playing</div>
        </div>
      </div>
    )
  }

  // Use real-time progress for better accuracy
  const trackWithProgress = {
    ...currentTrack,
    progress: currentProgress
  }

  return (
    <div className="queue-compact-page">
      <QueueItem 
        track={trackWithProgress} 
        isCurrentlyPlaying={true} 
        compact={true} 
      />
    </div>
  )
}

export default QueueCompact

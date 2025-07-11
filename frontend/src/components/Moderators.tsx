import React, { useState, useEffect, useRef } from 'react'
import axios from 'axios'
import LoadingSpinner from './LoadingSpinner'
import { useToast } from '../contexts/ToastContext'

interface Moderator {
  id: number
  twitch_id: string
  twitch_name: string
  avatar: string
  added_at: string
}

interface TwitchUser {
  id: string
  login: string
  display_name: string
  avatar: string
}

interface ModeratorsProps {
  userId: string
}

const Moderators: React.FC<ModeratorsProps> = ({ userId }) => {
  const [moderators, setModerators] = useState<Moderator[]>([])
  const [loading, setLoading] = useState<boolean>(true)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState<string>('')
  const [searchResults, setSearchResults] = useState<TwitchUser[]>([])
  const [searching, setSearching] = useState<boolean>(false)
  const [showDropdown, setShowDropdown] = useState<boolean>(false)
  const [adding, setAdding] = useState<boolean>(false)
  const [showSearch, setShowSearch] = useState<boolean>(false)
  const { showSuccess, showError } = useToast()
  const timeoutRef = useRef<number | null>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    loadModerators()
  }, [userId])

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowDropdown(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  const loadModerators = async () => {
    try {
      setLoading(true)
      setError(null)
      
      const response = await axios.get(`/api/user/${userId}/moderators`)
      
      if (response.data?.success) {
        setModerators(response.data.data || [])
      }
    } catch (err: any) {
      console.error('Error loading moderators:', err)
      setError(err.response?.data?.error || 'Failed to load moderators')
    } finally {
      setLoading(false)
    }
  }

  const searchUsers = async (query: string) => {
    if (query.trim().length < 2) {
      setSearchResults([])
      setShowDropdown(false)
      return
    }

    try {
      setSearching(true)
      
      const response = await axios.get(`/api/user/${userId}/twitch/search?q=${encodeURIComponent(query)}`)
      
      if (response.data?.success) {
        const results = response.data.data || []
        setSearchResults(results)
        setShowDropdown(results.length > 0)
      } else {
        setSearchResults([])
        setShowDropdown(false)
      }
    } catch (err: any) {
      console.error('Error searching users:', err)
      setSearchResults([])
      setShowDropdown(false)
    } finally {
      setSearching(false)
    }
  }

  const handleSearchInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const query = e.target.value
    setSearchQuery(query)
    
    // Clear existing timeout
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }
    
    // Debounce search
    timeoutRef.current = setTimeout(() => {
      searchUsers(query)
    }, 300)
  }

  const addModerator = async (twitchName: string) => {
    try {
      setAdding(true)
      
      const response = await axios.post(`/api/user/${userId}/moderators`, {
        twitch_name: twitchName
      })
      
      if (response.data?.success) {
        showSuccess('Moderator added successfully')
        setSearchQuery('')
        setSearchResults([])
        setShowDropdown(false)
        setShowSearch(false)
        await loadModerators()
      }
    } catch (err: any) {
      console.error('Error adding moderator:', err)
      showError(err.response?.data?.error || 'Failed to add moderator')
    } finally {
      setAdding(false)
    }
  }

  const removeModerator = async (moderatorId: number) => {
    if (!window.confirm('Are you sure you want to remove this moderator?')) {
      return
    }

    try {
      const response = await axios.delete(`/api/user/${userId}/moderators/${moderatorId}`)
      
      if (response.data?.success) {
        showSuccess('Moderator removed successfully')
        await loadModerators()
      }
    } catch (err: any) {
      console.error('Error removing moderator:', err)
      showError(err.response?.data?.error || 'Failed to remove moderator')
    }
  }

  if (loading) {
    return <LoadingSpinner message="Loading moderators..." />
  }

  if (error) {
    return (
      <div className="alert alert-error">
        {error}
        <button onClick={loadModerators} className="btn btn-secondary btn-small ml-2">
          Try Again
        </button>
      </div>
    )
  }

  return (
    <div className="p-3" style={{ backgroundColor: 'var(--twitch-bg-medium)', borderRadius: '8px', border: '1px solid var(--twitch-border)' }}>
      <div className="flex justify-between align-center mb-3">
        <h2 className="text-lg font-semibold text-primary">Bot Moderators</h2>
        <button 
          onClick={() => setShowSearch(!showSearch)}
          className="btn btn-primary btn-small"
        >
          {showSearch ? 'Cancel' : '+ Add Moderator'}
        </button>
      </div>

      <p className="text-muted mb-3 text-sm">
        Bot moderators can use special commands like !volume. They are separate from Twitch channel moderators.
      </p>

      {/* Add Moderator Section */}
      {showSearch && (
        <div className="mb-4 p-3 rounded" style={{ backgroundColor: 'var(--twitch-bg-light)', border: '1px solid var(--twitch-border)' }}>
          <h3 className="mb-3 text-sm font-semibold text-secondary">Add New Moderator</h3>
          
          <div className="search-input-container" ref={dropdownRef}>
            <input
              type="text"
              className="form-input search-input"
              placeholder="Search Twitch username..."
              value={searchQuery}
              onChange={handleSearchInputChange}
              disabled={adding}
              autoComplete="off"
            />
            
            {searchQuery && (
              <button
                type="button"
                onClick={() => {
                  setSearchQuery('')
                  setSearchResults([])
                  setShowDropdown(false)
                }}
                className="clear-button"
                disabled={adding}
              >
                ✕
              </button>
            )}
            
            {searching && (
              <div className="search-loading">
                <div className="spinner"></div>
              </div>
            )}

            {showDropdown && searchResults.length > 0 && (
              <div className="search-dropdown">
                {searchResults.map((user) => (
                  <div 
                    key={user.id} 
                    className="search-result"
                    onClick={() => addModerator(user.login)}
                  >
                    <div className="result-content">
                      <img 
                        src={user.avatar} 
                        alt={user.display_name}
                        className="result-image"
                      />
                      <div className="result-text">
                        <div className="result-name">{user.display_name}</div>
                        <div className="result-info">@{user.login}</div>
                      </div>
                      <div className="result-type" style={{ color: 'var(--twitch-purple)' }}>
                        {moderators.some(m => m.twitch_id === user.id) ? '✓' : '+'}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {searchQuery.trim().length >= 2 && !searching && searchResults.length === 0 && (
              <div className="search-error">
                No users found for "{searchQuery}". Make sure the username is spelled correctly.
              </div>
            )}
          </div>
        </div>
      )}

      {/* Moderators List */}
      {moderators.length > 0 ? (
        <div>
          <h3 className="mb-3 text-sm font-semibold text-secondary">Current Moderators ({moderators.length})</h3>
          <div className="space-y-2">
            {moderators.map((moderator) => (
              <div key={moderator.id} className="flex align-center justify-between p-2 rounded" style={{ backgroundColor: 'var(--twitch-bg-light)', border: '1px solid var(--twitch-border)' }}>
                <div className="flex align-center">
                  <img 
                    src={moderator.avatar} 
                    alt={moderator.twitch_name}
                    className="w-8 h-8 rounded-full mr-2"
                  />
                  <div>
                    <div className="text-sm font-medium text-primary">{moderator.twitch_name}</div>
                    <div className="text-xs text-muted">Added: {new Date(moderator.added_at).toLocaleDateString()}</div>
                  </div>
                </div>
                <button
                  onClick={() => removeModerator(moderator.id)}
                  className="btn btn-danger btn-small"
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        </div>
      ) : (
        <div className="text-center py-4 text-muted">
          <p className="text-sm">No moderators added yet.</p>
          <p className="text-xs">Add moderators to allow them to use special bot commands.</p>
        </div>
      )}
    </div>
  )
}

export default Moderators

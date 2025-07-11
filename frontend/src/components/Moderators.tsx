import React, { useState, useEffect, useRef } from 'react'
import axios from 'axios'
import { useToast } from '../contexts/ToastContext'
import { Button, Card, SearchableDropdown, Avatar, Spinner, ListItem } from './ui'

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
    return <Spinner size="large" />
  }

  if (error) {
    return (
      <Card>
        <div className="alert alert-error">
          {error}
          <Button onClick={loadModerators} variant="secondary" size="small">
            Try Again
          </Button>
        </div>
      </Card>
    )
  }

  return (
    <Card>
      <div className="flex justify-between align-center mb-3">
        <h2 className="text-lg font-semibold text-primary">Bot Moderators</h2>
        <Button 
          onClick={() => setShowSearch(!showSearch)}
          variant="primary" 
          size="small"
        >
          {showSearch ? 'Cancel' : '+ Add Moderator'}
        </Button>
      </div>

      <p className="text-muted mb-3 text-sm">
        Bot moderators can use special commands like !volume. They are separate from Twitch channel moderators.
      </p>

      {/* Add Moderator Section */}
      {showSearch && (
        <Card variant="flat" className="mb-4">
          <h3 className="mb-3 text-sm font-semibold text-secondary">Add New Moderator</h3>
          
          <SearchableDropdown
            ref={dropdownRef}
            placeholder="Search Twitch username..."
            value={searchQuery}
            onChange={handleSearchInputChange}
            onClear={() => {
              setSearchQuery('')
              setSearchResults([])
              setShowDropdown(false)
            }}
            loading={searching}
            disabled={adding}
            showDropdown={showDropdown}
            error={searchQuery.trim().length >= 2 && !searching && searchResults.length === 0 ? 
              `No users found for "${searchQuery}". Make sure the username is spelled correctly.` : undefined}
          >
            {searchResults.map((user) => (
              <ListItem
                key={user.id}
                onClick={() => addModerator(user.login)}
                leftContent={<Avatar src={user.avatar} alt={user.display_name} size="medium" />}
                clickable
              >
                <div className="result-text">
                  <div className="result-name">{user.display_name}</div>
                  <div className="result-info">@{user.login}</div>
                </div>
              </ListItem>
            ))}
          </SearchableDropdown>
        </Card>
      )}

      {/* Moderators List */}
      {moderators.length > 0 ? (
        <div>
          <h3 className="mb-3 text-sm font-semibold text-secondary">Current Moderators ({moderators.length})</h3>
          <div className="space-y-2">
            {moderators.map((moderator) => (
              <ListItem
                key={moderator.id}
                left={<Avatar src={moderator.avatar} alt={moderator.twitch_name} size="medium" />}
                right={
                  <Button
                    onClick={() => removeModerator(moderator.id)}
                    variant="danger"
                    size="small"
                  >
                    Remove
                  </Button>
                }
              >
                <div>
                  <div className="text-sm font-medium text-primary">{moderator.twitch_name}</div>
                  <div className="text-xs text-muted">Added: {new Date(moderator.added_at).toLocaleDateString()}</div>
                </div>
              </ListItem>
            ))}
          </div>
        </div>
      ) : (
        <div className="text-center py-4 text-muted">
          <p className="text-sm">No moderators added yet.</p>
          <p className="text-xs">Add moderators to allow them to use special bot commands.</p>
        </div>
      )}
    </Card>
  )
}

export default Moderators

import React, { useState, useEffect, useRef } from 'react'
import { SpotifySearchResult } from '../types'
import axios from 'axios'

interface SpotifyAutocompleteProps {
  onSelect: (result: SpotifySearchResult) => void;
  searchType: 'artist' | 'track' | 'both';
  placeholder?: string;
  userId: string;
  disabled?: boolean;
  className?: string;
}

const SpotifyAutocomplete: React.FC<SpotifyAutocompleteProps> = ({
  onSelect,
  searchType,
  placeholder = 'Search...',
  userId,
  disabled = false,
  className = ''
}) => {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SpotifySearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [showDropdown, setShowDropdown] = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(-1)
  const [error, setError] = useState<string | null>(null)
  
  const inputRef = useRef<HTMLInputElement>(null)
  const timeoutRef = useRef<number | null>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowDropdown(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const searchSpotify = async (searchQuery: string) => {
    if (!searchQuery.trim()) {
      setResults([])
      setShowDropdown(false)
      return
    }

    try {
      setLoading(true)
      setError(null)
      
      const type = searchType === 'both' ? 'artist,track' : searchType
      const response = await axios.get(`/api/user/${userId}/spotify/search`, {
        params: {
          q: searchQuery,
          type: type,
          limit: 10
        }
      })

      if (response.data?.success) {
        const searchResults = response.data.data?.results || []
        setResults(searchResults)
        setShowDropdown(searchResults.length > 0)
        setSelectedIndex(-1)
      } else {
        setError(response.data?.error || 'Search failed')
        setResults([])
        setShowDropdown(false)
      }
    } catch (err: any) {
      console.error('Spotify search error:', err)
      setError(err.response?.data?.error || 'Search failed')
      setResults([])
      setShowDropdown(false)
    } finally {
      setLoading(false)
    }
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value
    setQuery(value)

    // Clear existing timeout
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }

    // Debounce search
    timeoutRef.current = setTimeout(() => {
      searchSpotify(value)
    }, 300)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!showDropdown) return

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        setSelectedIndex(prev => 
          prev < results.length - 1 ? prev + 1 : prev
        )
        break
      case 'ArrowUp':
        e.preventDefault()
        setSelectedIndex(prev => prev > 0 ? prev - 1 : -1)
        break
      case 'Enter':
        e.preventDefault()
        if (selectedIndex >= 0 && selectedIndex < results.length) {
          handleSelect(results[selectedIndex])
        }
        break
      case 'Escape':
        setShowDropdown(false)
        setSelectedIndex(-1)
        break
    }
  }

  const handleSelect = (result: SpotifySearchResult) => {
    setQuery(result.name)
    setShowDropdown(false)
    setSelectedIndex(-1)
    onSelect(result)
  }

  const clearInput = () => {
    setQuery('')
    setResults([])
    setShowDropdown(false)
    setSelectedIndex(-1)
    if (inputRef.current) {
      inputRef.current.focus()
    }
  }

  const formatArtistInfo = (result: SpotifySearchResult) => {
    if (result.type === 'track' && result.artists) {
      return result.artists.join(', ')
    }
    return result.type === 'artist' ? 'Artist' : 'Track'
  }

  return (
    <div className={`spotify-autocomplete ${className}`} ref={dropdownRef}>
      <div className="search-input-container">
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          className="form-input search-input"
          disabled={disabled}
          autoComplete="off"
        />
        
        {query && (
          <button
            type="button"
            onClick={clearInput}
            className="clear-button"
            disabled={disabled}
          >
            ✕
          </button>
        )}
        
        {loading && (
          <div className="search-loading">
            <div className="spinner"></div>
          </div>
        )}
      </div>

      {error && (
        <div className="search-error">
          {error}
        </div>
      )}

      {showDropdown && results.length > 0 && (
        <div className="search-dropdown">
          {results.map((result, index) => (
            <div
              key={`${result.type}-${result.id}`}
              className={`search-result ${index === selectedIndex ? 'selected' : ''}`}
              onClick={() => handleSelect(result)}
              onMouseEnter={() => setSelectedIndex(index)}
            >
              <div className="result-content">
                {result.image && (
                  <img
                    src={result.image}
                    alt=""
                    className="result-image"
                  />
                )}
                <div className="result-text">
                  <div className="result-name">{result.name}</div>
                  <div className="result-info">
                    {formatArtistInfo(result)}
                  </div>
                </div>
                <div className={`result-type result-type-${result.type}`}>
                  {result.type === 'artist' ? '🎤' : '🎵'}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default SpotifyAutocomplete

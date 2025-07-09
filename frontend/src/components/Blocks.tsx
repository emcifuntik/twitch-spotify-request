import React, { useState, useEffect } from 'react'
import { Block, BlockRequest, SpotifySearchResult } from '../types'
import LoadingSpinner from './LoadingSpinner'
import SpotifyAutocomplete from './SpotifyAutocomplete'
import ToggleSwitch from './ToggleSwitch'
import { useToast } from '../contexts/ToastContext'
import axios from 'axios'

interface BlocksProps {
  userId: string;
}

const Blocks: React.FC<BlocksProps> = ({ userId }) => {
  const [blocks, setBlocks] = useState<Block[]>([])
  const [loading, setLoading] = useState<boolean>(true)
  const [adding, setAdding] = useState<boolean>(false)
  const [error, setError] = useState<string | null>(null)
  const [newBlock, setNewBlock] = useState<BlockRequest>({
    spotify_id: '',
    name: '',
    type: 'artist'
  })
  const [selectedResult, setSelectedResult] = useState<SpotifySearchResult | null>(null)
  const { showSuccess, showError } = useToast()

  useEffect(() => {
    loadBlocks()
  }, [userId])

  const loadBlocks = async (): Promise<void> => {
    try {
      setLoading(true)
      setError(null)
      const response = await axios.get(`/api/user/${userId}/blocks`)
      
      if (response.data?.success) {
        setBlocks(response.data.data || [])
      } else {
        setError(response.data?.error || 'Failed to load blocks')
      }
    } catch (err: any) {
      setError(err.response?.data?.error || err.message || 'Failed to load blocks')
    } finally {
      setLoading(false)
    }
  }

  const handleSpotifySelect = (result: SpotifySearchResult): void => {
    setSelectedResult(result)
    setNewBlock({
      spotify_id: result.id,
      name: result.name,
      type: result.type as 'artist' | 'track'
    })
    setError(null)
  }

  const addBlock = async (): Promise<void> => {
    if (!newBlock.spotify_id || !newBlock.name) {
      setError('Please select an artist or track from the search results')
      return
    }

    try {
      setAdding(true)
      setError(null)
      
      const response = await axios.post(`/api/user/${userId}/blocks`, newBlock)
      
      if (response.data?.success) {
        showSuccess('Block added successfully!')
        setNewBlock({ spotify_id: '', name: '', type: 'artist' })
        setSelectedResult(null)
        loadBlocks()
      } else {
        const errorMsg = response.data?.error || 'Failed to add block'
        setError(errorMsg)
        showError(errorMsg)
      }
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || err.message || 'Failed to add block'
      setError(errorMsg)
      showError(errorMsg)
    } finally {
      setAdding(false)
    }
  }

  const removeBlock = async (blockId: number): Promise<void> => {
    try {
      setError(null)
      
      const response = await axios.delete(`/api/user/${userId}/blocks/${blockId}`)
      
      if (response.data?.success) {
        showSuccess('Block removed successfully!')
        loadBlocks()
      } else {
        const errorMsg = response.data?.error || 'Failed to remove block'
        setError(errorMsg)
        showError(errorMsg)
      }
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || err.message || 'Failed to remove block'
      setError(errorMsg)
      showError(errorMsg)
    }
  }

  const handleSubmit = (e: React.FormEvent): void => {
    e.preventDefault()
    addBlock()
  }

  const formatArtistNames = (block: Block): string => {
    if (block.type === 'track') {
      // For tracks, we'll show the name as is since it should include artist info
      return block.name
    }
    return block.name
  }

  if (loading) {
    return <LoadingSpinner message="Loading blocks..." />
  }

  return (
    <div className="blocks-section">
      <h3>Blocked Artists & Tracks</h3>

      {error && (
        <div className="alert alert-error">
          {error}
        </div>
      )}

      <div className="add-block-form">
        <h4>Add New Block</h4>
        <form onSubmit={handleSubmit} className="flex gap-2">
          <ToggleSwitch
            leftLabel="Artist"
            rightLabel="Track"
            value={newBlock.type === 'track'}
            onChange={(isTrack) => {
              const newType = isTrack ? 'track' : 'artist'
              setNewBlock(prev => ({ ...prev, type: newType }))
              setSelectedResult(null)
            }}
            disabled={adding}
          />
          
          <div className="flex-1">
            <SpotifyAutocomplete
              onSelect={handleSpotifySelect}
              searchType={newBlock.type}
              placeholder={`Search for ${newBlock.type}...`}
              userId={userId}
              disabled={adding}
            />
          </div>
          
          <button
            type="submit"
            className="btn btn-primary"
            disabled={adding || !selectedResult}
          >
            {adding ? 'Adding...' : 'Add Block'}
          </button>
        </form>
        
        {selectedResult && (
          <div className="selected-result">
            <small className="text-muted">
              Selected: <strong>{selectedResult.name}</strong>
              {selectedResult.type === 'track' && selectedResult.artists && (
                <span> by {selectedResult.artists.join(', ')}</span>
              )}
            </small>
          </div>
        )}
        
        <small className="text-muted">
          Search and select artists or tracks from Spotify to block them from requests
        </small>
      </div>

      <div className="blocks-list">
        <h4>Current Blocks ({blocks.length})</h4>
        
        {blocks.length === 0 ? (
          <div className="text-center text-muted">
            No blocks configured. Add some artists or tracks to block above.
          </div>
        ) : (
          <div className="blocks-grid">
            {blocks.map((block) => (
              <div key={block.id} className="block-item">
                <div className="block-info">
                  <span className={`block-type block-type-${block.type}`}>
                    {block.type === 'artist' ? '🎤' : '🎵'}
                  </span>
                  <div className="block-details">
                    <div className="block-target">{formatArtistNames(block)}</div>
                    <div className="block-type-label">{block.type}</div>
                    <div className="block-spotify-id">ID: {block.spotify_id}</div>
                  </div>
                </div>
                <button
                  onClick={() => removeBlock(block.id)}
                  className="btn btn-danger btn-small"
                  title="Remove block"
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default Blocks

import React, { useState, useEffect } from 'react'
import { Button, Card, Input, Spinner } from './ui'
import ToggleSwitch from './ToggleSwitch'
import { useToast } from '../contexts/ToastContext'
import { Command, CommandRequest, RequestModeToggle, APIResponse } from '../types'
import axios from 'axios'

interface CommandsProps {
  userId: string
}

const Commands: React.FC<CommandsProps> = ({ userId }) => {
  const [commands, setCommands] = useState<Command[]>([])
  const [loading, setLoading] = useState<boolean>(true)
  const [saving, setSaving] = useState<boolean>(false)
  const [useCommands, setUseCommands] = useState<boolean>(false)
  const [editingCommand, setEditingCommand] = useState<number | null>(null)
  const [tempCommandName, setTempCommandName] = useState<string>('')
  const { showSuccess, showError } = useToast()

  useEffect(() => {
    loadCommands()
  }, [userId])

  const loadCommands = async () => {
    try {
      setLoading(true)
      const response = await axios.get<APIResponse<{ commands: Command[]; use_commands: boolean }>>(`/api/user/${userId}/commands`)
      
      if (response.data.success && response.data.data) {
        setCommands(response.data.data.commands)
        setUseCommands(response.data.data.use_commands)
      } else {
        showError('Failed to load commands')
      }
    } catch (error) {
      console.error('Error loading commands:', error)
      showError('Error loading commands')
    } finally {
      setLoading(false)
    }
  }

  const initializeCommands = async () => {
    try {
      setSaving(true)
      const response = await axios.post<APIResponse>(`/api/user/${userId}/commands/initialize`)
      
      if (response.data.success) {
        showSuccess('Commands initialized successfully')
        loadCommands()
      } else {
        showError('Failed to initialize commands')
      }
    } catch (error) {
      console.error('Error initializing commands:', error)
      showError('Error initializing commands')
    } finally {
      setSaving(false)
    }
  }

  const toggleRequestMode = async (newUseCommands: boolean) => {
    try {
      setSaving(true)
      const requestData: RequestModeToggle = { use_commands: newUseCommands }
      const response = await axios.put<APIResponse>(`/api/user/${userId}/request-mode`, requestData)
      
      if (response.data.success) {
        setUseCommands(newUseCommands)
        showSuccess(`Switched to ${newUseCommands ? 'commands' : 'rewards'} mode`)
      } else {
        showError('Failed to toggle request mode')
      }
    } catch (error) {
      console.error('Error toggling request mode:', error)
      showError('Error toggling request mode')
    } finally {
      setSaving(false)
    }
  }

  const updateCommand = async (command: Command) => {
    try {
      setSaving(true)
      const requestData: CommandRequest = {
        type: command.type,
        name: command.name,
        is_enabled: command.is_enabled
      }
      const response = await axios.put<APIResponse>(`/api/user/${userId}/commands`, requestData)
      
      if (response.data.success) {
        showSuccess('Command updated successfully')
        loadCommands()
      } else {
        showError('Failed to update command')
      }
    } catch (error) {
      console.error('Error updating command:', error)
      showError('Error updating command')
    } finally {
      setSaving(false)
    }
  }

  const handleCommandNameEdit = (commandId: number, currentName: string) => {
    setEditingCommand(commandId)
    setTempCommandName(currentName)
  }

  const handleCommandNameSave = async (command: Command) => {
    if (tempCommandName.trim() === '') {
      showError('Command name cannot be empty')
      return
    }

    const updatedCommand = { ...command, name: tempCommandName.trim() }
    await updateCommand(updatedCommand)
    setEditingCommand(null)
    setTempCommandName('')
  }

  const handleCommandNameCancel = () => {
    setEditingCommand(null)
    setTempCommandName('')
  }

  const handleToggleCommand = async (command: Command) => {
    const updatedCommand = { ...command, is_enabled: !command.is_enabled }
    await updateCommand(updatedCommand)
  }

  if (loading) {
    return (
      <div className="commands-container">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="commands-container">
      <Card>
        <h2>Song Request Mode</h2>
        <p>Choose how viewers can request songs:</p>
        
        <div className="request-mode-toggle">
          <ToggleSwitch
            leftLabel="Rewards Mode"
            rightLabel="Commands Mode"
            value={useCommands}
            onChange={toggleRequestMode}
            disabled={saving}
          />
          <p className="mode-description">
            {useCommands 
              ? 'Viewers use chat commands to request songs' 
              : 'Viewers use channel point rewards to request songs'
            }
          </p>
        </div>

        {!useCommands && (
          <div className="rewards-info">
            <p>⚠️ Rewards mode is only available for Twitch Partners and Affiliates.</p>
          </div>
        )}
      </Card>

      {useCommands && (
        <Card>
          <h2>Chat Commands</h2>
          <p>Customize the chat commands for your channel:</p>
          
          {commands.length === 0 ? (
            <div className="no-commands">
              <p>No commands found. Initialize default commands to get started.</p>
              <Button 
                onClick={initializeCommands}
                disabled={saving}
                variant="primary"
              >
                {saving ? 'Initializing...' : 'Initialize Commands'}
              </Button>
            </div>
          ) : (
            <div className="commands-list">
              {commands.map((command) => (
                <div key={command.id} className="command-item">
                  <div className="command-info">
                    <div className="command-type">
                      <strong>{command.type}</strong>
                    </div>
                    <div className="command-name">
                      {editingCommand === command.id ? (
                        <div className="edit-command">
                          <Input
                            value={tempCommandName}
                            onChange={(e) => setTempCommandName(e.target.value)}
                            placeholder="Command name"
                          />
                          <div className="edit-buttons">
                            <Button 
                              onClick={() => handleCommandNameSave(command)}
                              disabled={saving}
                              variant="primary"
                              size="small"
                            >
                              Save
                            </Button>
                            <Button 
                              onClick={handleCommandNameCancel}
                              disabled={saving}
                              variant="secondary"
                              size="small"
                            >
                              Cancel
                            </Button>
                          </div>
                        </div>
                      ) : (
                        <div className="command-display">
                          <span className="command-prefix">!</span>
                          <span className="command-text">{command.name}</span>
                          <Button 
                            onClick={() => handleCommandNameEdit(command.id, command.name)}
                            variant="secondary"
                            size="small"
                          >
                            Edit
                          </Button>
                        </div>
                      )}
                    </div>
                  </div>
                  <div className="command-actions">
                    <ToggleSwitch
                      leftLabel="Disabled"
                      rightLabel="Enabled"
                      value={command.is_enabled}
                      onChange={() => handleToggleCommand(command)}
                      disabled={saving}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}

          <div className="commands-actions">
            <Button 
              onClick={initializeCommands}
              disabled={saving}
              variant="secondary"
            >
              {saving ? 'Resetting...' : 'Reset to Defaults'}
            </Button>
          </div>
        </Card>
      )}
    </div>
  )
}

export default Commands

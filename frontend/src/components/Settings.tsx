import React, { useState, useEffect } from 'react'
import { StreamerSettings } from '../types'
import { Button, Card, Spinner } from './ui'
import AppleTimePicker from './AppleTimePicker'
import { useToast } from '../contexts/ToastContext'
import axios from 'axios'

interface SettingsProps {
  userId: string;
}

const Settings: React.FC<SettingsProps> = ({ userId }) => {
  const [settings, setSettings] = useState<StreamerSettings | null>(null)
  const [loading, setLoading] = useState<boolean>(true)
  const [saving, setSaving] = useState<boolean>(false)
  const [error, setError] = useState<string | null>(null)
  const [tempSettings, setTempSettings] = useState<StreamerSettings | null>(null)
  const { showSuccess, showError } = useToast()

  useEffect(() => {
    loadSettings()
  }, [userId])

  const loadSettings = async (): Promise<void> => {
    try {
      setLoading(true)
      setError(null)
      const response = await axios.get(`/api/user/${userId}/config`)
      
      if (response.data?.success) {
        setSettings(response.data.data)
      } else {
        setError(response.data?.error || 'Failed to load settings')
      }
    } catch (err: any) {
      setError(err.response?.data?.error || err.message || 'Failed to load settings')
    } finally {
      setLoading(false)
    }
  }

  const updateSettings = async (updatedSettings: Partial<StreamerSettings>): Promise<void> => {
    try {
      setSaving(true)
      setError(null)
      
      const response = await axios.post(`/api/user/${userId}/config`, updatedSettings)
      
      if (response.data?.success) {
        setSettings(response.data.data)
        showSuccess('Settings updated successfully!')
      } else {
        const errorMsg = response.data?.error || 'Failed to update settings'
        setError(errorMsg)
        showError(errorMsg)
      }
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || err.message || 'Failed to update settings'
      setError(errorMsg)
      showError(errorMsg)
    } finally {
      setSaving(false)
    }
  }

  const handleMaxLengthChange = (seconds: number): void => {
    if (settings && seconds >= 30 && seconds <= 86399) { // Max 23:59:59
      updateSettings({ max_song_length: seconds })
    }
  }

  const handleMaxLengthImmediate = (seconds: number): void => {
    if (settings && seconds >= 30 && seconds <= 86399) {
      setTempSettings({ ...settings, max_song_length: seconds })
    }
  }

  const handleCooldownChange = (seconds: number): void => {
    if (settings && seconds >= 0 && seconds <= 86399) { // Max 23:59:59
      updateSettings({ cooldown_same_song: seconds })
    }
  }

  const handleCooldownImmediate = (seconds: number): void => {
    if (settings && seconds >= 0 && seconds <= 86399) {
      setTempSettings({ ...settings, cooldown_same_song: seconds })
    }
  }

  const handleWebUIToggle = (): void => {
    if (settings) {
      updateSettings({ web_ui_enabled: !settings.web_ui_enabled })
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
          <Button onClick={loadSettings} variant="secondary" size="small" style={{ marginLeft: '1rem' }}>
            Retry
          </Button>
        </div>
      </Card>
    )
  }

  if (!settings) {
    return <Card>No settings found</Card>
  }

  return (
    <Card>
      <h3>Bot Settings</h3>

      <div className="settings-grid">
        <div className="setting-item">
          <label htmlFor="maxLength">Max Song Length</label>
          <AppleTimePicker
            id="maxLength"
            value={(tempSettings || settings).max_song_length}
            onChange={handleMaxLengthChange}
            onChangeImmediate={handleMaxLengthImmediate}
            maxHours={23}
            maxMinutes={59}
            maxSeconds={59}
            minSeconds={30}
            disabled={saving}
            className="setting-time-picker"
            debounceMs={3000}
          />
          <small className="text-muted">Minimum 30 seconds, maximum 23:59:59</small>
        </div>

        <div className="setting-item">
          <label htmlFor="cooldown">Cooldown for Same Song</label>
          <AppleTimePicker
            id="cooldown"
            value={(tempSettings || settings).cooldown_same_song}
            onChange={handleCooldownChange}
            onChangeImmediate={handleCooldownImmediate}
            maxHours={23}
            maxMinutes={59}
            maxSeconds={59}
            minSeconds={0}
            disabled={saving}
            className="setting-time-picker"
            debounceMs={3000}
          />
          <small className="text-muted">0 seconds to 23:59:59 (0 = no cooldown)</small>
        </div>

        <div className="setting-item">
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={settings.web_ui_enabled}
              onChange={handleWebUIToggle}
              disabled={saving}
            />
            <span className="checkmark"></span>
            Enable Web UI
          </label>
          <small className="text-muted">Allow public access to queue page</small>
        </div>
      </div>

      {saving && <Spinner size="medium" />}
    </Card>
  )
}

export default Settings

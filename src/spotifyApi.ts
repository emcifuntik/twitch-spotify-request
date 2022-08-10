import got from 'got';
import { EventEmitter } from 'node:events';
import { Streamer } from './db/models/streamer';

export class SpotifyAPI extends EventEmitter {
  private token: string;
  private refreshToken: string;
  private static API_HOSTNAME: string = 'https://api.spotify.com/v1/';
  constructor(token: string, refreshToken: string) {
    super();
    this.token = token;
    this.refreshToken = refreshToken;
  }

  private async refreshTokens() {
    const refreshTokenResult = await this.httpClient.post('https://accounts.spotify.com/api/token', {
      headers: { 'Authorization': 'Basic ' + (Buffer.from(process.env.SPOTIFY_CLIENT_ID + ':' + process.env.SPOTIFY_CLIENT_SECRET).toString('base64')) },
      form: {
        grant_type: 'refresh_token',
        refresh_token: this.refreshToken
      },
      responseType: 'json'
    });

    const responseBody = refreshTokenResult.body as Record<string, any>;
    this.token = responseBody.access_token;
    this.emit('tokenUpdated', responseBody.access_token)
  }

  private get httpClient() {
    return got.extend({
      throwHttpErrors: false,
      responseType: 'json',
      headers: {
        'Authorization': `Bearer ${this.token}`
      },
      hooks: {
        afterResponse: [
          async (response, retryWithMergedOptions) => {
            // Unauthorized
            if (response.statusCode === 401) {
              // Refresh the access token
              await this.refreshTokens();
              const updatedOptions = {
                headers: {
                  'Authorization': `Bearer ${this.token}`
                }
              };
              // Make a new retry
              return retryWithMergedOptions(updatedOptions);
            }
    
            // No changes otherwise
            return response;
          }
        ],
      }
    });
  }

  public async searchQuery(queryText: string): Promise<any> {
    const searchResult = await this.httpClient.get(SpotifyAPI.API_HOSTNAME + 'search', {
      searchParams: {
        q: queryText,
        type: 'track'
      }
    });

    return searchResult.body;
  }

  public async enqueueTrack(trackUri: string): Promise<any> {
    const enqueueResult = await this.httpClient.post(SpotifyAPI.API_HOSTNAME + 'me/player/queue', {
      json: {
        uri: trackUri
      }
    });

    return enqueueResult.body;
  }

  public async nextTrack(): Promise<any> {
    const skipResult = await this.httpClient.post(SpotifyAPI.API_HOSTNAME + 'me/player/next');
    return skipResult.body;
  }

  public async previousTrack(): Promise<any> {
    const prevTrackResult = await this.httpClient.post(SpotifyAPI.API_HOSTNAME + 'me/player/previous');
    return prevTrackResult.body;
  }

  public async getRecentlyPlayed(count: number): Promise<any> {
    const prevTracks = await this.httpClient.get(SpotifyAPI.API_HOSTNAME + 'me/player/recently-played', {
      searchParams: {limit: count}
    });
    return prevTracks.body;
  }

  public async getCurrentTrack(): Promise<any> {
    const currentTrack = await this.httpClient.get(SpotifyAPI.API_HOSTNAME + 'me/player/currently-playing');
    return currentTrack.body;
  }

  public async getTrackById(trackId: string): Promise<any> {
    const trackInfo = await this.httpClient.get(SpotifyAPI.API_HOSTNAME + 'tracks/' + trackId);
    return trackInfo.body;
  }

  public async setPlayerVolume(volume: number): Promise<any> {
    const volumeChangeResult = await this.httpClient.put(SpotifyAPI.API_HOSTNAME + 'me/player/volume', {
      json: { 
        volume_percent: volume.toFixed(0) 
      }
    });
    return volumeChangeResult.body;
  }
}
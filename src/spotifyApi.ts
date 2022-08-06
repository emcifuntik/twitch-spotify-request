import got, { Method, OptionsOfJSONResponseBody } from 'got';

export class SpotifyAPI {
  private token: string;
  private static API_HOSTNAME: string = 'https://api.spotify.com/v1/';
  constructor(token: string) {
    this.token = token;
  }

  private async httpRequestGet(route: string, query: Record<string, any> = {}) {
    const gotOptions: OptionsOfJSONResponseBody = {
      throwHttpErrors: false,
      method: 'GET',
      responseType: 'json',
      headers: {
        'Authorization': `Bearer ${this.token}`
      },
      searchParams: query
    };

    const response = await got(`${SpotifyAPI.API_HOSTNAME}${route}`, gotOptions);
    return response.body as Record<string, any>;
  }

  private async httpRequestPut(route: string, query: Record<string, any> = {}) {
    const gotOptions: OptionsOfJSONResponseBody = {
      throwHttpErrors: false,
      method: 'PUT',
      responseType: 'json',
      headers: {
        'Authorization': `Bearer ${this.token}`
      },
      searchParams: query
    };

    const response = await got(`${SpotifyAPI.API_HOSTNAME}${route}`, gotOptions);
    return response.body as Record<string, any>;
  }

  private async httpRequestPost(route: string, query: Record<string, any> = {}, payload: Record<string, any> = {}) {
    const gotOptions: OptionsOfJSONResponseBody = {
      throwHttpErrors: false,
      method: 'POST',
      responseType: 'json',
      headers: {
        'Authorization': `Bearer ${this.token}`
      },
      searchParams: query,
      json: payload
    };

    const response = await got(`${SpotifyAPI.API_HOSTNAME}${route}`, gotOptions);
    return response.body as Record<string, any>;
  }

  public async searchQuery(queryText: string) {
    const searchResult = await this.httpRequestGet('search', {
      q: queryText,
      type: 'track'
    });

    console.log(searchResult);

    return searchResult;
  }

  public async enqueueTrack(trackUri: string) {
    const enqueueResult = await this.httpRequestPost('me/player/queue', {
      uri: trackUri
    });

    console.log(enqueueResult);

    return enqueueResult;
  }

  public async nextTrack() {
    const skipResult = await this.httpRequestPost('me/player/next');
    return skipResult;
  }

  public async previousTrack() {
    const prevTrackResult = await this.httpRequestPost('me/player/previous');
    return prevTrackResult;
  }

  public async getRecentlyPlayed(count: number) {
    const prevTracks = await this.httpRequestGet('me/player/recently-played', {limit: count});
    return prevTracks;
  }

  public async getCurrentTrack() {
    const currentTrack = await this.httpRequestGet('me/player/currently-playing');
    return currentTrack;
  }

  public async getTrackById(trackId: string) {
    const trackInfo = await this.httpRequestGet('tracks/' + trackId);
    return trackInfo;
  }

  public async setPlayerVolume(volume: number) {
    const volumeChangeResult = await this.httpRequestPut('me/player/volume', { volume_percent: volume.toFixed(0) });
    return volumeChangeResult;
  }
}
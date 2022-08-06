import dotenv from 'dotenv';
import { dupStore } from './dublicateStore';
dotenv.config();

import { SpotifyAPI } from './spotifyApi';
import { getTrackIdFromUrl, isSpotifyUrl, songItemToReadable } from './spotifyUtils';
import { TwitchListener } from './twitchListener';

async function entryPoint() {
  const listener = new TwitchListener(process.env.TWITCH_CLIENT_ID, process.env.TWITCH_OAUTH_SECRET, [ process.env.TWITCH_OBSERVER_CHANNEL ]);
  const spotifyAPI = new SpotifyAPI(process.env.SPOTIFY_TOKEN);

  listener.on('songRequest', async (query: string, requester: string) => {
    if (isSpotifyUrl(query)) {
      const trackId = getTrackIdFromUrl(query);
      const trackInfo = await spotifyAPI.getTrackById(trackId);
      if (trackInfo.uri === undefined) {
        listener.sendMessage(process.env.TWITCH_OBSERVER_CHANNEL, `@${requester} ничего не найдено в Spotify`);
      } else {
        const uri = trackInfo.uri;
        if (dupStore.exist(uri)) {
          listener.sendMessage(process.env.TWITCH_OBSERVER_CHANNEL, `@${requester} этот трек уже играл за последний час`);
          return;
        }
        const songName = songItemToReadable(trackInfo);
        const enqueueResult = await spotifyAPI.enqueueTrack(uri);
        dupStore.add(uri);
        listener.sendMessage(process.env.TWITCH_OBSERVER_CHANNEL, `@${requester} ${songName} добавлена в очередь`);
      }
    } else {
      const queryResult = await spotifyAPI.searchQuery(query);
    
      if (queryResult.tracks !== undefined && queryResult.tracks.items.length > 0) {
        const mostRelevant = queryResult.tracks.items[0];
        const mostRelevantTrackId = mostRelevant.uri;
        if (dupStore.exist(mostRelevantTrackId)) {
          listener.sendMessage(process.env.TWITCH_OBSERVER_CHANNEL, `@${requester} этот трек уже играл за последний час`);
          return;
        }
  
        const songName = songItemToReadable(mostRelevant);
  
        const enqueueResult = await spotifyAPI.enqueueTrack(mostRelevantTrackId);
        dupStore.add(mostRelevantTrackId);
  
        listener.sendMessage(process.env.TWITCH_OBSERVER_CHANNEL, `@${requester} ${songName} добавлена в очередь`);
        //accept points charge
      } else {
        listener.sendMessage(process.env.TWITCH_OBSERVER_CHANNEL, `@${requester} ничего не найдено в Spotify`);
      }
    }
  });

  listener.on('songSkip', async (requester: string) => {
    await spotifyAPI.nextTrack();
    listener.sendMessage(process.env.TWITCH_OBSERVER_CHANNEL, `@${requester} трек пропущен по твоему запросу`);
  });

  listener.on('songCurrent', async (channel: string, user: string) => {
    const currentTrack = await spotifyAPI.getCurrentTrack();
    if (currentTrack.item === undefined) {
      listener.sendMessage(channel, `@${user} сорян, у меня какая-то ошибка произошла`);
      return;
    }

    const songName = songItemToReadable(currentTrack.item);
    const message = `@${user} Текущий трек ${songName}`;
    listener.sendMessage(channel, message);
  });

  listener.on('songsRecent', async (channel: string, user: string) => {
    const recentTracks = await spotifyAPI.getRecentlyPlayed(5);
    const readableNames = recentTracks.items.map((track: any) => songItemToReadable(track.track));
    
    listener.sendMessage(channel, `@${user} последние проигранные треки: ${readableNames.join('; ')}`);
  });

  listener.on('songHelp', (channel: string, user: string) => {
    listener.sendMessage(channel, `@${user} список доступных комманд: !sc - узнать текущий трек; !sr - список прошлых треков`);
  });

  listener.on('changeVolume', (volume) => {
    spotifyAPI.setPlayerVolume(volume);
    listener.sendMessage(process.env.TWITCH_OBSERVER_CHANNEL, `Звук выставлен на ${volume.toFixed(0)}%`);
  });
  
  listener.listen();
}

entryPoint();

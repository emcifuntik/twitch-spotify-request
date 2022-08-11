import { ApiClient, HelixChannel } from "@twurple/api";
import { AccessToken, RefreshingAuthProvider } from "@twurple/auth";
import { ChatClient } from "@twurple/chat";
import { EventSubChannelRedemptionAddEvent } from "@twurple/eventsub";
import { Reward, RewardID, RewardModel } from "../db/models/reward";
import { Streamer, StreamerModel } from "../db/models/streamer";
import { dupStore } from "../dublicateStore";
import { QueueController, QueueItem } from "../spotify/queueController";
import { SpotifyAPI } from "../spotifyApi";
import { getTrackIdFromUrl, isSpotifyUrl, songItemToReadable } from "../spotifyUtils";
import { formatTime } from "../utils/formatTime";
import { eventSub } from "./eventSub";

enum ChatCommands {
  SongQueue = '!sq',
  SongCurrent = '!sc',
  SongsRecent = '!sr',
  SongVolume = '!volume',
  SongHelp = '!songhelp'
}

export class RewardListener {
  private static QUEUE_CALC_DELAY: number = 30000;
  private static ALLOWED_BROADCASTER_TYPES: Set<string> = new Set(['partner', 'affiliate']);

  private streamerTwitchId: number;
  private streamerId: number;
  private lastQueueCalcTime: number = 0;
  private lastQueue: string[] = [];
  private streamerName: string;
  private streamerNameLastUpdate: number = 0;

  private rewards: RewardModel[];
  private authProvider: RefreshingAuthProvider;
  private apiClient: ApiClient;
  private spotifyClient: SpotifyAPI;
  private chatClient: ChatClient;
  private queueController: QueueController;

  constructor(streamerData: StreamerModel) {
    this.streamerTwitchId = streamerData.streamer_channel_id;
    this.streamerId = streamerData.streamer_id;

    this.authProvider = new RefreshingAuthProvider({
      clientId: process.env.TWITCH_CLIENT_ID,
      clientSecret: process.env.TWITCH_CLIENT_SECRET,
      onRefresh: this.onTokenRefresh.bind(this)
    }, {
      accessToken: streamerData.streamer_twitch_token,
      refreshToken: streamerData.streamer_twitch_refresh,
      expiresIn: 100,
      obtainmentTimestamp: Date.now()
    });
    this.apiClient = new ApiClient({ authProvider: this.authProvider });
    this.queueController = new QueueController(this.streamerId);

    //Spotify related code
    this.spotifyClient = new SpotifyAPI(streamerData.streamer_spotify_token, streamerData.streamer_spotify_refresh);
    this.spotifyClient.on('tokenUpdated', this.spotifyTokenUpdated.bind(this));
  }

  private async spotifyTokenUpdated(token: string) {
    await Streamer.refreshSpotifyToken(this.streamerId, token);
  }

  private getRewardByInternalId(reward: RewardID) {
    const requestReward = this.rewards.filter((value) => value.reward_internal_id === reward);
    if (requestReward.length > 0) {
      return requestReward[0];
    }
    return null;
  }

  private async onTokenRefresh(token: AccessToken) {
    await Streamer.refreshTwitchTokens(this.streamerId, token.accessToken, token.refreshToken);
  }

  public async setupRequestSongReward() {
    try {
      const requestSongResult = await this.apiClient.channelPoints.createCustomReward(this.streamerTwitchId, {
        cost: 300,
        title: 'Request song',
        backgroundColor: '#aaaa00',
        userInputRequired: true,
        prompt: 'Enter artist and song name to add request',
        isEnabled: true
      });
      await Reward.createReward(this.streamerId, RewardID.RequestSong, requestSongResult.id);
      const subscriptionResult = await eventSub.subscribeToChannelRedemptionAddEventsForReward(this.streamerTwitchId, requestSongResult.id, this.onSongRequest.bind(this));
    }
    catch(err) {
      console.error(err);
    }
  }

  public async setupSkipSongReward() {
    try {
      const skipSongResult = await this.apiClient.channelPoints.createCustomReward(this.streamerTwitchId, {
        cost: 1000,
        title: 'Skip song',
        backgroundColor: '#00aaaa',
        userInputRequired: false,
        prompt: 'Skip current song',
        isEnabled: true
      });
      await Reward.createReward(this.streamerId, RewardID.SkipSong, skipSongResult.id);
      const subscriptionResult = await eventSub.subscribeToChannelRedemptionAddEventsForReward(this.streamerTwitchId, skipSongResult.id, this.onSongSkip.bind(this));
    }
    catch(err) {
      console.error(err);
    }
  }

  public async beforeSetup() {
    const rewards = await Reward.getRewards(this.streamerId) as any[];
    this.rewards = rewards;
  }

  public async sendMessage(text: string) {
    const channelName = await this.getChannelName();
    await this.chatClient.action(channelName, text);
  }

  public async getChannelName() {
    if (Date.now() - this.streamerNameLastUpdate < 3600000) return this.streamerName;

    const channelInfo = await this.apiClient.channels.getChannelInfoById(this.streamerTwitchId);
    this.streamerName = channelInfo.name;

    this.streamerNameLastUpdate = Date.now();
    return this.streamerName;
  }

  public async setup() {
    const channelInfo: HelixChannel = await this.apiClient.channels.getChannelInfoById(this.streamerTwitchId);
    const broadcaster = await channelInfo.getBroadcaster();
    if (!RewardListener.ALLOWED_BROADCASTER_TYPES.has(broadcaster.broadcasterType)) return;

    await this.beforeSetup();

    // Join chat and listen to commands
    this.chatClient = new ChatClient({
      authProvider: this.authProvider,
      channels: [ channelInfo.name ]
    });
    this.chatClient.onMessage(this.onMessage.bind(this));
    await this.chatClient.connect();

    await this.apiClient.eventSub.deleteAllSubscriptions();

    const requestSongReward = this.getRewardByInternalId(RewardID.RequestSong);
    if (requestSongReward === null) {
      await this.setupRequestSongReward();
    } else {
      try {
        await eventSub.subscribeToChannelRedemptionAddEventsForReward(this.streamerTwitchId, requestSongReward.reward_twitch_id, this.onSongRequest.bind(this));
      } catch(err) {

      }
    }

    const skipSongReward = this.getRewardByInternalId(RewardID.SkipSong);
    if (skipSongReward === null) {
      await this.setupSkipSongReward();
    } else {
      try {
        await eventSub.subscribeToChannelRedemptionAddEventsForReward(this.streamerTwitchId, skipSongReward.reward_twitch_id, this.onSongSkip.bind(this));
      } catch(err) {

      }
    }
  }

  private async onSongRequest(e: EventSubChannelRedemptionAddEvent) {
    try {
      const query = e.input;
      if (isSpotifyUrl(query)) {
        const trackId = getTrackIdFromUrl(query);
        const trackInfo = await this.spotifyClient.getTrackById(trackId);
        if (trackInfo.uri === undefined) {
          this.sendMessage(`@${e.userName} ничего не найдено в Spotify`);
        } else {
          const uri = trackInfo.uri;
          if (dupStore.exist(uri)) {
            this.sendMessage(`@${e.userName} этот трек уже играл за последний час`);
            return;
          }
          const songName = songItemToReadable(trackInfo);

          this.queueController.add(songName, trackInfo.duration_ms, trackInfo.uri);

          const enqueueResult = await this.spotifyClient.enqueueTrack(uri);
          dupStore.add(uri);
          this.sendMessage(`@${e.userName} ${songName} добавлена в очередь`);
        }
      } else {
        const queryResult = await this.spotifyClient.searchQuery(query);
      
        if (queryResult.tracks !== undefined && queryResult.tracks.items.length > 0) {
          const mostRelevant = queryResult.tracks.items[0];
          const mostRelevantTrackId = mostRelevant.uri;
          if (dupStore.exist(mostRelevantTrackId)) {
            this.sendMessage(`@${e.userName} этот трек уже играл за последний час`);
            return;
          }
    
          const songName = songItemToReadable(mostRelevant);

          this.queueController.add(songName, mostRelevant.duration_ms, mostRelevant.uri);
    
          const enqueueResult = await this.spotifyClient.enqueueTrack(mostRelevantTrackId);
          dupStore.add(mostRelevantTrackId);
    
          this.sendMessage(`@${e.userName} ${songName} добавлена в очередь`);
          await this.apiClient.channelPoints.updateRedemptionStatusByIds(this.streamerTwitchId, e.rewardId, [ e.id ], 'FULFILLED');
        } else {
          this.sendMessage(`@${e.userName} ничего не найдено в Spotify`);
          await this.apiClient.channelPoints.updateRedemptionStatusByIds(this.streamerTwitchId, e.rewardId, [ e.id ], 'CANCELED');
        }
      }
    }
    catch(err) {
      console.error(err);
    }
  }

  private async onSongSkip(e: EventSubChannelRedemptionAddEvent) {
    await this.apiClient.channelPoints.updateRedemptionStatusByIds(this.streamerTwitchId, e.rewardId, [ e.id ], 'FULFILLED');
    await this.spotifyClient.nextTrack();

    await this.sendMessage(`@${e.userName} трек пропущен по твоему запросу`);
  }

  private async onSongHelp(channel: string, user: string, args: string[]) {
    this.sendMessage(`@${user} список доступных комманд: !sc - узнать текущий трек; !sr - список прошлых треков`);
  }

  private async onChangeVolume(channel: string, user: string, args: string[]) {
    const mods = await this.chatClient.getMods(channel);
    if (args.length < 1) return;
    const volume = +args[0];
    if (Number.isNaN(volume)) return;
    const volumeLevel = Math.min(Math.max(volume, 0), 100)
    
    if (mods.includes(user) || user == channel.replace('#', '')) {
      this.spotifyClient.setPlayerVolume(volumeLevel);
      this.sendMessage(`@${user} Звук выставлен на ${volume.toFixed(0)}%`);
    }
  }

  private async onRecentSongs(channel: string, user: string, args: string[]) {
    const recentTracks = await this.spotifyClient.getRecentlyPlayed(5);
    const readableNames = recentTracks.items.map((track: any) => songItemToReadable(track.track));
    this.sendMessage(`@${user} последние проигранные треки: ${readableNames.join('; ')}`);
  }

  private async onCurrentSong(channel: string, user: string, args: string[]) {
    const currentTrack = await this.spotifyClient.getCurrentTrack();
    if (currentTrack.item === undefined) {
      this.sendMessage(`@${user} сорян, у меня какая-то ошибка произошла`);
      return;
    }

    const songName = songItemToReadable(currentTrack.item);
    const message = `@${user} Текущий трек ${songName}`;
    this.sendMessage(message);
  }
  
  private async onSongsQueue(channel: string, user: string, args: string[]) {
    const prettyQueue = await this.getFullQueue();

    if (prettyQueue.length > 5) {
      this.sendMessage(`@${user} текущая очередь треков: ${prettyQueue.slice(0, 5).join('; ')}. https://catjammusic.com/queue/${this.streamerId}`);
    } else {
      this.sendMessage(`@${user} текущая очередь треков: ${prettyQueue.join('; ')}`);
    }
  }

  public async getFullQueue() {
    if (Date.now() - this.lastQueueCalcTime < RewardListener.QUEUE_CALC_DELAY) {
      return this.lastQueue;
    }

    const currentTrack = await this.spotifyClient.getCurrentTrack();
    if (currentTrack.item === undefined) {
      return [];
    }

    const currentProgress = currentTrack.progress_ms;
    const duration = currentTrack.item.duration_ms;

    const timeLeft = duration - currentProgress;

    const queue = this.queueController.getFrom(currentTrack.item.uri);

    let currentWaitTime = timeLeft;
    for (let q of queue) {
      q.timeTillSong = currentWaitTime;
      currentWaitTime += q.duration;
    }

    const prettyQueue = queue.map((value) => {
      return `${value.songName} (${formatTime(value.timeTillSong)})`;
    });

    this.lastQueue = prettyQueue;
    this.lastQueueCalcTime = Date.now();
    return prettyQueue;
  }

  private async onCommand(channel: string, user: string, command: string, args: string[]) {
    switch(command) {
      case ChatCommands.SongHelp:
        this.onSongHelp(channel, user, args);
        break
      case ChatCommands.SongVolume:
        this.onChangeVolume(channel, user, args);
        break;
      case ChatCommands.SongsRecent:
        this.onRecentSongs(channel, user, args);
        break;
      case ChatCommands.SongCurrent:
        this.onCurrentSong(channel, user, args);
        break;
      case ChatCommands.SongQueue:
        this.onSongsQueue(channel, user, args);
        break;
      default:
        break;
    }
  }

  private async onMessage(channel: string, user: string, message: string) {
    if (!message.startsWith('!'))
      return;

    const parts = message.split(' ');
    const command = parts[0];
    parts.shift();

    this.onCommand(channel, user, command, parts);
  }
}

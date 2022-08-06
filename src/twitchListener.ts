import { EventEmitter } from 'node:events';
import { StaticAuthProvider } from '@twurple/auth';
import { ChatClient } from '@twurple/chat';
import { PubSubClient, PubSubRedemptionMessage } from '@twurple/pubsub';

enum RewardID {
  SongRequest = 'b5600737-da87-45c3-a564-8d7969b7f400',
  SongSkip = 'd99ccaf2-0290-44d2-b7b6-c335fa4556ac',
}

enum ChatCommands {
  SongQueue = '!sq',
  SongCurrent = '!sc',
  SongsRecent = '!sr',
  SongVolume = '!volume',
  SongHelp = '!songhelp'
}

export class TwitchListener extends EventEmitter {
  private authProvider: StaticAuthProvider;
  private pubSubClient: PubSubClient;
  private channels: string[];
  private chatClient: ChatClient;

  constructor(clientId: string, oauthToken: string, channels: string[]) {
    super();

    this.channels = channels;
    this.authProvider = new StaticAuthProvider(clientId, oauthToken);
  }

  public sendMessage(channel: string, message: string) {
    this.chatClient.action(channel, message);
  }

  onRedemption(message: PubSubRedemptionMessage) {
    switch(message.rewardId) {
      case RewardID.SongRequest:
        this.emit('songRequest', message.message, message.userName);
        break;
      case RewardID.SongSkip:
        this.emit('songSkip', message.userName);
        break;
      default:
        console.log(`Received unknown reward id: ${message.rewardId}`);
        break;
    }
  }

  async onMessage(channel: string, user: string, message: string) {
    if (message.length > 0 && message.charAt(0) == '!') {
      const messageParts = message.split(' ');
      const command = messageParts[0];
      messageParts.shift();

      switch(command) {
        case ChatCommands.SongQueue:
          
          break;
        case ChatCommands.SongCurrent:
          this.emit('songCurrent', channel, user);
          break;
        case ChatCommands.SongsRecent:
          this.emit('songsRecent', channel, user);
          break;
        case ChatCommands.SongHelp:
          this.emit('songHelp', channel, user);
          break;
        case ChatCommands.SongVolume:
          if (messageParts.length > 0) {
            const volume = +messageParts[0];
            if (!Number.isNaN(volume)) {
              const mods = await this.chatClient.getMods(channel);
              if (mods.includes(user)) {
                this.emit('changeVolume', Math.min(Math.max(volume, 0), 100));
              }
            }
          }
          break;
      }
      console.log(message);
    }
  }

  public async listen() {
    this.pubSubClient = new PubSubClient();
    const userId = await this.pubSubClient.registerUserListener(this.authProvider);

    this.chatClient = new ChatClient({ authProvider: this.authProvider, channels: this.channels });
    
    this.chatClient.onMessage(this.onMessage.bind(this));
    await this.chatClient.connect();

    const redemptionListener = await this.pubSubClient.onRedemption(userId, this.onRedemption.bind(this));
    console.log(userId);
  }
}
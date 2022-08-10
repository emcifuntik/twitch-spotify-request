import { EventSubMiddleware } from '@twurple/eventsub';
import { ClientCredentialsAuthProvider } from '@twurple/auth';
import { ApiClient } from '@twurple/api';

const authProvider = new ClientCredentialsAuthProvider(process.env.TWITCH_CLIENT_ID, process.env.TWITCH_CLIENT_SECRET);

export const eventSub = new EventSubMiddleware({
  hostName: 'catjammusic.com',
  pathPrefix: '/twitch_eventsub',
  secret: process.env.TWITCH_EVENTSUB_SECRET,
  apiClient: new ApiClient({
    authProvider: authProvider
  })
});


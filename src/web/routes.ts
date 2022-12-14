import { exchangeCode, getTokenInfo } from '@twurple/auth';
import Express from 'express';
import { Streamer, StreamerModel } from '../db/models/streamer';
import { makeid } from '../utils/randomString';
import { stringify as makeQuery } from 'querystring';
import got from 'got';
import { addListener, getListener } from '../twitch/listenAll';
import { formatTime } from '../utils/formatTime';
import path from 'path';

const router = Express.Router();
const redirectURITwitch = process.env.BOT_HOST + 'oauth/twitch';
const redirectURISpotify = process.env.BOT_HOST + 'oauth/spotify';

const requiredScopesTwitch = [
  'chat:read',
  'chat:edit',
  'channel:read:redemptions',
  'channel:manage:redemptions'
];

const requiredScopesSpotify = [
  'user-modify-playback-state',
  'user-read-currently-playing',
  'user-read-playback-position',
  'user-read-playback-state',
  'user-read-recently-played'
];

router.get('/oauth/spotify', async (req, res, next) => {
  const code = req.query.code;
  const state = req.query.state;

  if (!code || !state || typeof code !== 'string' || typeof state !== 'string') {
    res.status(400).end('Invalid params');
    return;
  }

  const response = await got.post('https://accounts.spotify.com/api/token', {
    throwHttpErrors: false,
    headers: {
      'Authorization': 'Basic ' + Buffer.from(process.env.SPOTIFY_CLIENT_ID + ':' + process.env.SPOTIFY_CLIENT_SECRET).toString('base64')
    },
    form: {
      code: code,
      redirect_uri: redirectURISpotify,
      grant_type: 'authorization_code'
    },
    responseType: 'json'
  });

  const responseData = response.body as Record<string, any>;
  if (!responseData.access_token || !responseData.refresh_token) {
    res.status(400).end('Invalid response');
    return;
  }

  await Streamer.updateSpotifyTokens(state, responseData.access_token, responseData.refresh_token);
  const streamer: StreamerModel = await Streamer.getBySpotifyState(state) as StreamerModel;
  addListener(streamer);
  res.status(200).end('OK');
});

router.get('/auth', async (req, res, next) => {
  res.redirect(`https://id.twitch.tv/oauth2/authorize?client_id=${process.env.TWITCH_CLIENT_ID}&redirect_uri=${redirectURITwitch}&response_type=code&scope=${requiredScopesTwitch.join('+')}`)
});

router.get('/oauth/twitch', async (req, res, next) => {
  const code = req.query.code;
  const scopes = req.query.scope;
  if (!code || !scopes || typeof code !== 'string') {
    res.status(400).end('Invalid params');
    return;
  }

  try {
    const accessToken = await exchangeCode(process.env.TWITCH_CLIENT_ID, process.env.TWITCH_CLIENT_SECRET, code, redirectURITwitch);
    const tokenInfo = await getTokenInfo(accessToken.accessToken, process.env.TWITCH_CLIENT_ID);
    
    const spotifyRandomState = makeid(32);
    Streamer.createOrUpdateTwitchData(tokenInfo.userId, tokenInfo.userName, accessToken.accessToken, accessToken.refreshToken, spotifyRandomState);

    res.redirect('https://accounts.spotify.com/authorize?' +
      makeQuery({
        response_type: 'code',
        client_id: process.env.SPOTIFY_CLIENT_ID,
        scope: requiredScopesSpotify.join(' '),
        redirect_uri: redirectURISpotify,
        state: spotifyRandomState
      })
    );
  } catch(err) {
    console.error(err);
  }
});

router.get('/api/queue/:streamerId', async (req, res, next) => {
  res.header("Access-Control-Allow-Origin", "*"); // update to match the domain you will make the request from
  res.header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept");

  const streamerId = req.params.streamerId;
  if (!streamerId || Number.isNaN(+streamerId)) {
    return res.status(400).end('Wrong streamer id');
  }
  const streamerNumericId = +streamerId;

  const listener = getListener(streamerNumericId);
  if (!listener) {
    return res.status(400).end('Streamer does not exist');
  }

  const queueData = await listener.getQueueData();
  
  res.status(200).json({
    ts: listener.queueUpdateTime,
    q: queueData
  });
});

router.use('/queue', async (req, res, next) => {
  res.sendFile(path.resolve('static/index.html'));
});

export default router;

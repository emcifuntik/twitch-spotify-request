import Express from 'express';
import https from 'https';
import { readFileSync } from 'fs';
import { eventSub } from '../twitch/eventSub';
import router from './routes';

const app = Express();

eventSub.apply(app);

app.use(router);

const server = https.createServer({
  key: readFileSync('./ssl/key.pem'),
  cert: readFileSync('./ssl/cert.pem')
}, app);

server.listen(443, () => {
  console.log('Listening on 443');
  eventSub.markAsReady();
});

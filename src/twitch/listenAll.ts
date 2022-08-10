import { Streamer } from "../db/models/streamer";
import { RewardListener } from "./rewardListener";

const streamers = await Streamer.getAll();

for (let streamer of streamers) {
  const listener = new RewardListener(streamer);
  await listener.setup();
}

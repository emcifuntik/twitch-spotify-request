import { Streamer } from "../db/models/streamer";
import { RewardListener } from "./rewardListener";

const streamers = await Streamer.getAll();

const listeners: Map<number, RewardListener> = new Map();

for (let streamer of streamers) {
  try {
    const listener = new RewardListener(streamer);
    await listener.setup();
    listeners.set(streamer.streamer_id, listener);
  } catch(err) {
    if (err._statusCode === 400) {
      // Somebody revoked our application
      await Streamer.cascadeDelete(streamer.streamer_id);
    }
  }
}

export function getListener(streamerId: number): RewardListener | null {
  if (!listeners.has(streamerId)) return null;
  return listeners.get(streamerId);
}
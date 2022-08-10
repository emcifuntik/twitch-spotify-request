import connection from "../connection";
import { ResultSetHeader, RowDataPacket } from 'mysql2';

export enum RewardID {
  RequestSong = 1,
  SkipSong = 2
}

export interface RewardModel {
  reward_id: number,
  reward_streamer: number,
  reward_internal_id: RewardID,
  reward_twitch_id: string
}

export class Reward {
  public static getRewards(streamerId: number) {
    return new Promise((resolve, reject) => {
      connection.execute('SELECT * FROM `rewards` WHERE `reward_streamer` = ?', [streamerId], (err, result: RowDataPacket[][]) => {
        if (err) {
          console.error(err);
          return;
        }

        return resolve(result);
      });
    });
  }

  public static createReward(streamerId: number, internalId: number, twitchId: string) {
    return new Promise((resolve, reject) => {
      connection.execute('INSERT INTO `rewards` (reward_streamer, reward_internal_id, reward_twitch_id) VALUES (?, ?, ?)', [streamerId, internalId, twitchId], (err, result: ResultSetHeader) => {
        if (err) {
          console.error(err);
          return;
        }

        return resolve(result.insertId);
      });
    });
  }
}
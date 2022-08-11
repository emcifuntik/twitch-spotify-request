import connection from "../connection";
import { ResultSetHeader, RowDataPacket } from 'mysql2';

export interface StreamerModel {
  streamer_id: number,
  streamer_channel_id: number,
  streamer_name: string,
  streamer_twitch_token: string,
  streamer_twitch_refresh: string,
  streamer_spotify_token: string,
  streamer_spotify_refresh: string,
  streamer_spotify_state: string
}

export class Streamer {
  public static createOrUpdateTwitchData(userId: string, userName: string, accessToken: string, refreshToken: string, spotifyState: string): Promise<number> {
    const userIdNumber = +userId;
    return new Promise((resolve, reject) => {
      connection.execute('INSERT INTO `streamer` (streamer_channel_id, streamer_name, streamer_twitch_token, streamer_twitch_refresh, streamer_spotify_state) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE streamer_name=?, streamer_twitch_token=?, streamer_twitch_refresh=?, streamer_spotify_state=?', [userIdNumber, userName, accessToken, refreshToken, spotifyState, userName, accessToken, refreshToken, spotifyState], (err, result: ResultSetHeader) => {
        if (err) {
          console.error(err);
          return reject(err);
        }
  
        return resolve(result.insertId);
      });
    });
  }

  public static updateSpotifyTokens(state: string, token: string, refreshToken: string) {
    return new Promise((resolve, reject) => {
      connection.execute('UPDATE streamer SET `streamer_spotify_token` = ?, `streamer_spotify_refresh` = ? WHERE `streamer_spotify_state` = ? LIMIT 1', [token, refreshToken, state], (err, result: ResultSetHeader) => {
        if (err) {
          console.error(err);
          return reject(err);
        }
  
        return resolve(result.affectedRows);
      });
    });
  }

  public static refreshTwitchTokens(streamerId: number, accessToken: string, refreshToken: string) {
    return new Promise((resolve, reject) => {
      connection.execute('UPDATE `streamer` SET `streamer_twitch_token` = ?, `streamer_twitch_refresh` = ? WHERE `streamer_id` = ? LIMIT 1', [accessToken, refreshToken, streamerId], (err, result: ResultSetHeader) => {
        if (err) {
          console.error(err);
          return reject(err);
        }

        return resolve(result.affectedRows);
      })
    })
  }

  public static refreshSpotifyToken(streamerId: number, accessToken: string) {
    return new Promise((resolve, reject) => {
      connection.execute('UPDATE `streamer` SET `streamer_spotify_token` = ? WHERE `streamer_id` = ? LIMIT 1', [accessToken, streamerId], (err, result: ResultSetHeader) => {
        if (err) {
          console.error(err);
          return reject(err);
        }

        return resolve(result.affectedRows);
      })
    })
  }

  public static getAll(): Promise<StreamerModel[]> {
    return new Promise((resolve, reject) => {
      connection.execute('SELECT * FROM `streamer`', (err, result: RowDataPacket[][]) => {
        if (err) {
          console.error(err);
          return reject(err);
        }
  
        return resolve(result as unknown as Promise<StreamerModel[]>);
      });
    });
  }

  public static getById(streamerId: string) {
    const streamerIdNumber = +streamerId;

    return new Promise((resolve, reject) => {
      connection.execute('SELECT * FROM `streamer` WHERE `streamer_channel_id` = ? LIMIT 1', [streamerIdNumber], (err, result: RowDataPacket[][]) => {
        if (err) {
          console.error(err);
          return reject(err);
        }
  
        return resolve(result[0]);
      });
    });
  }

  public static cascadeDelete(streamerId: number) {
    return new Promise((resolve, reject) => {
      connection.execute('DELETE FROM `rewards` WHERE `reward_streamer` = ?', [streamerId], (err, result) => {
        if (err) {
          console.error(err);
          return reject(err);
        }
  
        connection.execute('DELETE FROM `streamer` WHERE `streamer_id` = ?', [streamerId], (err, result) => {
          if (err) {
            console.error(err);
            return reject(err);
          }
    
          return resolve(true);
        });
      });
    });
  }
}
import { readFileSync, writeFileSync, existsSync } from 'fs';

import AsyncLock from 'async-lock';

export interface QueueItem {
  songName: string,
  duration: number,
  uri: string,
  timeTillSong?: number
}

export class QueueController {
  private queue: QueueItem[] = [];
  private path: string;
  private asyncLocker: AsyncLock;
  constructor(streamerId: number) {
    this.path = `./queueStore/${streamerId}.json`;
    this.asyncLocker = new AsyncLock();
    if (existsSync(this.path)) {
      try {
        const jsonText = readFileSync(this.path, 'utf-8');
        const jsonObj = JSON.parse(jsonText);
        this.queue = jsonObj;
      } catch(err) {
        
      }
    }
  }

  private save() {
    this.asyncLocker.acquire(this.path, (done) => {
      const jsonText = JSON.stringify(this.queue);
      writeFileSync(this.path, jsonText);
      done();
    });
  }

  add(songName: string, duration: number, uri: string) {
    this.queue.push({
      songName,
      duration,
      uri
    });
    this.save();
  }

  getFrom(uri: string) {
    let index = -1;

    if (this.queue.length > 100) {
      this.queue.splice(0, this.queue.length - 100);
      this.save();
    }

    for (let i = 0; i < this.queue.length; ++i) {
      if (this.queue[i].uri === uri) {
        index = i;
        break;
      }
    }

    if (index === -1) {
      return [];
    } else {
      return this.queue.slice(index + 1);
    }
  }
}
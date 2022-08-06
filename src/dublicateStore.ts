

class DuplicateStore {
  private static HOLD_TIME: number = 60 * 60 * 1000; 
  private store: Map<string, number> = new Map();

  constructor() {

  }

  public add(id: string) {
    if (!this.store.has(id)) {
      this.store.set(id, Date.now());
    }
  }
  
  public exist(id: string) {
    if (!this.store.has(id)) return false;

    const lastPlayTime = this.store.get(id);
    const timeFromLastPlay = Date.now() - lastPlayTime;
    if (timeFromLastPlay < DuplicateStore.HOLD_TIME) return true;

    this.store.delete(id);
    return false;
  }
}

export const dupStore = new DuplicateStore();



export function formatTime(timeInMs: number) {
  const timeSeconds = Math.floor(timeInMs / 1000);

  const seconds = timeSeconds % 60;
  const minutes = Math.floor(timeSeconds / 60);

  return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
}
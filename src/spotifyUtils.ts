
const regex = /https:\/\/open\.spotify\.com\/track\/([0-9A-Za-z]+)(\?.+)?/

export function songItemToReadable(songdata: any) {
  const artists = songdata.artists;
  const artistsNames = artists.map((value: any) => value.name);
  const leftPart = artistsNames.join(' & ');
  const rightPart = songdata.name;
  return `${leftPart} - ${rightPart}`;
}

export function isSpotifyUrl(url: string): boolean {
  return regex.test(url);
}

export function getTrackIdFromUrl(url: string): string {
  const match = regex.exec(url);

  if (match) {
    return match[1];
  }
  return null;
}
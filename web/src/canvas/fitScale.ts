// How much to scale a width x height sprite so it fits a box x box square
// without distorting it: a whole-number factor when the sprite fits (so
// every sprite pixel stays an equal, crisp block), otherwise a fractional
// shrink. The preview and the frame-grid thumbnails both size themselves
// with this instead of stretching into a square.
export function fitScale(width: number, height: number, box: number): number {
  const longest = Math.max(width, height, 1);
  return longest <= box ? Math.floor(box / longest) : box / longest;
}

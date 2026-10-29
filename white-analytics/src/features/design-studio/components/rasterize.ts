/**
 * Browser-only: draw an image URL (the mock SVG data URL) onto a canvas at the
 * format's pixel size and return a PNG File. Storage rejects SVG on purpose,
 * so this is how a mock design enters the media library.
 */
export async function rasterizeToPng(src: string, width: number, height: number, filename: string): Promise<File> {
  const img = new Image();
  img.decoding = "async";
  img.src = src;
  await img.decode();
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("NO_CANVAS");
  ctx.drawImage(img, 0, 0, width, height);
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, "image/png"));
  if (!blob) throw new Error("NO_BLOB");
  return new File([blob], filename, { type: "image/png" });
}

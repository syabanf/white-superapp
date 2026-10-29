/**
 * Satu-satunya tempat versi Graph API. Meta menghapus sebuah versi dua tahun
 * setelah penggantinya rilis (v21.0 dihapus 21 Jan 2027), jadi naikkan di sini
 * saja lalu cek changelog untuk metrik yang berubah.
 */
export const META_GRAPH_VERSION = "v26.0";
export const META_GRAPH_BASE = `https://graph.facebook.com/${META_GRAPH_VERSION}`;
export const META_DIALOG_BASE = `https://www.facebook.com/${META_GRAPH_VERSION}/dialog/oauth`;

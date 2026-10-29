import { describe, expect, it } from "vitest";
import { META_PERMISSIONS } from "@/features/setup/integrations";
import {
  isDemoConnectionId,
  META_SCOPES,
  parseMetaConnectionMeta,
  resolveMetaAsset,
  type MetaAsset,
} from "@/lib/providers/meta-oauth";

const ASSETS: MetaAsset[] = [
  { pageId: "1001", pageName: "Kopi Nusantara", pageUsername: "kopinusantara.id", igId: "17841400000000001", igUsername: "KopiNusantara" },
  { pageId: "1002", pageName: "Toko Tanpa IG", pageUsername: null, igId: null, igUsername: null },
];

describe("resolveMetaAsset", () => {
  it("maps an Instagram username to the numeric IG Business id", () => {
    expect(resolveMetaAsset(ASSETS, "INSTAGRAM", "@kopinusantara")).toEqual({
      externalId: "17841400000000001",
      username: "kopinusantara",
      displayName: "KopiNusantara",
    });
  });

  it("matches a Facebook Page by username, id, or name", () => {
    expect(resolveMetaAsset(ASSETS, "FACEBOOK", "kopinusantara.id")?.externalId).toBe("1001");
    expect(resolveMetaAsset(ASSETS, "FACEBOOK", "1002")?.externalId).toBe("1002");
    expect(resolveMetaAsset(ASSETS, "FACEBOOK", "toko tanpa ig")?.username).toBe("1002");
  });

  it("returns null for accounts the login did not grant", () => {
    expect(resolveMetaAsset(ASSETS, "INSTAGRAM", "kompetitor")).toBeNull();
    expect(resolveMetaAsset(ASSETS, "INSTAGRAM", "")).toBeNull();
  });
});

describe("meta connection helpers", () => {
  it("recognises simulated demo connections", () => {
    expect(isDemoConnectionId("demo-meta")).toBe(true);
    expect(isDemoConnectionId("10203040")).toBe(false);
  });

  it("parses stored connection meta defensively", () => {
    expect(parseMetaConnectionMeta(null)).toBeNull();
    expect(parseMetaConnectionMeta({ assets: "x" })).toBeNull();
    expect(parseMetaConnectionMeta({ assets: ASSETS })?.grantedScopes).toEqual([]);
  });

  it("requests every permission on the App Review checklist that the app uses", () => {
    const required = META_PERMISSIONS.filter((p) => p.required).map((p) => p.name);
    expect([...META_SCOPES].sort()).toEqual([...required].sort());
  });
});

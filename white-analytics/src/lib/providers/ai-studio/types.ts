/**
 * AI studio: the generative helpers behind the brand-aware features
 * (design layouts, image generation, caption rewrites, campaign suggestions).
 * Every task takes the client's brand memory so output stays on-brand.
 * Real adapter = OpenRouter (text + image models); mock = deterministic,
 * offline output used in demo mode and when no key is configured.
 */

/** Subset of `BrandKit` the prompts need (plain, serializable). */
export type BrandContext = {
  clientName: string;
  industry: string | null;
  primaryColor: string;
  secondaryColor: string;
  accentColors: string[];
  headingFont: string;
  bodyFont: string;
  toneOfVoice: string;
  audience: string;
  doList: string[];
  dontList: string[];
  keywords: string[];
  hashtags: string[];
  sampleCaptions: string[];
  visualStyle: string;
};

export const DESIGN_FORMATS = ["FEED_SQUARE", "FEED_PORTRAIT", "STORY", "REEL_COVER", "CAROUSEL"] as const;
export type DesignFormat = (typeof DESIGN_FORMATS)[number];

export const FORMAT_SIZE: Record<DesignFormat, { width: number; height: number }> = {
  FEED_SQUARE: { width: 1080, height: 1080 },
  FEED_PORTRAIT: { width: 1080, height: 1350 },
  STORY: { width: 1080, height: 1920 },
  REEL_COVER: { width: 1080, height: 1920 },
  CAROUSEL: { width: 1080, height: 1350 },
};

export const COMPOSITIONS = ["center-stack", "split-left", "bottom-band", "framed", "big-type"] as const;
export type Composition = (typeof COMPOSITIONS)[number];

export type DesignLayout = {
  headline: string;
  subline: string;
  cta: string;
  composition: Composition;
  /** hex colors in order of use: background, surface, text, accent */
  palette: string[];
  /** human-readable layout notes for the designer */
  elements: string[];
  /** English prompt for the image model (no text in the image) */
  imagePrompt: string;
};

export type DesignLayoutInput = { brand: BrandContext; title: string; brief: string; format: DesignFormat };

export type GeneratedImage = { dataUrl: string; mimeType: string; model: string };
export type ImageInput = { brand: BrandContext; layout: DesignLayout; format: DesignFormat };

export type CaptionVariant = { label: string; hook: string; caption: string; hashtags: string[] };
export type CaptionInput = {
  brand: BrandContext;
  hook: string;
  caption: string;
  hashtags: string[];
  instruction: string;
  platform: string;
};

export type CampaignObjective = "OUTCOME_AWARENESS" | "OUTCOME_TRAFFIC" | "OUTCOME_ENGAGEMENT" | "OUTCOME_LEADS" | "OUTCOME_SALES";
export type CampaignSuggestInput = {
  brand: BrandContext;
  objective: CampaignObjective;
  offer: string;
  dailyBudget: number;
  currency: string;
};
export type CampaignSuggestion = {
  audience: { locations: string[]; ageMin: number; ageMax: number; genders: "ALL" | "MALE" | "FEMALE"; interests: string[] };
  placements: string[];
  creatives: { headline: string; primaryText: string; cta: string }[];
  notes: string;
};

export interface AiStudioProvider {
  readonly id: "openrouter" | "mock";
  readonly model: string;
  designLayout(input: DesignLayoutInput): Promise<DesignLayout>;
  generateImage(input: ImageInput): Promise<GeneratedImage>;
  rewriteCaption(input: CaptionInput): Promise<CaptionVariant[]>;
  suggestCampaign(input: CampaignSuggestInput): Promise<CampaignSuggestion>;
}

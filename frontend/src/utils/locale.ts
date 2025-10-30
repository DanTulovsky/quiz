// Utility to normalize various language identifiers to dataset language codes
// Examples:
//   "hi-IN" -> "hi"
//   "ru-RU" -> "ru"
//   "Hindi" -> "hi"
//   "russian" -> "ru"

// Runtime-loaded map of language names -> codes, hydrated from backend settings
let runtimeNameToCode: Record<string, string> = {};
let runtimeCodes: Set<string> = new Set();
let runtimeLoaded = false;
let runtimeLoadingPromise: Promise<void> | null = null;

export async function ensureLanguagesLoaded(): Promise<void> {
  if (runtimeLoaded) return;
  if (runtimeLoadingPromise) return runtimeLoadingPromise;
  runtimeLoadingPromise = (async () => {
    try {
      const res = await fetch('/v1/settings/languages');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data:
        | { languages?: Array<{ code: string; name: string }> }
        | Array<{ code: string; name: string }> = await res.json();
      const list = Array.isArray(data)
        ? data
        : Array.isArray((data as any)?.languages)
          ? ((data as any).languages as Array<{ code: string; name: string }>)
          : [];
      const map: Record<string, string> = {};
      for (const item of list) {
        if (!item || !item.name || !item.code) continue;
        const code = item.code.toLowerCase();
        map[item.name.toLowerCase()] = code;
        runtimeCodes.add(code);
      }
      runtimeNameToCode = map;
      runtimeLoaded = true;
    } catch (e) {
      // Leave map empty; normalization will still strip locale like hi-IN -> hi
      runtimeNameToCode = {};
      runtimeLoaded = true; // avoid retry loops during this session
    }
  })();
  return runtimeLoadingPromise;
}

/**
 * Normalize an input language identifier into a dataset key code.
 * Accepts BCP-47 locales (e.g., hi-IN), ISO codes (e.g., hi), or names (e.g., Hindi).
 */
export function normalizeLanguageKey(input?: string): string | undefined {
  if (!input) return undefined;

  const lowered = input.toLowerCase().trim();
  if (!lowered) return undefined;

  // If it looks like a locale (xx or xx-YY), strip region/script and return the language part
  const langPart = lowered.split('-')[0];

  // If it's a known runtime code from backend, accept it
  if (runtimeCodes.has(langPart)) return langPart;

  // Otherwise, treat the whole lowered input as a name and map
  if (runtimeNameToCode[lowered]) {
    return runtimeNameToCode[lowered];
  }

  // Unknown / unsupported language
  return undefined;
}

export function languageFallbackChain(preferred?: string): string[] {
  const normalized = normalizeLanguageKey(preferred);
  const chain: string[] = [];
  if (normalized) chain.push(normalized);
  chain.push('en');
  return chain;
}

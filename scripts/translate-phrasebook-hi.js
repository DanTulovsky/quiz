// Translate phrasebook entries to Hindi using Google Translate API.
// - Reads all category JSON files under frontend/src/data/phrasebook/**/xxx.json
// - For each word: if hi is missing or equals en, translate en -> hi.
// - Requires env TRANSLATION_PROVIDERS_GOOGLE_API_KEY.

const fs = require('fs');
const path = require('path');
const fetch = require('node-fetch');

const API_KEY = process.env.TRANSLATION_PROVIDERS_GOOGLE_API_KEY;
const BASE_URL = 'https://translation.googleapis.com/language/translate/v2';

if (!API_KEY) {
  console.error('Missing TRANSLATION_PROVIDERS_GOOGLE_API_KEY environment variable.');
  process.exit(2);
}

const root = path.resolve(__dirname, '..', 'frontend', 'src', 'data', 'phrasebook');

function collectFiles(dir) {
  const out = [];
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const e of entries) {
    const full = path.join(dir, e.name);
    if (e.isDirectory()) out.push(...collectFiles(full));
    else if (e.isFile() && e.name.endsWith('.json') && e.name !== 'index.json' && e.name !== 'info.json') out.push(full);
  }
  return out;
}

function gatherTextsNeedingTranslation(filePath) {
  const raw = fs.readFileSync(filePath, 'utf8');
  const data = JSON.parse(raw);
  const toTranslate = [];
  if (!data || !Array.isArray(data.sections)) return { data, toTranslate };
  for (let s = 0; s < data.sections.length; s++) {
    const section = data.sections[s];
    if (!Array.isArray(section.words)) continue;
    for (let w = 0; w < section.words.length; w++) {
      const word = section.words[w];
      if (!word || typeof word !== 'object') continue;
      const en = word.en && String(word.en).trim();
      if (!en) continue;
      const hi = word.hi && String(word.hi).trim();
      if (!hi || hi === en) {
        toTranslate.push({ s, w, text: en });
      }
    }
  }
  return { data, toTranslate };
}

async function translateBatch(texts) {
  if (texts.length === 0) return [];
  const body = {
    q: texts,
    target: 'hi',
    source: 'en',
    format: 'text',
    model: 'nmt'
  };
  const url = `${BASE_URL}?key=${encodeURIComponent(API_KEY)}`;
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  if (!res.ok) {
    const txt = await res.text();
    throw new Error(`Translate API error ${res.status}: ${txt}`);
  }
  const json = await res.json();
  const translations = (json.data && json.data.translations) || [];
  return translations.map(t => t.translatedText);
}

async function main() {
  const files = collectFiles(root);
  let filesChanged = 0;
  for (const file of files) {
    const { data, toTranslate } = gatherTextsNeedingTranslation(file);
    if (toTranslate.length === 0) continue;

    // Batch translate in chunks (Google allows many, keep it reasonable)
    const BATCH = 80;
    let translatedCount = 0;
    for (let i = 0; i < toTranslate.length; i += BATCH) {
      const slice = toTranslate.slice(i, i + BATCH);
      const inputs = slice.map(x => x.text);
      const outputs = await translateBatch(inputs);
      for (let j = 0; j < slice.length; j++) {
        const { s, w } = slice[j];
        const out = outputs[j];
        if (out) {
          data.sections[s].words[w].hi = out;
          translatedCount++;
        }
      }
      // brief delay to be gentle if many calls
      await new Promise(r => setTimeout(r, 150));
    }

    if (translatedCount > 0) {
      fs.writeFileSync(file, JSON.stringify(data, null, 2) + '\n', 'utf8');
      filesChanged++;
    }
  }
  console.log(`Translated and updated ${filesChanged} files`);
}

main().catch(err => {
  console.error(err.message || err);
  process.exit(1);
});



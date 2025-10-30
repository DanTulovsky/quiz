/*
  Adds missing "hi" translations to all phrasebook JSON files by copying the English ("en") value
  when "hi" is absent. Safe, idempotent.
*/

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..', 'frontend', 'src', 'data', 'phrasebook');

function processFile(filePath) {
  const raw = fs.readFileSync(filePath, 'utf8');
  const data = JSON.parse(raw);
  let changed = false;

  if (!data || !Array.isArray(data.sections)) return false;

  for (const section of data.sections) {
    if (!Array.isArray(section.words)) continue;
    for (const word of section.words) {
      if (word && typeof word === 'object') {
        const en = word.en;
        if (en && word.hi === undefined) {
          word.hi = en; // fallback: copy English as placeholder
          changed = true;
        }
      }
    }
  }

  if (changed) {
    fs.writeFileSync(filePath, JSON.stringify(data, null, 2) + '\n', 'utf8');
  }
  return changed;
}

function walk(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  let totalChanges = 0;
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      totalChanges += walk(full);
    } else if (entry.isFile() && entry.name.endsWith('.json') && entry.name !== 'index.json' && entry.name !== 'info.json') {
      if (processFile(full)) totalChanges += 1;
    }
  }
  return totalChanges;
}

const changes = walk(root);
console.log(`Updated ${changes} files with missing hi translations`);



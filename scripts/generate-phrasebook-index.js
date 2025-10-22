#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

/**
 * Generate phrasebook index.json by scanning directories
 */
function generatePhrasebookIndex() {
  // Try different possible paths for the phrasebook directory
  const possiblePaths = [
    path.join(__dirname, '../frontend/src/data/phrasebook'), // Local development
    path.join(process.cwd(), 'src/data/phrasebook'), // Docker build context
    path.join(__dirname, '../../frontend/src/data/phrasebook'), // Alternative local path
    path.join(__dirname, '../src/data/phrasebook'), // Docker: script in /app/scripts/, frontend in /app/src/
  ];

  let phrasebookDir = null;
  console.log('🔍 Checking possible paths:');
  for (const possiblePath of possiblePaths) {
    console.log(`   - ${possiblePath} ${fs.existsSync(possiblePath) ? '✅' : '❌'}`);
    if (fs.existsSync(possiblePath)) {
      phrasebookDir = possiblePath;
      break;
    }
  }

  if (!phrasebookDir) {
    console.error('❌ Could not find phrasebook directory. Tried paths:');
    possiblePaths.forEach(p => console.error(`   - ${p}`));
    process.exit(1);
  }

  const indexPath = path.join(phrasebookDir, 'index.json');

  console.log('🔍 Scanning phrasebook directories...');
  console.log(`📁 Using phrasebook directory: ${phrasebookDir}`);

  // Read all directories in the phrasebook folder
  const entries = fs.readdirSync(phrasebookDir, { withFileTypes: true });
  const categories = [];

  for (const entry of entries) {
    if (entry.isDirectory()) {
      const categoryDir = path.join(phrasebookDir, entry.name);
      const infoPath = path.join(categoryDir, 'info.json');

      // Check if info.json exists
      if (fs.existsSync(infoPath)) {
        try {
          const infoContent = fs.readFileSync(infoPath, 'utf8');
          const info = JSON.parse(infoContent);

          // Validate that it has required fields
          if (info.id && info.name && info.emoji && info.description) {
            categories.push(entry.name);
            console.log(`✅ Found category: ${entry.name} (${info.name})`);
          } else {
            console.warn(`⚠️  Skipping ${entry.name}: missing required fields in info.json`);
          }
        } catch (error) {
          console.warn(`⚠️  Skipping ${entry.name}: invalid info.json - ${error.message}`);
        }
      } else {
        console.warn(`⚠️  Skipping ${entry.name}: no info.json found`);
      }
    }
  }

  // Sort categories alphabetically for consistent ordering
  categories.sort();

  // Generate the index.json content
  const indexContent = {
    categories: categories,
    generatedBy: 'generate-phrasebook-index.js'
  };

  // Write the index file
  fs.writeFileSync(indexPath, JSON.stringify(indexContent, null, 2));

  console.log(`\n✅ Generated index.json with ${categories.length} categories:`);
  categories.forEach((cat, index) => {
    console.log(`   ${index + 1}. ${cat}`);
  });
  console.log(`\n📁 Written to: ${indexPath}`);
}

// Run the script
if (require.main === module) {
  try {
    generatePhrasebookIndex();
    console.log('\n🎉 Phrasebook index generation complete!');
  } catch (error) {
    console.error('\n❌ Error generating phrasebook index:', error.message);
    process.exit(1);
  }
}

module.exports = { generatePhrasebookIndex };

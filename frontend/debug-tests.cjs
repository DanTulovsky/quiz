#!/usr/bin/env node

const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

// Find all test files
function findTestFiles(dir) {
  const testFiles = [];

  function scanDirectory(currentDir) {
    const items = fs.readdirSync(currentDir);

    for (const item of items) {
      const fullPath = path.join(currentDir, item);
      const stat = fs.statSync(fullPath);

      if (stat.isDirectory() && !item.startsWith('.') && item !== 'node_modules') {
        scanDirectory(fullPath);
      } else if (stat.isFile() && (item.endsWith('.test.tsx') || item.endsWith('.test.ts') || item.endsWith('.spec.tsx') || item.endsWith('.spec.ts'))) {
        const relativePath = path.relative(process.cwd(), fullPath);
        testFiles.push(relativePath);
      }
    }
  }

  scanDirectory(dir);
  return testFiles.sort(); // Sort lexicographically
}

console.log('🔍 Finding all test files...');
const testFiles = findTestFiles('./src');
console.log(`📋 Found ${testFiles.length} test files`);

console.log('\n🚀 Running tests individually to find hanging test...\n');

// Run each test file individually with timeout
let completedTests = 0;

for (const testFile of testFiles) {
  console.log(`\n[${++completedTests}/${testFiles.length}] Testing: ${testFile}`);

  const startTime = Date.now();

  try {
    const vitest = spawn('npx', ['vitest', 'run', testFile], {
      stdio: 'pipe',
      timeout: 60000 // 60 second timeout per test
    });

    let output = '';
    let errorOutput = '';

    vitest.stdout.on('data', (data) => {
      output += data.toString();
    });

    vitest.stderr.on('data', (data) => {
      errorOutput += data.toString();
    });

    vitest.on('close', (code, signal) => {
      const duration = Date.now() - startTime;

      if (signal === 'SIGTERM') {
        console.log(`❌ HANGING TEST FOUND: ${testFile}`);
        console.log(`⏱️  Ran for ${duration}ms before timeout`);
        console.log(`📄 Last output: ${output.slice(-500)}`); // Last 500 chars
        process.exit(1); // Exit immediately when we find the hanging test
      } else if (code === 0) {
        console.log(`✅ PASSED: ${testFile} (${duration}ms)`);
      } else {
        console.log(`❌ FAILED: ${testFile} (${duration}ms)`);
        console.log(`Error output: ${errorOutput.slice(-500)}`);
      }
    });

    vitest.on('error', (err) => {
      console.log(`🚨 SPAWN ERROR for ${testFile}:`, err.message);
    });

  } catch (error) {
    console.log(`🚨 ERROR running ${testFile}:`, error.message);
  }
}

console.log('\n✅ All tests completed without hanging!');

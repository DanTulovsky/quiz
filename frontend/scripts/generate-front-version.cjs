// Script to generate dist/meta/version.json after build (CommonJS)
const fs = require('fs');
const path = require('path');

function main() {
    const outDir = path.resolve(__dirname, '..', 'dist', 'meta');
    try {
        fs.mkdirSync(outDir, { recursive: true });
    } catch (e) { }

    const version = process.env.VITE_APP_VERSION || process.env.APP_VERSION || 'dev';
    const buildTime = process.env.VITE_BUILD_TIME || new Date().toISOString();
    const commit = (process.env.VITE_COMMIT_HASH || process.env.COMMIT_HASH || 'dev').substring(0, 8);

    const payload = {
        frontend: {
            version,
            commitHash: commit,
            buildTime,
        },
    };

    fs.writeFileSync(path.join(outDir, 'version.json'), JSON.stringify(payload, null, 2));
    console.log('Wrote', path.join(outDir, 'version.json'));
}

if (require.main === module) main();



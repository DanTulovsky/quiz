const fs = require('fs');

// Read the file
let content = fs.readFileSync('api_comprehensive.spec.ts', 'utf8');

// Replace all instances of the old logging pattern with the new helper function
const oldPattern = /const response = await request\[testCase\.method\.toLowerCase\(\)\]\(url\.toString\(\), requestOptions\);\s*\n\s*\/\/ Check that we get one of the expected status codes\s*\n\s*expect\(testCase\.expectedStatusCodes\)\.toContain\(response\.status\(\)\.toString\(\)\);\s*\n\s*\/\/ Log the test result\s*\n\s*console\.log\(`✅ \$\{testCase\.method\} \$\{testCase\.path\}: \$\{response\.status\(\)\}`\);/g;

const newPattern = `const response = await request[testCase.method.toLowerCase()](url.toString(), requestOptions);

        // Log the response details
        await logResponse(testCase, response, 'Regular User', 'apitestuser');`;

content = content.replace(oldPattern, newPattern);

// Write back to file
fs.writeFileSync('api_comprehensive.spec.ts', content);

console.log('Updated logging in api_comprehensive.spec.ts');

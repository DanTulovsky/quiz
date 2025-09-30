import {execSync} from 'child_process';
import path from 'path';
import {fileURLToPath} from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export function resetTestDatabase() {
  const backendDir = path.join(__dirname, '../../backend');
  const command = 'go run ./cmd/setup-test-db';

  console.log(`Running: ${command}`);
  console.log(`Working directory: ${backendDir}`);

  try {
    execSync(command, {
      cwd: backendDir,
      encoding: 'utf8',
      env: {
        ...process.env,
        DATABASE_URL: process.env.DATABASE_URL || 'postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable',
        QUIZ_CONFIG_FILE: process.env.QUIZ_CONFIG_FILE || path.join(__dirname, '../../merged.config.yaml'),
        MIGRATIONS_PATH: process.env.MIGRATIONS_PATH || 'file://migrations',
      },
    });
  } catch (error: unknown) {
    console.error('=== RESET DB COMMAND FAILED ===');
    console.error('Command:', command);
    console.error('Working directory:', backendDir);
    if (typeof error === 'object' && error !== null && 'status' in error) {
      console.error('Exit code:', (error as {status?: number}).status);
    }

    // Print stdout and stderr if available
    if (typeof error === 'object' && error !== null) {
      if ('stdout' in error && (error as {stdout?: Buffer | string}).stdout) {
        console.error('STDOUT:', (error as {stdout?: Buffer | string}).stdout?.toString());
      }
      if ('stderr' in error && (error as {stderr?: Buffer | string}).stderr) {
        console.error('STDERR:', (error as {stderr?: Buffer | string}).stderr?.toString());
      }
    }

    // Also check the output array
    if (typeof error === 'object' && error !== null && 'output' in error && Array.isArray((error as {output?: Array<Buffer | string>}).output)) {
      const outputArr = (error as {output?: Array<Buffer | string>}).output ?? [];
      if (outputArr.length > 0) {
        console.error('OUTPUT ARRAY:');
        outputArr.forEach((output, index) => {
          if (output) {
            console.error(`  [${index}]:`, output.toString());
          }
        });
      }
    }

    // Print the full error for debugging
    console.error('Full error:', error);
    throw error;
  }
}

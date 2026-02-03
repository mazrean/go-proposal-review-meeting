/**
 * Generate all OGP images by scanning the content directory
 * This script:
 * 1. Generates the home page OGP image
 * 2. Scans content/ directory for weekly content
 * 3. Generates OGP images for each week and proposal
 */

import { readdir, readFile } from 'fs/promises';
import { join } from 'path';
import { fileURLToPath } from 'url';
import { dirname } from 'path';
import { generateHomeImage, generateWeeklyImage, generateProposalImage } from './generate-ogp-image.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const contentDir = join(__dirname, '..', 'content');
const distDir = join(__dirname, '..', 'dist');
const publicDir = join(__dirname, '..', 'web', 'public');

/**
 * Parse front matter from a markdown file
 */
function parseFrontMatter(content) {
  const frontMatterMatch = content.match(/^---\n([\s\S]*?)\n---/);
  if (!frontMatterMatch) {
    return {};
  }

  const frontMatter = {};
  const lines = frontMatterMatch[1].split('\n');

  for (const line of lines) {
    const match = line.match(/^(\w+):\s*(.*)$/);
    if (match) {
      const [, key, value] = match;
      // Remove quotes from string values
      frontMatter[key] = value.replace(/^["']|["']$/g, '');
    }
  }

  return frontMatter;
}

/**
 * Generate OGP images for a specific week
 */
async function generateWeekOGPImages(year, weekStr) {
  const week = parseInt(weekStr.replace(/^W0?/, ''));
  const weekDir = join(contentDir, year, weekStr);
  const outputDir = join(distDir, year, `w${String(week).padStart(2, '0')}`);

  // Read all proposal files
  const files = await readdir(weekDir);
  const proposalFiles = files.filter(f => f.startsWith('proposal-') && f.endsWith('.md'));

  console.log(`\n📅 ${year}-W${String(week).padStart(2, '0')} (${proposalFiles.length} proposals)`);

  // Generate weekly OGP image
  const weeklyOGPPath = join(outputDir, 'ogp.png');
  const weeklySize = await generateWeeklyImage(
    parseInt(year),
    week,
    proposalFiles.length,
    weeklyOGPPath
  );
  console.log(`  ✓ Weekly: ${weeklyOGPPath} (${(weeklySize / 1024).toFixed(2)} KB)`);

  // Generate OGP images for each proposal
  for (const file of proposalFiles) {
    const issueMatch = file.match(/proposal-(\d+)\.md$/);
    if (!issueMatch) continue;

    const issueNumber = parseInt(issueMatch[1]);
    const content = await readFile(join(weekDir, file), 'utf-8');
    const frontMatter = parseFrontMatter(content);
    const title = frontMatter.title || `Proposal #${issueNumber}`;
    const previousStatus = frontMatter.previous_status || '';
    const currentStatus = frontMatter.current_status || '';

    const proposalOGPPath = join(outputDir, `${issueNumber}-ogp.png`);
    const proposalSize = await generateProposalImage(
      issueNumber,
      title,
      proposalOGPPath,
      previousStatus,
      currentStatus
    );
    console.log(`  ✓ #${issueNumber}: ${(proposalSize / 1024).toFixed(2)} KB`);
  }
}

/**
 * Main function
 */
async function main() {
  console.log('=== Generating All OGP Images ===\n');

  // Generate home page OGP image
  console.log('🏠 Home page');
  const homeOGPPath = join(publicDir, 'ogp.png');
  const homeSize = await generateHomeImage(homeOGPPath);
  console.log(`  ✓ Home: ${homeOGPPath} (${(homeSize / 1024).toFixed(2)} KB)`);

  // Scan content directory for years
  const years = await readdir(contentDir);

  for (const year of years) {
    // Skip non-directory entries
    if (year === 'state.json' || !year.match(/^\d{4}$/)) {
      continue;
    }

    const yearDir = join(contentDir, year);
    const weeks = await readdir(yearDir);

    for (const week of weeks) {
      // Process only week directories (W01, W02, etc.)
      if (!week.match(/^W\d{2}$/)) {
        continue;
      }

      await generateWeekOGPImages(year, week);
    }
  }

  console.log('\n✅ All OGP images generated successfully!\n');
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(error => {
    console.error('❌ Error:', error.message);
    process.exit(1);
  });
}

export { main as generateAllOGPImages };

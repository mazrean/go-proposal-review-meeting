/**
 * Generate OGP image for social media sharing
 * This script creates a 1200x630px image for Open Graph Protocol
 *
 * Features:
 * - Automatically downloads Noto Sans JP font if not present
 * - Works in any environment with Node.js installed
 * - No external dependencies beyond npm packages
 */

import { createCanvas, registerFont } from 'canvas';
import { writeFile, mkdir, access } from 'fs/promises';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { constants } from 'fs';

const __dirname = dirname(fileURLToPath(import.meta.url));

// OGP image dimensions (recommended by Facebook/Twitter)
const WIDTH = 1200;
const HEIGHT = 630;

// Go brand colors
const GO_BLUE = '#00ADD8';
const GO_BLUE_DARK = '#007d9c';
const WHITE = '#ffffff';
const TEXT_PRIMARY = '#202224';

// Font configuration
const FONT_URL = 'https://github.com/google/fonts/raw/main/ofl/notosansjp/NotoSansJP%5Bwght%5D.ttf';
const FONT_DIR = join(__dirname, '..', 'web', 'fonts');
const FONT_PATH = join(FONT_DIR, 'NotoSansJP.ttf');

/**
 * Check if a file exists
 */
async function fileExists(path) {
  try {
    await access(path, constants.F_OK);
    return true;
  } catch {
    return false;
  }
}

/**
 * Download the Noto Sans JP font from Google Fonts repository
 */
async function downloadFont() {
  console.log('Downloading Noto Sans JP font...');

  try {
    // Create fonts directory if it doesn't exist
    await mkdir(FONT_DIR, { recursive: true });

    // Download font file
    const response = await fetch(FONT_URL);
    if (!response.ok) {
      throw new Error(`Failed to download font: HTTP ${response.status}`);
    }

    const buffer = Buffer.from(await response.arrayBuffer());
    await writeFile(FONT_PATH, buffer);

    console.log(`✓ Font downloaded successfully: ${FONT_PATH}`);
    console.log(`  Size: ${(buffer.length / 1024 / 1024).toFixed(2)} MB`);
  } catch (error) {
    throw new Error(`Font download failed: ${error.message}`);
  }
}

/**
 * Ensure the Japanese font is available and register it
 */
async function ensureFont() {
  // Check if font already exists
  if (!(await fileExists(FONT_PATH))) {
    await downloadFont();
  } else {
    console.log(`✓ Font already exists: ${FONT_PATH}`);
  }

  // Register the font with canvas
  try {
    registerFont(FONT_PATH, { family: 'Noto Sans JP' });
    console.log('✓ Japanese font registered successfully');
  } catch (error) {
    throw new Error(`Failed to register font: ${error.message}`);
  }
}

async function generateOGPImage() {
  const canvas = createCanvas(WIDTH, HEIGHT);
  const ctx = canvas.getContext('2d');

  // Background gradient
  const gradient = ctx.createLinearGradient(0, 0, WIDTH, HEIGHT);
  gradient.addColorStop(0, GO_BLUE);
  gradient.addColorStop(1, GO_BLUE_DARK);
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, WIDTH, HEIGHT);

  // White content area
  const margin = 60;
  const contentX = margin;
  const contentY = margin;
  const contentWidth = WIDTH - margin * 2;
  const contentHeight = HEIGHT - margin * 2;

  ctx.fillStyle = WHITE;
  ctx.fillRect(contentX, contentY, contentWidth, contentHeight);

  // Draw Go gopher-like accent (simplified geometric shape)
  ctx.fillStyle = GO_BLUE;

  // Draw circles as decoration
  const circleRadius = 40;
  ctx.beginPath();
  ctx.arc(WIDTH - 120, 100, circleRadius, 0, Math.PI * 2);
  ctx.fill();

  ctx.beginPath();
  ctx.arc(WIDTH - 180, 140, circleRadius * 0.6, 0, Math.PI * 2);
  ctx.fill();

  // Title text
  ctx.fillStyle = TEXT_PRIMARY;
  ctx.font = 'bold 72px "Noto Sans JP", sans-serif';
  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';

  const titleY = contentY + 80;
  ctx.fillText('Go Proposal', contentX + 60, titleY);
  ctx.fillText('Weekly Digest', contentX + 60, titleY + 90);

  // Subtitle (Japanese text)
  ctx.font = '32px "Noto Sans JP", sans-serif';
  ctx.fillStyle = GO_BLUE_DARK;
  ctx.fillText('最新のGo言語プロポーザルを週次でお届け', contentX + 60, titleY + 220);

  // Footer accent line
  ctx.fillStyle = GO_BLUE;
  ctx.fillRect(contentX + 60, HEIGHT - margin - 80, 200, 8);

  // Logo text
  ctx.font = 'bold 28px "Noto Sans JP", sans-serif';
  ctx.fillStyle = GO_BLUE_DARK;
  ctx.fillText('golang/go', contentX + 60, HEIGHT - margin - 50);

  // Save the image
  const outputPath = join(__dirname, '..', 'web', 'public', 'ogp.png');
  const buffer = canvas.toBuffer('image/png');
  await writeFile(outputPath, buffer);

  console.log(`✓ OGP image generated: ${outputPath}`);
  console.log(`  Size: ${WIDTH}x${HEIGHT}px`);
  console.log(`  File size: ${(buffer.length / 1024).toFixed(2)} KB`);
}

/**
 * Main execution
 */
async function main() {
  console.log('=== OGP Image Generator ===\n');

  try {
    // Step 1: Ensure font is available
    await ensureFont();
    console.log();

    // Step 2: Generate OGP image
    await generateOGPImage();
    console.log();

    console.log('✓ All done!');
  } catch (error) {
    console.error('✗ Error:', error.message);
    process.exit(1);
  }
}

main();

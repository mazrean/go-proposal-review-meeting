/**
 * Generate OGP images for social media sharing
 * This script creates 1200x630px images for Open Graph Protocol
 *
 * Features:
 * - Automatically downloads Noto Sans JP font if not present
 * - Works in any environment with Node.js installed
 * - Supports multiple image types (home, weekly, proposal)
 * - Can be called from command line or as a module
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

let fontInitialized = false;

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
    await mkdir(FONT_DIR, { recursive: true });

    const response = await fetch(FONT_URL);
    if (!response.ok) {
      throw new Error(`Failed to download font: HTTP ${response.status}`);
    }

    const buffer = Buffer.from(await response.arrayBuffer());
    await writeFile(FONT_PATH, buffer);

    console.log(`✓ Font downloaded: ${(buffer.length / 1024 / 1024).toFixed(2)} MB`);
  } catch (error) {
    throw new Error(`Font download failed: ${error.message}`);
  }
}

/**
 * Ensure the Japanese font is available and register it
 */
async function ensureFont() {
  if (fontInitialized) return;

  if (!(await fileExists(FONT_PATH))) {
    await downloadFont();
  }

  try {
    registerFont(FONT_PATH, { family: 'Noto Sans JP' });
    fontInitialized = true;
  } catch (error) {
    throw new Error(`Failed to register font: ${error.message}`);
  }
}

/**
 * Wrap text to fit within a maximum width
 */
function wrapText(ctx, text, maxWidth) {
  const words = text.split('');
  const lines = [];
  let currentLine = '';

  for (const char of words) {
    const testLine = currentLine + char;
    const metrics = ctx.measureText(testLine);

    if (metrics.width > maxWidth && currentLine.length > 0) {
      lines.push(currentLine);
      currentLine = char;
    } else {
      currentLine = testLine;
    }
  }

  if (currentLine) {
    lines.push(currentLine);
  }

  return lines;
}

/**
 * Create base canvas with gradient background and white content area
 */
function createBaseCanvas() {
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
  ctx.fillStyle = WHITE;
  ctx.fillRect(margin, margin, WIDTH - margin * 2, HEIGHT - margin * 2);

  // Decorative circles
  ctx.fillStyle = GO_BLUE;
  ctx.beginPath();
  ctx.arc(WIDTH - 120, 100, 40, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.arc(WIDTH - 180, 140, 24, 0, Math.PI * 2);
  ctx.fill();

  return { canvas, ctx, margin };
}

/**
 * Add footer to canvas
 */
function addFooter(ctx, margin) {
  // Footer accent line
  ctx.fillStyle = GO_BLUE;
  ctx.fillRect(margin + 60, HEIGHT - margin - 80, 200, 8);

  // Logo text
  ctx.font = 'bold 28px "Noto Sans JP", sans-serif';
  ctx.fillStyle = GO_BLUE_DARK;
  ctx.fillText('golang/go', margin + 60, HEIGHT - margin - 50);
}

/**
 * Generate home page OGP image
 */
async function generateHomeImage(outputPath) {
  await ensureFont();

  const { canvas, ctx, margin } = createBaseCanvas();

  // Title
  ctx.fillStyle = TEXT_PRIMARY;
  ctx.font = 'bold 72px "Noto Sans JP", sans-serif';
  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';

  const titleY = margin + 80;
  ctx.fillText('Go Proposal', margin + 60, titleY);
  ctx.fillText('Weekly Digest', margin + 60, titleY + 90);

  // Subtitle
  ctx.font = '32px "Noto Sans JP", sans-serif';
  ctx.fillStyle = GO_BLUE_DARK;
  ctx.fillText('最新のGo言語プロポーザルを週次でお届け', margin + 60, titleY + 220);

  addFooter(ctx, margin);

  const buffer = canvas.toBuffer('image/png');
  await mkdir(dirname(outputPath), { recursive: true });
  await writeFile(outputPath, buffer);

  return buffer.length;
}

/**
 * Generate weekly page OGP image
 */
async function generateWeeklyImage(year, week, proposalCount, outputPath) {
  await ensureFont();

  const { canvas, ctx, margin } = createBaseCanvas();

  // Week badge
  ctx.fillStyle = GO_BLUE;
  ctx.fillRect(margin + 60, margin + 70, 140, 140);

  ctx.fillStyle = WHITE;
  ctx.font = 'bold 48px "Noto Sans JP", sans-serif';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillText(`W${String(week).padStart(2, '0')}`, margin + 130, margin + 110);
  ctx.font = 'bold 32px "Noto Sans JP", sans-serif';
  ctx.fillText(String(year), margin + 130, margin + 160);

  // Title
  ctx.fillStyle = TEXT_PRIMARY;
  ctx.font = 'bold 56px "Noto Sans JP", sans-serif';
  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';

  const titleY = margin + 80;
  ctx.fillText(`${year}年 第${week}週`, margin + 230, titleY);

  // Proposal count
  ctx.font = '36px "Noto Sans JP", sans-serif';
  ctx.fillStyle = GO_BLUE_DARK;
  ctx.fillText(`${proposalCount}件のProposal更新`, margin + 230, titleY + 90);

  // Site name
  ctx.font = 'bold 28px "Noto Sans JP", sans-serif';
  ctx.fillText('Go Proposal Weekly Digest', margin + 60, margin + 270);

  addFooter(ctx, margin);

  const buffer = canvas.toBuffer('image/png');
  await mkdir(dirname(outputPath), { recursive: true });
  await writeFile(outputPath, buffer);

  return buffer.length;
}

/**
 * Get colors for a proposal status (background and text)
 */
function getStatusColors(status) {
  const statusLower = status.toLowerCase();
  switch (statusLower) {
    case 'accepted':
      return { bg: '#dcfce7', text: '#166534' }; // green-100 / green-800
    case 'declined':
      return { bg: '#fee2e2', text: '#991b1b' }; // red-100 / red-800
    case 'likely-accept':
    case 'likely_accept':
      return { bg: '#d1fae5', text: '#065f46' }; // emerald-100 / emerald-800
    case 'likely-decline':
    case 'likely_decline':
      return { bg: '#ffedd5', text: '#9a3412' }; // orange-100 / orange-800
    case 'hold':
      return { bg: '#fef9c3', text: '#854d0e' }; // yellow-100 / yellow-800
    case 'active':
      return { bg: '#e0f2fe', text: '#075985' }; // sky-100 / sky-800
    case 'discussions':
      return { bg: '#f3e8ff', text: '#6b21a8' }; // purple-100 / purple-800
    default:
      return { bg: '#f3f4f6', text: '#6b7280' }; // gray-100 / gray-600
  }
}

/**
 * Draw a rounded rectangle (manual implementation for compatibility)
 */
function drawRoundedRect(ctx, x, y, width, height, radius) {
  ctx.beginPath();
  ctx.moveTo(x + radius, y);
  ctx.lineTo(x + width - radius, y);
  ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
  ctx.lineTo(x + width, y + height - radius);
  ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
  ctx.lineTo(x + radius, y + height);
  ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
  ctx.lineTo(x, y + radius);
  ctx.quadraticCurveTo(x, y, x + radius, y);
  ctx.closePath();
}

/**
 * Draw a status badge with background
 */
function drawStatusBadge(ctx, text, x, y, isSpecial = false) {
  const padding = 14;
  const height = 44;
  const borderRadius = 8;

  ctx.font = 'bold 28px "Noto Sans JP", sans-serif';
  const textWidth = ctx.measureText(text).width;
  const badgeWidth = textWidth + padding * 2;

  // Calculate vertical positions for centered alignment
  const bgY = y - height / 2;
  const textY = y;

  // Draw background with rounded corners
  if (isSpecial) {
    // Special badge (new proposal) - solid blue background
    ctx.fillStyle = '#3b82f6';
  } else {
    // Regular status badge - colored background
    const colors = getStatusColors(text);
    ctx.fillStyle = colors.bg;
  }

  drawRoundedRect(ctx, x, bgY, badgeWidth, height, borderRadius);
  ctx.fill();

  // Draw text (centered vertically in badge) with outline for better readability
  ctx.textBaseline = 'middle';
  if (isSpecial) {
    ctx.fillStyle = WHITE;
  } else {
    const colors = getStatusColors(text);
    ctx.fillStyle = colors.text;
  }

  // Draw text with slight stroke for better readability
  ctx.lineWidth = 0.5;
  ctx.strokeStyle = ctx.fillStyle;
  ctx.strokeText(text, x + padding, textY);
  ctx.fillText(text, x + padding, textY);

  return badgeWidth;
}

/**
 * Generate proposal detail OGP image
 */
async function generateProposalImage(issueNumber, title, outputPath, previousStatus = '', currentStatus = '') {
  await ensureFont();

  const { canvas, ctx, margin } = createBaseCanvas();

  // Issue number badge
  ctx.fillStyle = GO_BLUE;
  ctx.font = 'bold 48px "Noto Sans JP", sans-serif';
  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  ctx.fillText(`#${issueNumber}`, margin + 60, margin + 70);

  // Title (wrapped)
  ctx.fillStyle = TEXT_PRIMARY;
  ctx.font = 'bold 42px "Noto Sans JP", sans-serif';
  const maxWidth = WIDTH - margin * 2 - 120;
  const titleLines = wrapText(ctx, title, maxWidth);

  let titleY = margin + 140;
  const maxLines = 4;
  const lineHeight = 58;

  for (let i = 0; i < Math.min(titleLines.length, maxLines); i++) {
    let line = titleLines[i];
    if (i === maxLines - 1 && titleLines.length > maxLines) {
      line = line.substring(0, line.length - 3) + '...';
    }
    ctx.fillText(line, margin + 60, titleY);
    titleY += lineHeight;
  }

  // Status badge (below title)
  if (currentStatus) {
    const statusY = titleY + 35;
    let currentX = margin + 60;

    if (!previousStatus) {
      // New proposal: show badge + current status
      const newBadgeWidth = drawStatusBadge(ctx, '🆕 新規提案', currentX, statusY, true);
      currentX += newBadgeWidth + 12;

      drawStatusBadge(ctx, currentStatus, currentX, statusY, false);
    } else if (previousStatus !== currentStatus) {
      // Status changed: previous → current with gray arrow
      const prevBadgeWidth = drawStatusBadge(ctx, previousStatus, currentX, statusY, false);
      currentX += prevBadgeWidth + 10;

      // Arrow
      ctx.font = 'bold 26px "Noto Sans JP", sans-serif';
      ctx.fillStyle = '#9ca3af'; // gray-400 for arrow
      ctx.fillText('→', currentX, statusY);
      currentX += ctx.measureText('→').width + 10;

      drawStatusBadge(ctx, currentStatus, currentX, statusY, false);
    } else {
      // Status unchanged
      drawStatusBadge(ctx, currentStatus, currentX, statusY, false);
    }
  }

  // Site name
  ctx.font = 'bold 28px "Noto Sans JP", sans-serif';
  ctx.fillStyle = GO_BLUE_DARK;
  ctx.fillText('Go Proposal Weekly Digest', margin + 60, HEIGHT - margin - 120);

  addFooter(ctx, margin);

  const buffer = canvas.toBuffer('image/png');
  await mkdir(dirname(outputPath), { recursive: true });
  await writeFile(outputPath, buffer);

  return buffer.length;
}

/**
 * CLI interface
 */
async function main() {
  const args = process.argv.slice(2);

  if (args.length === 0) {
    // Default: generate home page image
    console.log('=== OGP Image Generator ===\n');
    console.log('Generating home page image...');
    const outputPath = join(__dirname, '..', 'web', 'public', 'ogp.png');
    const size = await generateHomeImage(outputPath);
    console.log(`✓ Generated: ${outputPath}`);
    console.log(`  Size: ${(size / 1024).toFixed(2)} KB\n`);
    return;
  }

  const command = args[0];

  try {
    if (command === 'home') {
      const outputPath = args[1] || join(__dirname, '..', 'web', 'public', 'ogp.png');
      const size = await generateHomeImage(outputPath);
      console.log(`✓ Home: ${(size / 1024).toFixed(2)} KB`);
    } else if (command === 'weekly') {
      const [year, week, count, outputPath] = args.slice(1);
      const size = await generateWeeklyImage(
        parseInt(year),
        parseInt(week),
        parseInt(count),
        outputPath
      );
      console.log(`✓ Weekly ${year}-W${week}: ${(size / 1024).toFixed(2)} KB`);
    } else if (command === 'proposal') {
      const [issueNumber, title, outputPath, previousStatus, currentStatus] = args.slice(1);
      const size = await generateProposalImage(
        parseInt(issueNumber),
        title,
        outputPath,
        previousStatus || '',
        currentStatus || ''
      );
      console.log(`✓ Proposal #${issueNumber}: ${(size / 1024).toFixed(2)} KB`);
    } else {
      console.error('Unknown command:', command);
      console.error('Usage:');
      console.error('  node generate-ogp-image.js');
      console.error('  node generate-ogp-image.js home <output>');
      console.error('  node generate-ogp-image.js weekly <year> <week> <count> <output>');
      console.error('  node generate-ogp-image.js proposal <issue> <title> <output> [previousStatus] [currentStatus]');
      process.exit(1);
    }
  } catch (error) {
    console.error('✗ Error:', error.message);
    process.exit(1);
  }
}

// Export functions for use as module
export { generateHomeImage, generateWeeklyImage, generateProposalImage, ensureFont };

// Run CLI if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

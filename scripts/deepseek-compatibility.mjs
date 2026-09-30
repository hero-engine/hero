#!/usr/bin/env node
// Opt-in: node scripts/deepseek-compatibility.mjs --harness /path/to/deepseek-harness --hero /path/to/hero
// Requires the pinned harness checkout with `pnpm install --frozen-lockfile` completed.
import assert from 'node:assert/strict';
import { execFileSync, spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, readFileSync, readdirSync, rmSync, symlinkSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { parseArgs } from 'node:util';

const { values } = parseArgs({ options: {
  harness: { type: 'string' }, hero: { type: 'string' }, domain: { type: 'string', default: 'engineering' },
} });
assert(values.harness && values.hero, 'Required: --harness /path/to/deepseek-harness --hero /path/to/built/hero');
const harness = resolve(values.harness);
const hero = resolve(values.hero);
const domain = values.domain;
const pinned = '477b4f420553e8a52c2fbccc464d7561b239c443';
const revision = execFileSync('git', ['-C', harness, 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim();
assert.equal(revision, pinned, 'Harness revision changed; review loader contracts before updating the compatibility baseline');

if (process.env.HERO_DEEPSEEK_COMPAT_CHILD !== '1') {
  const result = spawnSync(process.execPath, [
    '--import', join(harness, 'node_modules/tsx/dist/esm/index.mjs'),
    fileURLToPath(import.meta.url), ...process.argv.slice(2),
  ], {
    stdio: 'inherit', timeout: 120_000,
    env: { ...process.env, HERO_DEEPSEEK_COMPAT_CHILD: '1', TSX_TSCONFIG_PATH: join(harness, 'tsconfig.json') },
  });
  if (result.error) throw result.error;
  process.exit(result.status ?? 1);
}

const source = relative => import(pathToFileURL(join(harness, relative)).href);
const scratch = mkdtempSync(join(tmpdir(), 'hero-deepseek-compat-'));
const workspace = join(scratch, 'project with spaces');
const home = join(scratch, 'home');
const dshHome = join(home, '.dsh');
const bin = join(scratch, 'bin');
for (const dir of [workspace, home, dshHome, bin]) mkdirSync(dir, { recursive: true });
mkdirSync(join(workspace, '.git'));
symlinkSync(hero, join(bin, 'hero'));
// Isolate host configuration and select the supplied binary without rewriting the portable overlay.
process.env.HOME = home;
process.env.DSH_HOME = dshHome;
process.env.DSH_AGENTS_HOME = join(home, '.agents');
delete process.env.DSH_BUNDLED_SKILL_DIR;
process.env.PATH = bin + ':' + process.env.PATH;
process.chdir(workspace);
let ctx;
let provider;
try {
  execFileSync(hero, ['init', '--domain', domain, '--no-agents', '--no-hooks'], { cwd: workspace, encoding: 'utf8', timeout: 30_000 });
  const installOutput = execFileSync(hero, [
    'install', 'project', workspace, '--domain', domain, '--target', 'deepseek', '--only-target', '--no-hooks', '--json',
  ], { cwd: workspace, encoding: 'utf8', timeout: 30_000 });
  JSON.parse(installOutput);
  const { Context } = await source('vendor/cordis/src/index.ts');
  const { FileSystemSkillProvider } = await source('packages/skill/skill-filesystem/src/index.ts');
  const { loadBaselineInstructionSet } = await source('packages/context/agent-instructions/src/files.ts');
  ctx = new Context();
  provider = new FileSystemSkillProvider(ctx, {
    invalidate() {}, signal: new AbortController().signal,
  }, { watch: false, dshHome, agentsHome: process.env.DSH_AGENTS_HOME });
  const discovered = await provider.list({ cwd: workspace });
  const candidates = Array.isArray(discovered) ? discovered : discovered.candidates;
  if (!Array.isArray(discovered)) assert(discovered.complete, 'Native skill discovery was incomplete');
  const generated = readdirSync(join(workspace, '.dsh/skills'));
  assert.equal(candidates.length, generated.length, 'Every generated skill must be discovered');
  for (const candidate of candidates) {
    const skill = await provider.get(candidate, { cwd: workspace });
    assert(skill?.name && skill.description && skill.content, 'Native parser rejected a generated skill');
    if (skill.name.startsWith('role-')) {
      assert.equal(skill.invocation.userInvocable, false, 'Roles must not advertise user commands');
      assert.equal(skill.invocation.modelInvocable, true, 'Roles must remain available to the model');
    }
    const raw = readFileSync(skill.path, 'utf8');
    assert(raw.includes(skill.content.trim()), `Native loader altered ${skill.name} content`);
  }
  assert(candidates.some(skill => skill.name.startsWith('command-')));
  assert(candidates.some(skill => skill.name.startsWith('role-')));
  assert(candidates.some(skill => !/^(command|role)-/.test(skill.name)));
  const nested = join(workspace, 'nested');
  mkdirSync(nested);
  const nestedDiscovery = await provider.list({ cwd: nested });
  const nestedCandidates = Array.isArray(nestedDiscovery) ? nestedDiscovery : nestedDiscovery.candidates;
  assert.deepEqual(nestedCandidates.map(skill => skill.name), candidates.map(skill => skill.name), 'Nested cwd must discover Git-root skills');
  const instructions = await loadBaselineInstructionSet({ cwd: workspace, dshHome, maxBytes: 1_000_000 });
  assert(instructions?.included.some(file => file.absolutePath === join(workspace, 'AGENTS.md') && file.content.includes('DeepSeek')));
  console.log(`PASS native loaders (${domain}): ${candidates.length} skills (canonical, workflows, roles), AGENTS.md`);

  const { SkillRegistry } = await source('packages/skill/skill/src/index.ts');
  await ctx.plugin(SkillRegistry);
  const duplicate = candidates[0].name;
  for (const root of [join(workspace, '.agents/skills'), join(dshHome, 'skills')]) {
    mkdirSync(join(root, duplicate), { recursive: true });
    writeFileSync(join(root, duplicate, 'SKILL.md'), `---\nname: ${duplicate}\ndescription: Lower priority fixture\n---\nLower priority body.\n`);
  }
  ctx.skills.registerProvider(() => provider);
  const winner = await ctx.skills.get(duplicate, { cwd: nested });
  assert.equal(winner?.source, 'project-dsh', 'Project .dsh must win collisions with .agents and global skills');
  console.log('PASS native registry precedence and Git-root discovery from nested cwd');

  const { loadProfile, composeEntries } = await source('packages/boot/app-boot/src/profile.ts');
  const { loadOverlayPatches } = await source('packages/boot/app-boot/src/index.ts');
  const overlayPath = join(workspace, '.dsh/hero.cordis.patch.yml');
  const patches = loadOverlayPatches('hero-compatibility', overlayPath);
  const flatten = rows => rows.flatMap(row => [row, ...(row.group && Array.isArray(row.config) ? flatten(row.config) : [])]);
  // web is the shipped interactive profile Hero's launch guidance names; headless is the scripted path.
  let clients;
  for (const name of ['web', 'headless']) {
    const profile = loadProfile('dsh', name, join(harness, 'apps/cli/package.json'), dshHome);
    const warnings = [];
    const entries = composeEntries([...profile.layers.map(layer => layer.patches), profile.patches, patches], warning => warnings.push(warning));
    assert.deepEqual(warnings, [], `Cordis composition skipped a patch (${name})`);
    clients = flatten(entries).filter(row => row.name === '@deepseek-ai/dsh-mcp-client' && row.config?.serverName === 'hero');
    assert.equal(clients.length, 1, `Composed ${name} profile must contain exactly one Hero MCP client`);
    console.log(`PASS Cordis composition: ${name} profile (${profile.layers.length} bundle layers) plus generated overlay`);
  }

  // Exercise only the real MCP service dependencies: no agent loop, provider, or paid model request.
  const { default: SystemPrompt } = await source('packages/core/system-prompt/src/index.ts');
  const { default: Tools } = await source('packages/core/tools/src/index.ts');
  const McpClient = await source('packages/mcp/mcp-client/src/index.ts');
  await ctx.plugin(SystemPrompt);
  await ctx.plugin(Tools);
  process.chdir(nested);
  const config = McpClient.Config(clients[0].config);
  const fiber = await ctx.plugin(McpClient, config);
  const names = ctx.tools.schemas().map(tool => tool.name).filter(name => name.startsWith('mcp__hero__'));
  assert(names.length > 0, `Hero initialization/tool listing failed; MCP plugin state: ${fiber.state}`);
  assert(names.some(name => name.includes('hero_status')), 'Hero status tool missing from native MCP registry');
  console.log(`PASS native MCP client: initialize + tools/list, ${names.length} Hero tools registered`);
  const { ToolCallId } = await source('packages/llm/llm/src/index.ts');
  const status = await ctx.tools.execute({
    name: 'mcp__hero__hero_status', arguments: {},
    callId: ToolCallId('hero-compatibility-status'), signal: AbortSignal.timeout(15_000),
  });
  assert.equal(status.isError, false, 'Hero status failed through native MCP tool execution');
  console.log('PASS native MCP tool execution: hero_status from nested workspace cwd');
  await fiber.dispose();
  execFileSync(hero, [
    'install', 'project', workspace, '--domain', domain, '--target', 'deepseek', '--only-target',
    '--no-hooks', '--json', '--workspace', nested,
  ], { cwd: workspace, encoding: 'utf8', timeout: 30_000 });
  const boundPatches = loadOverlayPatches('hero-compatibility', join(nested, '.dsh/hero.cordis.patch.yml'));
  const boundEntries = composeEntries([boundPatches], warning => { throw new Error(warning); });
  const boundClient = flatten(boundEntries).find(row => row.name === '@deepseek-ai/dsh-mcp-client');
  assert(boundClient, 'Workspace overlay must include the MCP client');
  assert.deepEqual(boundClient.config.args, ['mcp', '--project-root', workspace]);
  // A launch outside either workspace must still connect to the explicitly bound project.
  process.chdir(scratch);
  await ctx.plugin(McpClient, McpClient.Config(boundClient.config));
  const boundStatus = await ctx.tools.execute({
    name: 'mcp__hero__hero_status', arguments: {},
    callId: ToolCallId('hero-compatibility-bound-status'), signal: AbortSignal.timeout(15_000),
  });
  assert.equal(boundStatus.isError, false, 'Workspace-bound overlay failed outside the project');
  console.log('PASS native MCP workspace binding: generated subdirectory overlay resolves explicit project root');
  console.log(JSON.stringify({ harnessRevision: revision, heroVersion: execFileSync(hero, ['--version'], { encoding: 'utf8' }).trim(), domain, skills: candidates.length, tools: names.length, modelCalls: 0 }));
} finally {
  await provider?.dispose();
  await ctx?.fiber.dispose();
  process.chdir(dirname(fileURLToPath(import.meta.url)));
  rmSync(scratch, { recursive: true, force: true });
}

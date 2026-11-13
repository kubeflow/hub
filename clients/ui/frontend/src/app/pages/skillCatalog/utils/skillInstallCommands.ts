import type { Skill } from '~/app/skillCatalogTypes';

// The bare `npx skills add <repo-url>` form installs every skill the repository
// carries. `--skill <name>` scopes the install to the one skill the user picked in
// the catalog, matching the per-skill install offered by the Claude Code and Manual
// tabs. skill.name is the leaf identifier the CLI expects (e.g. "find-skills"), not
// the full in-repo path.
export const buildNpxCommand = (skill: Skill): string => {
  const repository = skill.repository ?? skill.repositoryUrl ?? '';
  return `npx skills add ${repository} --skill ${skill.name}`;
};

// `git clone <repository>` creates a local directory named after the repository, not the
// URL itself — this recovers that local directory name (last path segment, minus ".git")
// so the cp source in buildManualCommand refers to a real local path.
const getCloneDirName = (repository: string): string => {
  const withoutTrailingSlash = repository.replace(/\/+$/, '');
  const lastSegment = withoutTrailingSlash.split('/').pop() ?? withoutTrailingSlash;
  return lastSegment.replace(/\.git$/, '');
};

export const buildManualCommand = (skill: Skill): string => {
  const repository = skill.repository ?? skill.repositoryUrl ?? '';
  const cloneDir = getCloneDirName(repository);
  const installDir = `~/.claude/skills/${skill.name}`;
  return [
    `git clone ${repository}`,
    `cp -r ${cloneDir}/${skill.path ?? skill.name} ${installDir}`,
  ].join('\n');
};

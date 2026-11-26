import type { Skill } from '~/app/skillCatalogTypes';

export type FileTreeNode = {
  id: string;
  name: string;
  /** Present on leaf (file) nodes — clicking the row opens this URL. */
  url?: string;
  children?: FileTreeNode[];
};

const stripGitSuffix = (url: string): string => url.replace(/\.git$/, '');

export const getSupportingFileUrl = (skill: Skill, filePath: string): string | undefined => {
  if (!skill.repository) {
    return undefined;
  }
  const ref = skill.resolvedCommit || skill.version;
  if (!ref) {
    return undefined;
  }
  return `${stripGitSuffix(skill.repository)}/blob/${ref}/${filePath}`;
};

type MutableNode = {
  name: string;
  relativePath: string;
  url?: string;
  children: Map<string, MutableNode>;
};

const toFileTreeNodes = (node: MutableNode): FileTreeNode[] =>
  Array.from(node.children.values()).map((child) => {
    const hasChildren = child.children.size > 0;
    return {
      id: child.relativePath,
      name: child.name,
      url: hasChildren ? undefined : child.url,
      children: hasChildren ? toFileTreeNodes(child) : undefined,
    };
  });

// Builds a compact file-browser tree from a skill's flat supportingFiles paths, rooted at
// the skill's own directory.
export const buildSupportingFilesTree = (skill: Skill): FileTreeNode | undefined => {
  if (!skill.supportingFiles || skill.supportingFiles.length === 0) {
    return undefined;
  }

  const rootPrefix = skill.path ? `${skill.path}/` : '';
  const root: MutableNode = { name: '', relativePath: '', children: new Map() };

  skill.supportingFiles.forEach((filePath) => {
    const relative = filePath.startsWith(rootPrefix) ? filePath.slice(rootPrefix.length) : filePath;
    const segments = relative.split('/').filter(Boolean);

    let node = root;
    let cumulativePath = '';
    segments.forEach((segment) => {
      cumulativePath = cumulativePath ? `${cumulativePath}/${segment}` : segment;
      let child = node.children.get(segment);
      if (!child) {
        child = { name: segment, relativePath: cumulativePath, children: new Map() };
        node.children.set(segment, child);
      }
      node = child;
    });

    node.url = getSupportingFileUrl(skill, filePath);
  });

  return {
    id: 'root',
    name: skill.path || skill.name,
    children: toFileTreeNodes(root),
  };
};

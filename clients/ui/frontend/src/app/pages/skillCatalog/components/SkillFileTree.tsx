import * as React from 'react';
import {
  AngleDownIcon,
  AngleRightIcon,
  FileIcon,
  FolderIcon,
  FolderOpenIcon,
} from '@patternfly/react-icons';
import type { FileTreeNode } from '~/app/pages/skillCatalog/utils/skillSupportingFilesTree';

const ROW_HEIGHT = '22px';
const INDENT_PER_DEPTH = 14;

type SkillFileTreeRowProps = {
  node: FileTreeNode;
  depth: number;
};

const SkillFileTreeRow: React.FC<SkillFileTreeRowProps> = ({ node, depth }) => {
  const [isExpanded, setIsExpanded] = React.useState(true);
  const hasChildren = !!node.children && node.children.length > 0;
  const isClickable = hasChildren || !!node.url;

  return (
    <>
      <div
        role={isClickable ? 'button' : undefined}
        tabIndex={isClickable ? 0 : undefined}
        data-testid={`skill-file-tree-row-${node.id}`}
        onClick={() => {
          if (hasChildren) {
            setIsExpanded((prev) => !prev);
          } else if (node.url) {
            window.open(node.url, '_blank', 'noopener,noreferrer');
          }
        }}
        onKeyDown={(event) => {
          if (event.key !== 'Enter' && event.key !== ' ') {
            return;
          }
          event.preventDefault();
          if (hasChildren) {
            setIsExpanded((prev) => !prev);
          } else if (node.url) {
            window.open(node.url, '_blank', 'noopener,noreferrer');
          }
        }}
        style={{
          display: 'flex',
          alignItems: 'center',
          height: ROW_HEIGHT,
          paddingLeft: `${depth * INDENT_PER_DEPTH}px`,
          fontSize: '13px',
          cursor: isClickable ? 'pointer' : 'default',
        }}
      >
        <span
          style={{
            display: 'inline-flex',
            width: '12px',
            marginRight: '2px',
            fontSize: '10px',
            flexShrink: 0,
          }}
        >
          {hasChildren && (isExpanded ? <AngleDownIcon /> : <AngleRightIcon />)}
        </span>
        <span
          style={{ display: 'inline-flex', marginRight: '4px', fontSize: '12px', flexShrink: 0 }}
        >
          {hasChildren ? (
            isExpanded ? (
              <FolderOpenIcon color="var(--pf-t--global--icon--color--regular)" />
            ) : (
              <FolderIcon color="var(--pf-t--global--icon--color--regular)" />
            )
          ) : (
            <FileIcon color="var(--pf-t--global--icon--color--subtle)" />
          )}
        </span>
        <span
          style={{
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            color: node.url ? 'var(--pf-t--global--text--color--link--default)' : undefined,
          }}
        >
          {node.name}
        </span>
      </div>
      {hasChildren &&
        isExpanded &&
        node.children?.map((child) => (
          <SkillFileTreeRow key={child.id} node={child} depth={depth + 1} />
        ))}
    </>
  );
};

type SkillFileTreeProps = {
  root: FileTreeNode;
};

const SkillFileTree: React.FC<SkillFileTreeProps> = ({ root }) => (
  <div data-testid="skill-file-tree">
    <SkillFileTreeRow node={root} depth={0} />
  </div>
);

export default SkillFileTree;

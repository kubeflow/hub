import * as React from 'react';
import { Tab, Tabs, TabTitleText } from '@patternfly/react-core';
import type { Skill } from '~/app/skillCatalogTypes';
import {
  buildManualCommand,
  buildNpxCommand,
} from '~/app/pages/skillCatalog/utils/skillInstallCommands';
import SkillInstallCodeBlock from '~/app/pages/skillCatalog/components/SkillInstallCodeBlock';
import SkillClaudeCodeInstallCommand from '~/app/pages/skillCatalog/components/SkillClaudeCodeInstallCommand';

type SkillInstallTabsProps = {
  skill: Skill;
  /** Prefixes every DOM/test id so two instances (e.g. the tile modal and the
   *  details page's inline card) can be mounted on the same page without collisions. */
  idPrefix: string;
};

enum InstallTab {
  NPX = 'npx',
  CLAUDE_CODE = 'claude-code',
  MANUAL = 'manual',
}

const SkillInstallTabs: React.FC<SkillInstallTabsProps> = ({ skill, idPrefix }) => {
  const [activeTab, setActiveTab] = React.useState<InstallTab>(InstallTab.NPX);

  return (
    <Tabs
      activeKey={activeTab}
      onSelect={(_event, key) => {
        const validTab = Object.values(InstallTab).find((t) => t === key);
        if (validTab) {
          setActiveTab(validTab);
        }
      }}
      mountOnEnter
      aria-label="Skill install options"
      data-testid={`${idPrefix}-tabs-${skill.id}`}
    >
      <Tab
        eventKey={InstallTab.NPX}
        title={<TabTitleText>npx</TabTitleText>}
        data-testid={`${idPrefix}-tab-npx`}
      >
        <SkillInstallCodeBlock
          id={`${idPrefix}-npx-${skill.id}`}
          content={buildNpxCommand(skill)}
        />
      </Tab>
      <Tab
        eventKey={InstallTab.CLAUDE_CODE}
        title={<TabTitleText>Claude Code</TabTitleText>}
        data-testid={`${idPrefix}-tab-claude-code`}
      >
        <SkillClaudeCodeInstallCommand skill={skill} idPrefix={idPrefix} />
      </Tab>
      <Tab
        eventKey={InstallTab.MANUAL}
        title={<TabTitleText>Manual</TabTitleText>}
        data-testid={`${idPrefix}-tab-manual`}
      >
        <SkillInstallCodeBlock
          id={`${idPrefix}-manual-${skill.id}`}
          content={buildManualCommand(skill)}
        />
      </Tab>
    </Tabs>
  );
};

export default SkillInstallTabs;

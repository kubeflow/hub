import * as React from 'react';
import {
  Flex,
  FlexItem,
  Label,
  Modal,
  ModalBody,
  ModalHeader,
  ModalVariant,
} from '@patternfly/react-core';
import type { Skill } from '~/app/skillCatalogTypes';
import { SKILL_TRUST_TIER_LABEL_MAPPING } from '~/app/pages/skillCatalog/const';
import SkillInstallTabs from '~/app/pages/skillCatalog/components/SkillInstallTabs';

type SkillInstallModalProps = {
  skill: Skill;
  isOpen: boolean;
  onClose: () => void;
};

const SkillInstallModal: React.FC<SkillInstallModalProps> = ({ skill, isOpen, onClose }) => {
  const skillName = skill.displayName || skill.name;

  return (
    <Modal
      variant={ModalVariant.small}
      isOpen={isOpen}
      onClose={onClose}
      data-testid={`skill-install-modal-${skill.id}`}
    >
      <ModalHeader
        title={`Install ${skillName}`}
        data-testid={`skill-install-modal-title-${skill.id}`}
      />
      <ModalBody>
        <Flex
          alignItems={{ default: 'alignItemsCenter' }}
          spaceItems={{ default: 'spaceItemsSm' }}
          className="pf-v6-u-mb-sm"
        >
          {skill.version && (
            <FlexItem>
              <strong>Latest version:</strong> {skill.version}
            </FlexItem>
          )}
          {skill.trustTier && (
            <FlexItem>
              <Label color="green" data-testid={`skill-install-modal-trust-tier-${skill.id}`}>
                {SKILL_TRUST_TIER_LABEL_MAPPING[skill.trustTier] ?? skill.trustTier}
              </Label>
            </FlexItem>
          )}
        </Flex>
        <SkillInstallTabs skill={skill} idPrefix="skill-install-modal" />
      </ModalBody>
    </Modal>
  );
};

export default SkillInstallModal;

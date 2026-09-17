import * as React from 'react';
import { ExpectedYamlFormatDrawer } from '~/app/shared/catalogSettings';
import sampleCatalogYamlContent from '~/app/pages/modelCatalogSettings/sample-catalog.yaml';
import { EXPECTED_YAML_FORMAT_LABEL } from '~/app/pages/modelCatalogSettings/constants';

type ExpectedYamlFormatDrawerPanelProps = {
  onClose: () => void;
};

export const ExpectedYamlFormatDrawerPanel: React.FC<ExpectedYamlFormatDrawerPanelProps> = ({
  onClose,
}) => (
  <ExpectedYamlFormatDrawer
    onClose={onClose}
    title={EXPECTED_YAML_FORMAT_LABEL}
    sampleYaml={sampleCatalogYamlContent}
  />
);

import * as React from 'react';
import {
  FormFieldGroupExpandable,
  FormFieldGroupHeader,
  TextArea,
  FormHelperText,
  HelperText,
  HelperTextItem,
} from '@patternfly/react-core';
import { ThemeAwareFormGroupWrapper } from 'mod-arch-shared';
import FormSection from '~/app/pages/modelRegistry/components/pf-overrides/FormSection';

export type IncludeExcludeFiltersSectionTestIds = {
  section?: string;
  includedInput?: string;
  excludedInput?: string;
};

type IncludeExcludeFiltersSectionProps = {
  includedValue: string;
  onIncludedChange: (value: string) => void;
  excludedValue: string;
  onExcludedChange: (value: string) => void;
  isDefaultExpanded?: boolean;
  sectionTitle: string;
  sectionTitleId: string;
  sectionDescription: React.ReactNode;
  includedFieldId: string;
  excludedFieldId: string;
  includedLabel: string;
  excludedLabel: string;
  includedDescription: React.ReactNode;
  excludedDescription: React.ReactNode;
  includedHelperText?: React.ReactNode;
  excludedHelperText?: React.ReactNode;
  includedPlaceholder?: string;
  excludedPlaceholder?: string;
  testIds: IncludeExcludeFiltersSectionTestIds;
};

const IncludeExcludeFiltersSection: React.FC<IncludeExcludeFiltersSectionProps> = ({
  includedValue,
  onIncludedChange,
  excludedValue,
  onExcludedChange,
  isDefaultExpanded = false,
  sectionTitle,
  sectionTitleId,
  sectionDescription,
  includedFieldId,
  excludedFieldId,
  includedLabel,
  excludedLabel,
  includedDescription,
  excludedDescription,
  includedHelperText,
  excludedHelperText,
  includedPlaceholder,
  excludedPlaceholder,
  testIds,
}) => {
  const {
    section = 'filters-section',
    includedInput = 'included-input',
    excludedInput = 'excluded-input',
  } = testIds;

  const includedInputNode = (
    <TextArea
      id={includedFieldId}
      name={includedFieldId}
      data-testid={includedInput}
      value={includedValue}
      onChange={(_event, value) => onIncludedChange(value)}
      rows={3}
      resizeOrientation="vertical"
      placeholder={includedPlaceholder}
    />
  );

  const includedDescriptionTxtNode = (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{includedDescription}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  );

  const includedHelperTxtNode = includedHelperText ? (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{includedHelperText}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  ) : undefined;

  const excludedInputNode = (
    <TextArea
      id={excludedFieldId}
      name={excludedFieldId}
      data-testid={excludedInput}
      value={excludedValue}
      onChange={(_event, value) => onExcludedChange(value)}
      rows={3}
      resizeOrientation="vertical"
      placeholder={excludedPlaceholder}
    />
  );

  const excludedDescriptionTxtNode = (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{excludedDescription}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  );

  const excludedHelperTxtNode = excludedHelperText ? (
    <FormHelperText>
      <HelperText>
        <HelperTextItem>{excludedHelperText}</HelperTextItem>
      </HelperText>
    </FormHelperText>
  ) : undefined;

  return (
    <FormSection>
      <FormFieldGroupExpandable
        toggleAriaLabel={sectionTitle}
        header={
          <FormFieldGroupHeader
            titleText={{ text: sectionTitle, id: sectionTitleId }}
            titleDescription={sectionDescription}
          />
        }
        isExpanded={isDefaultExpanded}
        data-testid={section}
      >
        <ThemeAwareFormGroupWrapper
          label={includedLabel}
          fieldId={includedFieldId}
          descriptionTextNode={includedDescriptionTxtNode}
          helperTextNode={includedHelperTxtNode}
        >
          {includedInputNode}
        </ThemeAwareFormGroupWrapper>

        <ThemeAwareFormGroupWrapper
          label={excludedLabel}
          fieldId={excludedFieldId}
          descriptionTextNode={excludedDescriptionTxtNode}
          helperTextNode={excludedHelperTxtNode}
        >
          {excludedInputNode}
        </ThemeAwareFormGroupWrapper>
      </FormFieldGroupExpandable>
    </FormSection>
  );
};

export default IncludeExcludeFiltersSection;

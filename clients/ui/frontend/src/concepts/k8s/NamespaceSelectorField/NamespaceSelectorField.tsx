import React, { useRef } from 'react';
import {
  Alert,
  Button,
  FormGroupLabelHelp,
  HelperText,
  HelperTextItem,
  Popover,
  Stack,
  StackItem,
  TextInput,
} from '@patternfly/react-core';
import { SimpleSelect, ThemeAwareFormGroupWrapper } from 'mod-arch-shared';
import { SimpleSelectOption } from 'mod-arch-shared/dist/components/SimpleSelect';
import { useNamespaces } from '~/app/hooks/useNamespaces';
import { NamespaceSelectorMessages } from '~/app/utilities/const';

const WHO_IS_MY_ADMIN_POPOVER_CONTENT = (
  <Stack hasGutter>
    <StackItem>
      This list includes only namespaces that you have permission to access. To request access to a
      new or existing namespace, contact your administrator.
    </StackItem>
    <StackItem>
      <strong>Your administrator might be:</strong>
    </StackItem>
    <StackItem>
      <ul>
        <li>
          The person who assigned you your username, or who helped you log in for the first time
        </li>
        <li>Someone in your IT department or help desk</li>
        <li>A project manager or developer</li>
        <li>Your professor (at a school)</li>
      </ul>
    </StackItem>
  </Stack>
);

export type NamespaceSelectorFieldProps = {
  selectedNamespace: string;
  onSelect: (namespace: string) => void;
  hasAccess?: boolean | undefined;
  isLoading?: boolean;
  error?: Error | undefined;
  cannotCheck?: boolean;
  registryName?: string;
};

const NamespaceSelectorField: React.FC<NamespaceSelectorFieldProps> = ({
  selectedNamespace,
  onSelect,
  hasAccess,
  isLoading,
  error,
  cannotCheck,
  registryName,
}) => {
  const labelHelpRef = useRef<HTMLSpanElement>(null);
  const [namespaces, namespacesLoaded, namespacesLoadError] = useNamespaces();
  const [textInputValue, setTextInputValue] = React.useState(selectedNamespace);
  const debounceTimerRef = React.useRef<ReturnType<typeof setTimeout>>();

  // TODO: Replace this string matching with proper error code detection once mod-arch-core's
  // handleRestFailures is updated to preserve the HTTP status code from BFF error responses.
  // Currently handleRestFailures discards the error code and only keeps the message string.
  const cannotListNamespaces =
    namespacesLoadError?.message.toLowerCase().includes('forbidden') ?? false;

  const showTextInput =
    cannotListNamespaces || (namespacesLoaded && !namespacesLoadError && namespaces.length === 0);

  const handleTextInputChange = (_event: React.FormEvent, value: string) => {
    setTextInputValue(value);
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    if (value) {
      debounceTimerRef.current = setTimeout(() => {
        onSelect(value);
      }, 1000);
    }
  };

  const handleTextInputBlur = () => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    if (textInputValue && textInputValue !== selectedNamespace) {
      onSelect(textInputValue);
    }
  };

  React.useEffect(
    () => () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    },
    [],
  );

  const options: SimpleSelectOption[] = namespaces.map((ns) => ({
    key: ns.name,
    label: ns.name,
  }));

  const handleChange = (key: string, isPlaceholder: boolean) => {
    if (isPlaceholder || !key) {
      return;
    }
    onSelect(key);
  };

  const namespaceInputElement = showTextInput ? (
    <TextInput
      data-testid="form-namespace-text-input"
      value={textInputValue}
      onChange={handleTextInputChange}
      onBlur={handleTextInputBlur}
      placeholder="Enter a namespace name"
      aria-label="Namespace"
    />
  ) : (
    <div data-testid="form-namespace-selector-trigger">
      <SimpleSelect
        options={options}
        value={selectedNamespace}
        onChange={handleChange}
        placeholder="Select a namespace"
        isDisabled={namespaces.length === 0}
        isFullWidth
        isScrollable
        maxMenuHeight="300px"
        dataTestId="form-namespace-selector"
      />
    </div>
  );

  const showNoAccessAlert = selectedNamespace && !isLoading && hasAccess === false;

  const tooltipContent = showTextInput
    ? NamespaceSelectorMessages.TEXT_INPUT_TOOLTIP
    : NamespaceSelectorMessages.SELECTOR_TOOLTIP;

  const labelHelp = (
    <Popover triggerRef={labelHelpRef} bodyContent={tooltipContent} aria-label={tooltipContent}>
      <FormGroupLabelHelp ref={labelHelpRef} aria-label="More info for namespace field" />
    </Popover>
  );

  const helperTextNode = (
    <>
      {!namespacesLoaded && !namespacesLoadError && (
        <HelperText>
          <HelperTextItem>Loading namespaces...</HelperTextItem>
        </HelperText>
      )}
      {namespacesLoadError && !cannotListNamespaces && (
        <Alert
          isInline
          variant="danger"
          title="Failed to load namespaces"
          data-testid="namespace-load-error"
          className="pf-v6-u-mt-sm"
        >
          {namespacesLoadError.message}
        </Alert>
      )}
      {selectedNamespace && isLoading && (
        <HelperText>
          <HelperTextItem>Checking access...</HelperTextItem>
        </HelperText>
      )}
      {showNoAccessAlert && (
        <Alert
          isInline
          variant="warning"
          title={NamespaceSelectorMessages.SELECTED_NAMESPACE_NO_ACCESS}
          data-testid="namespace-registry-access-alert"
          className="pf-v6-u-mt-sm"
        >
          <Popover bodyContent={WHO_IS_MY_ADMIN_POPOVER_CONTENT} aria-label="Who is my admin?">
            <Button
              variant="link"
              isInline
              component="button"
              data-testid="who-is-my-admin-trigger"
              aria-label="Who is my admin?"
            >
              Who is my admin
            </Button>
          </Popover>
        </Alert>
      )}
      {selectedNamespace && !isLoading && cannotCheck && (
        <Alert
          isInline
          variant="info"
          title="Cannot check registry access with your permissions"
          data-testid="namespace-registry-cannot-check-alert"
          className="pf-v6-u-mt-sm"
        >
          Make sure this namespace has access to the {registryName} registry before proceeding, or
          the model storage job will fail.
        </Alert>
      )}
      {error && (
        <Alert
          isInline
          variant="danger"
          title="Could not verify namespace access"
          data-testid="namespace-registry-access-error"
          className="pf-v6-u-mt-sm"
        >
          {error.message}
        </Alert>
      )}
    </>
  );

  return (
    <ThemeAwareFormGroupWrapper
      label="Namespace"
      fieldId="namespace-select"
      isRequired
      labelHelp={labelHelp}
      helperTextNode={helperTextNode}
      data-testid="namespace-form-group"
    >
      {namespaceInputElement}
    </ThemeAwareFormGroupWrapper>
  );
};

export default NamespaceSelectorField;

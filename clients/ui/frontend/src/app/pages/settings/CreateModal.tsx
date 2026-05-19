import * as React from 'react';
import {
  Button,
  Form,
  FormGroup,
  HelperText,
  HelperTextItem,
  TextInput,
  Alert,
  Modal,
  ModalVariant,
  ModalFooter,
  ModalHeader,
  ModalBody,
} from '@patternfly/react-core';
import { useNavigate } from 'react-router';
import { FormSection, ThemeAwareFormGroupWrapper } from 'mod-arch-shared';
import { createModelRegistrySettings } from '~/app/api/k8s';
import ModelRegistryDatabasePassword from '~/app/pages/settings/ModelRegistryDatabasePassword';
import K8sNameDescriptionField, {
  useK8sNameDescriptionFieldData,
} from '~/concepts/k8s/K8sNameDescriptionField/K8sNameDescriptionField';

type ModelRegistryPayload = {
  modelRegistry: {
    metadata: {
      name: string;
      annotations: {
        'openshift.io/display-name': string;
        'openshift.io/description': string;
      };
    };
    spec: {
      mysql: {
        host: string;
        port: number;
        username: string;
        database: string;
      };
    };
  };
};

type CreateModalProps = {
  onClose: () => void;
  refresh: () => void;
};

const CreateModal: React.FC<CreateModalProps> = ({ onClose, refresh }) => {
  const [error, setError] = React.useState<Error>();
  const { data: nameDescData, onDataChange: onNameDescChange } = useK8sNameDescriptionFieldData();
  const [host, setHost] = React.useState('');
  const [port, setPort] = React.useState('');
  const [username, setUsername] = React.useState('');
  const [password, setPassword] = React.useState('');
  const [database, setDatabase] = React.useState('');
  const [isHostTouched, setIsHostTouched] = React.useState(false);
  const [isPortTouched, setIsPortTouched] = React.useState(false);
  const [isUsernameTouched, setIsUsernameTouched] = React.useState(false);
  const [isPasswordTouched, setIsPasswordTouched] = React.useState(false);
  const [isDatabaseTouched, setIsDatabaseTouched] = React.useState(false);
  const [showPassword, setShowPassword] = React.useState(false);

  const navigate = useNavigate();

  const onBeforeClose = () => {
    setError(undefined);
    onNameDescChange('name', '');
    onNameDescChange('description', '');
    setHost('');
    setPort('');
    setUsername('');
    setPassword('');
    setDatabase('');
    setIsHostTouched(false);
    setIsPortTouched(false);
    setIsUsernameTouched(false);
    setIsPasswordTouched(false);
    setIsDatabaseTouched(false);
    setShowPassword(false);
    onClose();
  };

  const hasContent = (value: string): boolean => !!value.trim().length;

  const canSubmit = () =>
    hasContent(nameDescData.name) &&
    hasContent(nameDescData.k8sName.value) &&
    !nameDescData.k8sName.state.invalidCharacters &&
    !nameDescData.k8sName.state.invalidLength &&
    hasContent(host) &&
    hasContent(password) &&
    hasContent(port) &&
    hasContent(username) &&
    hasContent(database);

  const onSubmit = async () => {
    setError(undefined);

    // This is a simplified payload for the BFF, not a full K8s object.
    const payload: ModelRegistryPayload = {
      modelRegistry: {
        metadata: {
          name: nameDescData.k8sName.value,
          annotations: {
            'openshift.io/display-name': nameDescData.name,
            'openshift.io/description': nameDescData.description,
          },
        },
        spec: {
          mysql: {
            host,
            port: Number(port),
            username,
            database,
          },
        },
      },
    };

    try {
      await createModelRegistrySettings(window.location.origin, {
        namespace: 'model-registry',
      })({}, payload);
      refresh();
      navigate(`/model-registry-settings`);
      onClose();
    } catch (e) {
      if (e instanceof Error) {
        setError(e);
      }
    }
  };

  const hostInput = (
    <TextInput
      isRequired
      type="text"
      id="mr-host"
      name="mr-host"
      value={host}
      onBlur={() => setIsHostTouched(true)}
      onChange={(_e, value) => setHost(value)}
    />
  );

  const hostHelperText = isHostTouched && !hasContent(host) && (
    <HelperText>
      <HelperTextItem variant="error" data-testid="mr-host-error">
        Host cannot be empty
      </HelperTextItem>
    </HelperText>
  );

  const portInput = (
    <TextInput
      isRequired
      type="text"
      id="mr-port"
      name="mr-port"
      value={port}
      onBlur={() => setIsPortTouched(true)}
      onChange={(_e, value) => setPort(value)}
    />
  );

  const portHelperText = isPortTouched && !hasContent(port) && (
    <HelperText>
      <HelperTextItem variant="error" data-testid="mr-port-error">
        Port cannot be empty
      </HelperTextItem>
    </HelperText>
  );

  const userNameInput = (
    <TextInput
      isRequired
      type="text"
      id="mr-username"
      name="mr-username"
      value={username}
      onBlur={() => setIsUsernameTouched(true)}
      onChange={(_e, value) => setUsername(value)}
    />
  );

  const usernameHelperText = isUsernameTouched && !hasContent(username) && (
    <HelperText>
      <HelperTextItem variant="error" data-testid="mr-username-error">
        Username cannot be empty
      </HelperTextItem>
    </HelperText>
  );

  const passwordInput = (
    <ModelRegistryDatabasePassword
      password={password || ''}
      setPassword={setPassword}
      isPasswordTouched={isPasswordTouched}
      setIsPasswordTouched={setIsPasswordTouched}
      showPassword={showPassword}
    />
  );

  const passwordHelperText = isPasswordTouched && !hasContent(password) && (
    <HelperText>
      <HelperTextItem variant="error" data-testid="mr-password-error">
        Password cannot be empty
      </HelperTextItem>
    </HelperText>
  );

  const databaseInput = (
    <TextInput
      isRequired
      type="text"
      id="mr-database"
      name="mr-database"
      value={database}
      onBlur={() => setIsDatabaseTouched(true)}
      onChange={(_e, value) => setDatabase(value)}
    />
  );

  const databaseHelperText = isDatabaseTouched && !hasContent(database) && (
    <HelperText>
      <HelperTextItem variant="error" data-testid="mr-database-error">
        Database cannot be empty
      </HelperTextItem>
    </HelperText>
  );

  return (
    <Modal
      isOpen
      variant={ModalVariant.medium}
      onClose={onBeforeClose}
      data-testid="create-model-registry-modal"
    >
      <ModalHeader title="Create model registry" />
      <ModalBody>
        <Form>
          <K8sNameDescriptionField
            dataTestId="mr"
            data={nameDescData}
            onDataChange={onNameDescChange}
          />
          <FormSection
            title="Connect to external MySQL database"
            description="This external database is where model data is stored."
          >
            <ThemeAwareFormGroupWrapper
              label="Host"
              fieldId="mr-host"
              isRequired
              hasError={!!hostHelperText}
              helperTextNode={hostHelperText}
            >
              {hostInput}
            </ThemeAwareFormGroupWrapper>

            <ThemeAwareFormGroupWrapper
              label="Port"
              fieldId="mr-port"
              isRequired
              hasError={!!portHelperText}
              helperTextNode={portHelperText}
            >
              {portInput}
            </ThemeAwareFormGroupWrapper>

            <ThemeAwareFormGroupWrapper
              label="Username"
              fieldId="mr-username"
              isRequired
              hasError={!!usernameHelperText}
              helperTextNode={usernameHelperText}
            >
              {userNameInput}
            </ThemeAwareFormGroupWrapper>

            <ThemeAwareFormGroupWrapper
              label="Password"
              fieldId="mr-password"
              isRequired
              hasError={!!passwordHelperText}
              helperTextNode={passwordHelperText}
            >
              {passwordInput}
            </ThemeAwareFormGroupWrapper>

            <ThemeAwareFormGroupWrapper
              label="Database"
              fieldId="mr-database"
              isRequired
              hasError={!!databaseHelperText}
              helperTextNode={databaseHelperText}
            >
              {databaseInput}
            </ThemeAwareFormGroupWrapper>

            {/* ... Optional TLS section ... */}
          </FormSection>

          {error && (
            <FormGroup>
              <Alert variant="danger" isInline title={error.message} data-testid="mr-error" />
            </FormGroup>
          )}
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button key="create-button" variant="primary" isDisabled={!canSubmit()} onClick={onSubmit}>
          Create
        </Button>
        <Button key="cancel-button" variant="secondary" onClick={onBeforeClose}>
          Cancel
        </Button>
      </ModalFooter>
    </Modal>
  );
};

export default CreateModal;

import * as React from 'react';
import {
  Alert,
  Flex,
  FlexItem,
  Label,
  Modal,
  ModalBody,
  ModalHeader,
  ModalVariant,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { ERROR_MESSAGES } from '~/app/pages/modelCatalogSettings/constants';

type CatalogSourceStatusErrorModalProps = {
  isOpen: boolean;
  onClose: () => void;
  errorMessage: string;
};

const CatalogSourceStatusErrorModal: React.FC<CatalogSourceStatusErrorModalProps> = ({
  isOpen,
  onClose,
  errorMessage,
}) => {
  const titleWithLabel = (
    <Flex spaceItems={{ default: 'spaceItemsSm' }} alignItems={{ default: 'alignItemsCenter' }}>
      <FlexItem>Source status</FlexItem>
      <FlexItem>
        <Label status="danger" variant="outline" icon={<ExclamationCircleIcon />}>
          Failed
        </Label>
      </FlexItem>
    </Flex>
  );

  return (
    <Modal
      variant={ModalVariant.medium}
      isOpen={isOpen}
      onClose={onClose}
      data-testid="catalog-source-status-error-modal"
    >
      <ModalHeader title={titleWithLabel} />
      <ModalBody>
        <Alert
          variant="danger"
          isInline
          title={ERROR_MESSAGES.SOURCE_VALIDATION_FAILED}
          data-testid="catalog-source-status-error-alert"
        >
          <p data-testid="catalog-source-status-error-details">
            {ERROR_MESSAGES.SOURCE_VALIDATION_FAILED_BODY}
          </p>
          {errorMessage && <p data-testid="catalog-source-status-error-message">{errorMessage}</p>}
        </Alert>
      </ModalBody>
    </Modal>
  );
};

export default CatalogSourceStatusErrorModal;

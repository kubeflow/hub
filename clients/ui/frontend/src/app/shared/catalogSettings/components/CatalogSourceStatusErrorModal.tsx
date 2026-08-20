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

type CatalogSourceStatusErrorModalProps = {
  isOpen: boolean;
  onClose: () => void;
  errorMessage: string;
  validationFailedTitle?: string;
  validationFailedBody?: string;
  testId?: string;
};

const CatalogSourceStatusErrorModal: React.FC<CatalogSourceStatusErrorModalProps> = ({
  isOpen,
  onClose,
  errorMessage,
  validationFailedTitle = 'Validation failed',
  validationFailedBody = 'The system cannot establish a connection to the source.',
  testId = 'catalog-source-status-error-modal',
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
    <Modal variant={ModalVariant.medium} isOpen={isOpen} onClose={onClose} data-testid={testId}>
      <ModalHeader title={titleWithLabel} />
      <ModalBody>
        <Alert
          variant="danger"
          isInline
          title={validationFailedTitle}
          data-testid="catalog-source-status-error-alert"
        >
          <p data-testid="catalog-source-status-error-details">{validationFailedBody}</p>
          {errorMessage && <p data-testid="catalog-source-status-error-message">{errorMessage}</p>}
        </Alert>
      </ModalBody>
    </Modal>
  );
};

export default CatalogSourceStatusErrorModal;

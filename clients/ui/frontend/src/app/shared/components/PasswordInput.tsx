import * as React from 'react';
import { Button, InputGroup, TextInput, InputGroupItem } from '@patternfly/react-core';
import { EyeIcon, EyeSlashIcon } from '@patternfly/react-icons';

type Props = React.ComponentProps<typeof TextInput> & {
  ariaLabelShow?: string;
  ariaLabelHide?: string;
  hideToggleButton?: boolean;
  forceHidden?: boolean;
};

const PasswordInput: React.FC<Props> = ({
  ariaLabelShow = 'Show password',
  ariaLabelHide = 'Hide password',
  hideToggleButton = false,
  forceHidden = false,
  ...props
}) => {
  const [isPasswordHidden, setPasswordHidden] = React.useState(true);

  React.useEffect(() => {
    if (forceHidden) {
      setPasswordHidden(true);
    }
  }, [forceHidden]);

  return (
    <InputGroup>
      <InputGroupItem isFill>
        <TextInput {...props} type={isPasswordHidden ? 'password' : 'text'} />
      </InputGroupItem>
      {!hideToggleButton && (
        <InputGroupItem>
          <Button
            aria-label={isPasswordHidden ? ariaLabelShow : ariaLabelHide}
            variant="control"
            onClick={() => setPasswordHidden(!isPasswordHidden)}
          >
            {isPasswordHidden ? <EyeSlashIcon /> : <EyeIcon />}
          </Button>
        </InputGroupItem>
      )}
    </InputGroup>
  );
};

export default PasswordInput;

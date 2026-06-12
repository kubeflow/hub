import * as React from 'react';
import { Slider, SliderOnChangeEvent } from '@patternfly/react-core';

type SliderWithInputProps = {
  value: number;
  min: number;
  max: number;
  isDisabled: boolean;
  onChange: (value: number) => void;
  suffix?: string;
  ariaLabel: string;
  shouldRound?: boolean;
  showBoundaries?: boolean;
  hasTooltipOverThumb?: boolean;
};

const SliderWithInput: React.FC<SliderWithInputProps> = ({
  value,
  min,
  max,
  isDisabled,
  onChange,
  suffix = '',
  ariaLabel,
  shouldRound = false,
  showBoundaries = false,
  hasTooltipOverThumb = false,
}) => {
  const roundValue = React.useCallback(
    (val: number) => (shouldRound ? Math.round(val) : val),
    [shouldRound],
  );

  // Maintain separate state for value and inputValue, following PatternFly's example pattern
  const [localValue, setLocalValue] = React.useState<number>(() => roundValue(value));
  const [localInputValue, setLocalInputValue] = React.useState<number>(() => roundValue(value));

  // Sync local state when prop value changes (from parent)
  React.useEffect(() => {
    const roundedValue = roundValue(value);
    setLocalValue(roundedValue);
    setLocalInputValue(roundedValue);
  }, [value, roundValue]);

  const handleChange = (
    _event: SliderOnChangeEvent,
    sliderValue: number,
    inputValueArg?: number,
    setPFInputValue?: React.Dispatch<React.SetStateAction<number>>,
  ) => {
    let newValue: number;

    if (inputValueArg === undefined) {
      newValue = roundValue(sliderValue);
      setLocalValue(newValue);
      setLocalInputValue(newValue);
    } else {
      if (inputValueArg > max) {
        newValue = max;
        setPFInputValue?.(max);
      } else if (inputValueArg < min) {
        newValue = min;
        setPFInputValue?.(min);
      } else {
        newValue = roundValue(inputValueArg);
      }
      setLocalValue(newValue);
      setLocalInputValue(newValue);
    }

    onChange(newValue);
  };

  return (
    <Slider
      min={min}
      max={max}
      value={localValue}
      inputValue={localInputValue}
      onChange={handleChange}
      isInputVisible
      inputLabel={suffix}
      inputAriaLabel={ariaLabel}
      isDisabled={isDisabled}
      showBoundaries={showBoundaries}
      hasTooltipOverThumb={hasTooltipOverThumb}
    />
  );
};

export default SliderWithInput;

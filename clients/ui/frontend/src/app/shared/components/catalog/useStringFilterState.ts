import * as React from 'react';

/**
 * Shared hook for managing a string-array filter selection.
 * Context-agnostic: each catalog provides its current values and a change callback.
 *
 * @param currentValues - The currently selected filter values (from context)
 * @param onChange - Callback to set the next selection array (writes back to context)
 * @returns Standardized filter state: { selectedValues, isSelected, setSelected }
 */
export function useStringFilterState(
  currentValues: string[],
  onChange: (nextValues: string[]) => void,
): {
  selectedValues: string[];
  isSelected: (value: string) => boolean;
  setSelected: (value: string, checked: boolean) => void;
} {
  const isSelected = React.useCallback(
    (value: string) => currentValues.includes(value),
    [currentValues],
  );

  const setSelected = React.useCallback(
    (value: string, checked: boolean) => {
      if (checked) {
        if (!currentValues.includes(value)) {
          onChange([...currentValues, value]);
        }
      } else {
        onChange(currentValues.filter((x) => x !== value));
      }
    },
    [currentValues, onChange],
  );

  return { selectedValues: currentValues, isSelected, setSelected };
}

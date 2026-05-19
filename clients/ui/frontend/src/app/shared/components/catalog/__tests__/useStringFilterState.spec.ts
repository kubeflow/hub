import { renderHook, act } from '@testing-library/react';
import { useStringFilterState } from '~/app/shared/components/catalog';

describe('useStringFilterState', () => {
  it('returns the current values as selectedValues', () => {
    const { result } = renderHook(() => useStringFilterState(['a', 'b'], jest.fn()));
    expect(result.current.selectedValues).toEqual(['a', 'b']);
  });

  it('isSelected returns true for values in the current list', () => {
    const { result } = renderHook(() => useStringFilterState(['a', 'b'], jest.fn()));
    expect(result.current.isSelected('a')).toBe(true);
    expect(result.current.isSelected('b')).toBe(true);
  });

  it('isSelected returns false for values not in the current list', () => {
    const { result } = renderHook(() => useStringFilterState(['a'], jest.fn()));
    expect(result.current.isSelected('z')).toBe(false);
  });

  it('setSelected with checked=true calls onChange with value appended', () => {
    const onChange = jest.fn();
    const { result } = renderHook(() => useStringFilterState(['a'], onChange));

    act(() => {
      result.current.setSelected('b', true);
    });

    expect(onChange).toHaveBeenCalledWith(['a', 'b']);
  });

  it('setSelected with checked=false calls onChange with value removed', () => {
    const onChange = jest.fn();
    const { result } = renderHook(() => useStringFilterState(['a', 'b', 'c'], onChange));

    act(() => {
      result.current.setSelected('b', false);
    });

    expect(onChange).toHaveBeenCalledWith(['a', 'c']);
  });

  it('setSelected with checked=false for non-existent value returns same array', () => {
    const onChange = jest.fn();
    const { result } = renderHook(() => useStringFilterState(['a'], onChange));

    act(() => {
      result.current.setSelected('z', false);
    });

    expect(onChange).toHaveBeenCalledWith(['a']);
  });

  it('setSelected with checked=true does not add duplicates', () => {
    const onChange = jest.fn();
    const { result } = renderHook(() => useStringFilterState(['a', 'b'], onChange));

    act(() => {
      result.current.setSelected('a', true);
    });

    expect(onChange).not.toHaveBeenCalled();
  });

  it('works with empty initial values', () => {
    const onChange = jest.fn();
    const { result } = renderHook(() => useStringFilterState([], onChange));

    expect(result.current.selectedValues).toEqual([]);
    expect(result.current.isSelected('a')).toBe(false);

    act(() => {
      result.current.setSelected('a', true);
    });

    expect(onChange).toHaveBeenCalledWith(['a']);
  });
});

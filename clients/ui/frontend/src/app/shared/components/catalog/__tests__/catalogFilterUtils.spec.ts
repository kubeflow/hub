/* eslint-disable camelcase */
import {
  wrapInQuotes,
  eqFilter,
  inFilter,
  andFilter,
  stringFiltersToFilterQuery,
} from '~/app/shared/components/catalog';

describe('wrapInQuotes', () => {
  it('wraps a simple string in single quotes', () => {
    expect(wrapInQuotes('hello')).toBe("'hello'");
  });

  it('escapes single quotes inside the value', () => {
    expect(wrapInQuotes("it's")).toBe("'it''s'");
  });

  it('handles empty string', () => {
    expect(wrapInQuotes('')).toBe("''");
  });
});

describe('eqFilter', () => {
  it('produces an equality clause', () => {
    expect(eqFilter('provider', 'Red Hat')).toBe("provider='Red Hat'");
  });

  it('escapes quotes in the value', () => {
    expect(eqFilter('name', "O'Reilly")).toBe("name='O''Reilly'");
  });
});

describe('inFilter', () => {
  it('produces an IN clause with multiple values', () => {
    expect(inFilter('license', ['MIT', 'Apache'])).toBe("license IN ('MIT','Apache')");
  });

  it('produces an IN clause with a single value', () => {
    expect(inFilter('license', ['MIT'])).toBe("license IN ('MIT')");
  });
});

describe('andFilter', () => {
  it('produces AND-joined equality clauses', () => {
    expect(andFilter('tasks', ['chat', 'code'])).toBe("tasks='chat' AND tasks='code'");
  });

  it('produces a single equality for one value', () => {
    expect(andFilter('tasks', ['chat'])).toBe("tasks='chat'");
  });
});

describe('stringFiltersToFilterQuery', () => {
  it('returns empty string when no filters are active', () => {
    expect(stringFiltersToFilterQuery({})).toBe('');
  });

  it('returns empty string when all filter values are empty arrays', () => {
    expect(stringFiltersToFilterQuery({ provider: [], license: [] })).toBe('');
  });

  it('skips undefined values', () => {
    expect(stringFiltersToFilterQuery({ provider: undefined, license: ['MIT'] })).toBe(
      "license='MIT'",
    );
  });

  it('uses equality for single-value filters', () => {
    expect(stringFiltersToFilterQuery({ provider: ['Red Hat'] })).toBe("provider='Red Hat'");
  });

  it('uses IN for multi-value filters (OR logic)', () => {
    expect(stringFiltersToFilterQuery({ provider: ['Red Hat', 'IBM'] })).toBe(
      "provider IN ('Red Hat','IBM')",
    );
  });

  it('joins multiple filter keys with AND', () => {
    const result = stringFiltersToFilterQuery({
      provider: ['Red Hat'],
      license: ['MIT', 'Apache'],
    });
    expect(result).toBe("provider='Red Hat' AND license IN ('MIT','Apache')");
  });

  it('applies keyMapping to remap frontend keys to backend keys', () => {
    const result = stringFiltersToFilterQuery(
      { supportedTransports: ['stdio'] },
      { supportedTransports: 'transports' },
    );
    expect(result).toBe("transports='stdio'");
  });

  it('uses AND logic for keys in matchAllKeys', () => {
    const result = stringFiltersToFilterQuery({ validated_tasks: ['chat', 'code'] }, undefined, [
      'validated_tasks',
    ]);
    expect(result).toBe("validated_tasks='chat' AND validated_tasks='code'");
  });

  it('uses IN for multi-value keys not in matchAllKeys', () => {
    const result = stringFiltersToFilterQuery(
      { validated_tasks: ['chat', 'code'], provider: ['Red Hat', 'IBM'] },
      undefined,
      ['validated_tasks'],
    );
    expect(result).toContain("validated_tasks='chat' AND validated_tasks='code'");
    expect(result).toContain("provider IN ('Red Hat','IBM')");
  });

  it('combines keyMapping and matchAllKeys', () => {
    const result = stringFiltersToFilterQuery({ vc: ['a', 'b'] }, { vc: 'validated_config' }, [
      'vc',
    ]);
    expect(result).toBe("validated_config='a' AND validated_config='b'");
  });
});

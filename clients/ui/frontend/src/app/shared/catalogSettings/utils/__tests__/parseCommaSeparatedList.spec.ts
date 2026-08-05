import { parseCommaSeparatedList } from '~/app/shared/catalogSettings/utils/parseCommaSeparatedList';

describe('parseCommaSeparatedList', () => {
  it('should split, trim, and drop empty entries', () => {
    expect(parseCommaSeparatedList('server1, server2,  server3  ')).toEqual([
      'server1',
      'server2',
      'server3',
    ]);
  });

  it('should return an empty array for empty input', () => {
    expect(parseCommaSeparatedList('')).toEqual([]);
    expect(parseCommaSeparatedList(' , , ')).toEqual([]);
  });
});

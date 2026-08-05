export const isNonEmptyString = (value: string): boolean => value.trim().length > 0;

export const validateSourceName = (name: string, maxLength: number): boolean =>
  isNonEmptyString(name) && name.length <= maxLength;

export const isSourceNameEmpty = (name: string): boolean => !isNonEmptyString(name);

export const validateYamlContent = (yamlContent: string): boolean => isNonEmptyString(yamlContent);

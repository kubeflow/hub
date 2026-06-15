import * as React from 'react';
import { Icon, Label, LabelGroup } from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';
import { MODEL_CATALOG_TASK_NAME_MAPPING, ModelCatalogTask } from '~/concepts/modelCatalog/const';

const isModelCatalogTask = (task: string): task is ModelCatalogTask =>
  Object.values<string>(ModelCatalogTask).includes(task);

const getTaskDisplayName = (task: string): string =>
  isModelCatalogTask(task) ? MODEL_CATALOG_TASK_NAME_MAPPING[task] : task;

type ModelCatalogLabelsProps = {
  tasks?: string[];
  validatedTasks?: string[];
  provider?: string;
  labels?: string[];
  numLabels: number;
};

const ModelCatalogLabels: React.FC<ModelCatalogLabelsProps> = ({
  tasks = [],
  validatedTasks,
  provider,
  labels = [],
  numLabels,
}) => {
  const validatedSet = React.useMemo(() => new Set(validatedTasks ?? []), [validatedTasks]);

  const sortedTasks = React.useMemo(() => {
    if (!validatedSet.size) {
      return tasks;
    }
    return tasks.toSorted((a, b) => {
      const aValidated = validatedSet.has(a) ? 0 : 1;
      const bValidated = validatedSet.has(b) ? 0 : 1;
      return aValidated - bValidated;
    });
  }, [tasks, validatedSet]);

  return (
    <LabelGroup numLabels={numLabels} isCompact>
      {sortedTasks.map((task) => {
        const isValidatedTask = validatedSet.has(task);
        return (
          <Label
            data-testid="model-catalog-label"
            key={task}
            variant="outline"
            icon={
              isValidatedTask ? (
                <Icon status="success">
                  <CheckCircleIcon />
                </Icon>
              ) : undefined
            }
          >
            {getTaskDisplayName(task)}
          </Label>
        );
      })}
      {provider && (
        <Label isCompact variant="outline">
          {provider}
        </Label>
      )}
      {labels.map((label) => (
        <Label data-testid="model-catalog-label" key={label} variant="outline">
          {label}
        </Label>
      ))}
    </LabelGroup>
  );
};

export default ModelCatalogLabels;

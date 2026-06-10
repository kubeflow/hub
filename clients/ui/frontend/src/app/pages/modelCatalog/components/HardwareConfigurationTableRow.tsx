import * as React from 'react';
import { Button, Popover } from '@patternfly/react-core';
import { Td, Tr } from '@patternfly/react-table';
import { CatalogPerformanceMetricsArtifact, HardwareConfiguration } from '~/app/modelCatalogTypes';
import CodeBlockComponent from '~/app/shared/markdown/components/CodeBlockComponent';
import {
  formatLatency,
  formatTps,
  formatTokenValue,
  getWorkloadType,
} from '~/app/pages/modelCatalog/utils/performanceMetricsUtils';
import { getDoubleValue, getIntValue, getStringValue } from '~/app/utils';
import { PerformancePropertyKey, EMPTY_CUSTOM_PROPERTY_VALUE } from '~/concepts/modelCatalog/const';
import {
  HardwareConfigColumnField,
  HardwareConfigColumn,
} from './HardwareConfigurationTableColumns';

type HardwareConfigurationTableRowProps = {
  performanceArtifact: CatalogPerformanceMetricsArtifact;
  columns: HardwareConfigColumn[];
  matchedHardwareConfig?: HardwareConfiguration;
};

const HardwareConfigurationTableRow: React.FC<HardwareConfigurationTableRowProps> = ({
  performanceArtifact,
  columns,
  matchedHardwareConfig,
}) => {
  const getCellValue = (field: HardwareConfigColumnField): string | number => {
    const { customProperties } = performanceArtifact;

    switch (field) {
      case PerformancePropertyKey.HARDWARE_CONFIGURATION:
        return getStringValue(customProperties, field);
      case PerformancePropertyKey.HARDWARE_TYPE:
        return getStringValue(customProperties, field);
      case PerformancePropertyKey.USE_CASE:
        return getWorkloadType(performanceArtifact);
      case 'hardware_count':
        return getIntValue(customProperties, 'hardware_count');
      case PerformancePropertyKey.REQUESTS_PER_SECOND:
        return getDoubleValue(customProperties, PerformancePropertyKey.REQUESTS_PER_SECOND);
      case 'replicas': {
        const replicasValue = getIntValue(customProperties, 'replicas');
        return replicasValue > 0 ? replicasValue : EMPTY_CUSTOM_PROPERTY_VALUE;
      }
      case 'total_requests_per_second': {
        const targetRpsValue = getDoubleValue(customProperties, 'total_requests_per_second');
        return targetRpsValue > 0 ? targetRpsValue : EMPTY_CUSTOM_PROPERTY_VALUE;
      }
      case 'ttft_mean':
      case 'ttft_p90':
      case 'ttft_p95':
      case 'ttft_p99':
      case 'e2e_mean':
      case 'e2e_p90':
      case 'e2e_p95':
      case 'e2e_p99':
      case 'itl_mean':
      case 'itl_p90':
      case 'itl_p95':
      case 'itl_p99':
        return formatLatency(getDoubleValue(customProperties, field));
      case 'tps_mean':
      case 'tps_p90':
      case 'tps_p95':
      case 'tps_p99':
        return formatTps(getDoubleValue(customProperties, field));
      case 'mean_input_tokens':
      case 'mean_output_tokens':
        return formatTokenValue(getDoubleValue(customProperties, field));
      case 'framework_version':
        return getStringValue(customProperties, field);
      default:
        return EMPTY_CUSTOM_PROPERTY_VALUE;
    }
  };

  const renderModelLevelCell = (field: HardwareConfigColumnField): React.ReactNode => {
    if (field === 'cold_start_load_time') {
      return matchedHardwareConfig
        ? `${matchedHardwareConfig.cold_start_time_to_load_seconds.toFixed(2)} s`
        : EMPTY_CUSTOM_PROPERTY_VALUE;
    }
    if (field === 'runtime_command') {
      return matchedHardwareConfig?.runtime_command ? (
        <Popover
          bodyContent={
            <CodeBlockComponent>{matchedHardwareConfig.runtime_command}</CodeBlockComponent>
          }
          position="left"
          maxWidth="450px"
        >
          <Button variant="link" isInline data-testid="view-runtime-command">
            View
          </Button>
        </Popover>
      ) : (
        EMPTY_CUSTOM_PROPERTY_VALUE
      );
    }
    return getCellValue(field);
  };

  return (
    <Tr>
      {columns.map((column) => (
        <Td
          key={column.field}
          dataLabel={column.label.replace('\n', ' ')}
          isStickyColumn={column.isStickyColumn}
          stickyMinWidth={column.stickyMinWidth}
          stickyLeftOffset={column.stickyLeftOffset}
          hasRightBorder={column.hasRightBorder}
          modifier="fitContent"
        >
          {renderModelLevelCell(column.field)}
        </Td>
      ))}
    </Tr>
  );
};

export default HardwareConfigurationTableRow;

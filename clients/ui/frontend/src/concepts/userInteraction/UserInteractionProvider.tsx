import * as React from 'react';
import { UserInteractionContext, type UserInteractionAPI } from './UserInteractionContext';

type UserInteractionProviderProps = {
  /** Custom tracking API implementation. When omitted, the default dev-mode logger is used. */
  api?: UserInteractionAPI;
  children: React.ReactNode;
};

/**
 * Provider for user interaction tracking.
 *
 * Wrap your app (or a subtree) with this provider to inject a custom tracking implementation.
 * When no `api` prop is provided, the default context value (dev-mode console logger) is used.
 *
 * Downstream integrations (e.g., ODH with Segment) can supply their own `api` implementation
 * to route events to their analytics backend.
 */
const UserInteractionProvider: React.FC<UserInteractionProviderProps> = ({ api, children }) => {
  const defaultApi = React.useContext(UserInteractionContext);
  const value = api ?? defaultApi;

  return (
    <UserInteractionContext.Provider value={value}>{children}</UserInteractionContext.Provider>
  );
};

export default UserInteractionProvider;

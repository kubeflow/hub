import React from 'react';
import { renderHook, act } from '@testing-library/react';
import { useUserInteraction } from '~/concepts/userInteraction/useUserInteraction';
import UserInteractionProvider from '~/concepts/userInteraction/UserInteractionProvider';
import { TrackingOutcome } from '~/concepts/userInteraction/trackingTypes';
import type { UserInteractionAPI } from '~/concepts/userInteraction/UserInteractionContext';

describe('useUserInteraction', () => {
  it('calls the injected provider implementation instead of the default', () => {
    const mockApi: UserInteractionAPI = {
      trackFormEvent: jest.fn(),
      trackLinkEvent: jest.fn(),
      trackSimpleEvent: jest.fn(),
      trackPageEvent: jest.fn(),
    };

    const wrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => (
      <UserInteractionProvider api={mockApi}>{children}</UserInteractionProvider>
    );

    const { result } = renderHook(() => useUserInteraction(), { wrapper });

    act(() => {
      result.current.trackFormEvent('Model Registered', {
        outcome: TrackingOutcome.submit,
        success: true,
        registryName: 'production',
      });
    });

    expect(mockApi.trackFormEvent).toHaveBeenCalledWith('Model Registered', {
      outcome: 'submit',
      success: true,
      registryName: 'production',
    });
    expect(mockApi.trackLinkEvent).not.toHaveBeenCalled();
  });

  it('routes different event types to their respective handlers', () => {
    const mockApi: UserInteractionAPI = {
      trackFormEvent: jest.fn(),
      trackLinkEvent: jest.fn(),
      trackSimpleEvent: jest.fn(),
      trackPageEvent: jest.fn(),
    };

    const wrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => (
      <UserInteractionProvider api={mockApi}>{children}</UserInteractionProvider>
    );

    const { result } = renderHook(() => useUserInteraction(), { wrapper });

    act(() => {
      result.current.trackLinkEvent('Model Card Clicked', {
        href: '/catalog/source1/granite-3b',
        section: 'Model Catalog',
      });
      result.current.trackSimpleEvent('Performance View Toggled');
      result.current.trackPageEvent();
    });

    expect(mockApi.trackLinkEvent).toHaveBeenCalledWith('Model Card Clicked', {
      href: '/catalog/source1/granite-3b',
      section: 'Model Catalog',
    });
    expect(mockApi.trackSimpleEvent).toHaveBeenCalledWith('Performance View Toggled');
    expect(mockApi.trackPageEvent).toHaveBeenCalledTimes(1);
  });

  it('swaps implementation when provider is re-mounted with a different api', () => {
    const firstApi: UserInteractionAPI = {
      trackFormEvent: jest.fn(),
      trackLinkEvent: jest.fn(),
      trackSimpleEvent: jest.fn(),
      trackPageEvent: jest.fn(),
    };
    const secondApi: UserInteractionAPI = {
      trackFormEvent: jest.fn(),
      trackLinkEvent: jest.fn(),
      trackSimpleEvent: jest.fn(),
      trackPageEvent: jest.fn(),
    };

    let currentApi = firstApi;

    const wrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => (
      <UserInteractionProvider api={currentApi}>{children}</UserInteractionProvider>
    );

    const { result, rerender } = renderHook(() => useUserInteraction(), { wrapper });

    act(() => {
      result.current.trackSimpleEvent('Event One');
    });
    expect(firstApi.trackSimpleEvent).toHaveBeenCalledWith('Event One');
    expect(secondApi.trackSimpleEvent).not.toHaveBeenCalled();

    currentApi = secondApi;
    rerender();

    act(() => {
      result.current.trackSimpleEvent('Event Two');
    });
    expect(secondApi.trackSimpleEvent).toHaveBeenCalledWith('Event Two');
    expect(firstApi.trackSimpleEvent).toHaveBeenCalledTimes(1);
  });
});

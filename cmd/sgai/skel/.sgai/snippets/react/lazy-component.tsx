---
name: Lazy Component
description: React.lazy + Suspense code splitting pattern
when_to_use: When loading heavy components on demand to reduce initial bundle size
---

/* Code splitting with React.lazy and Suspense for on-demand component loading */

import { lazy, Suspense, type ReactNode } from 'react';

const HeavyChart = lazy(() => import('./HeavyChart'));
const AdminPanel = lazy(() => import('./AdminPanel'));

interface LazyWrapperProps {
  fallback?: ReactNode;
  children: ReactNode;
}

function LazyWrapper({ fallback, children }: LazyWrapperProps) {
  return (
    <Suspense fallback={fallback ?? <div aria-busy="true">Loading...</div>}>
      {children}
    </Suspense>
  );
}

function Dashboard({ isAdmin }: { isAdmin: boolean }) {
  return (
    <div>
      <h1>Dashboard</h1>

      <LazyWrapper fallback={<div>Loading chart...</div>}>
        <HeavyChart />
      </LazyWrapper>

      {isAdmin && (
        <LazyWrapper fallback={<div>Loading admin panel...</div>}>
          <AdminPanel />
        </LazyWrapper>
      )}
    </div>
  );
}

export { Dashboard, LazyWrapper };

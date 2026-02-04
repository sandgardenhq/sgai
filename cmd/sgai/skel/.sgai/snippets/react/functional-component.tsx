---
name: Functional Component
description: Typed React functional component with props interface
when_to_use: When creating a new React component with TypeScript type safety
---

/* A typed React functional component with proper props interface and default values */

import { type ReactNode } from 'react';

interface ButtonProps {
  variant?: 'primary' | 'secondary' | 'danger';
  isLoading?: boolean;
  disabled?: boolean;
  children: ReactNode;
  onClick?: () => void;
}

export function Button({
  variant = 'primary',
  isLoading = false,
  disabled = false,
  children,
  onClick,
}: ButtonProps) {
  return (
    <button
      className={`btn btn-${variant}`}
      disabled={isLoading || disabled}
      onClick={onClick}
      aria-busy={isLoading}
    >
      {isLoading ? <span aria-label="Loading">Loading...</span> : children}
    </button>
  );
}

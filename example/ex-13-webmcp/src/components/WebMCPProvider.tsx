import type { ReactNode } from 'react';

import '@mcp-b/global';

export function WebMCPProvider({ children }: { children: ReactNode }) {
  return <>{children}</>;
}

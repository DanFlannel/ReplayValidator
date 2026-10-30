import { useEffect, useState } from 'react';

import { TamaguiProvider, Text, YStack } from 'tamagui';

import tamaguiConfig from './tamagui.config';

interface HealthResponse {
  status: string;
  version?: string;
}

export function App() {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/health')
      .then((res) => res.json())
      .then((data: HealthResponse) => setHealth(data))
      .catch((err: Error) => setError(err.message));
  }, []);

  return (
    <TamaguiProvider
      config={tamaguiConfig}
      defaultTheme="light"
    >
      <YStack
        flex={1}
        alignItems="center"
        justifyContent="center"
        gap="$3"
      >
        <Text fontSize="$8" fontWeight="700">
          ReplayValidator
        </Text>
        <Text fontSize="$4" color="$color10">
          Bootstrap scaffold — no domain logic yet.
        </Text>
        {error && <Text color="$red10">API unreachable: {error}</Text>}
        {health && (
          <Text fontSize="$5">
            API status: {health.status} (version {health.version ?? 'unknown'})
          </Text>
        )}
        {!health && !error && <Text>Checking API…</Text>}
      </YStack>
    </TamaguiProvider>
  );
}

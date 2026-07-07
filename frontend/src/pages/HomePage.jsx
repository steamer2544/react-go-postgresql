import { useEffect, useState } from 'react';
import { checkHealth } from '@/services/healthService';

function HomePage() {
  const [healthStatus, setHealthStatus] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);

  useEffect(() => {
    checkHealth()
      .then((status) => setHealthStatus(status))
      .catch(() => setHasError(true))
      .finally(() => setIsLoading(false));
  }, []);

  if (isLoading) {
    return <p>Loading...</p>;
  }

  if (hasError) {
    return <p>Backend status: unavailable</p>;
  }

  return <p>Backend status: {healthStatus}</p>;
}

export default HomePage;

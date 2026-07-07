import axios from 'axios';

// Plain axios instance (not apiClient) because /healthz returns a raw
// { status: "ok" } body, not the standard { data, message } envelope
// that apiClient's interceptor unwraps.
const healthClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
});

export async function checkHealth() {
  const res = await healthClient.get('/healthz');
  return res.data.status;
}

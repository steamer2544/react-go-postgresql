import axios from 'axios';

// Central axios instance with standardized interceptors.
const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
});

// Request interceptor: attach Authorization header when a token exists.
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken');
  if (token) {
    config.headers.Authorization = 'Bearer ' + token;
  }
  return config;
});

// Response interceptor: unwrap the backend envelope { data, message }.
// On success, callers receive response.data.data directly so they never
// need to navigate the envelope themselves.
//
// Exception: requests with responseType 'arraybuffer' or 'blob' must
// return the full axios response so callers can read raw bytes and
// content-type headers (e.g. fetching a signature image).
apiClient.interceptors.response.use(
  (response) => {
    const rt = response.config.responseType;
    if (rt === 'arraybuffer' || rt === 'blob') {
      return response;
    }
    return response.data.data;
  },
  (error) => {
    // On 401, clear the stored token before rejecting.
    if (error.response && error.response.status === 401) {
      localStorage.removeItem('accessToken');
    }
    // Normalize error shapes so callers see .code and .message only.
    const err = new Error();
    if (error.response && error.response.data && error.response.data.error) {
      err.code = error.response.data.error.code;
      err.message = error.response.data.error.message;
    } else {
      // Network error or unknown shape — fall back gracefully.
      err.code = 'NETWORK_ERROR';
      err.message = error.message || 'An unexpected error occurred';
    }
    return Promise.reject(err);
  },
);

export default apiClient;

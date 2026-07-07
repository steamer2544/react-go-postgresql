import axios from "axios";

// Central axios instance with standardized interceptors.
const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
});

// Response interceptor: unwrap the backend envelope { data, message }.
// On success, callers receive response.data.data directly so they never
// need to navigate the envelope themselves.
apiClient.interceptors.response.use(
  (response) => response.data.data,
  (error) => {
    // Normalize error shapes so callers see .code and .message only.
    const err = new Error();
    if (error.response && error.response.data && error.response.data.error) {
      err.code = error.response.data.error.code;
      err.message = error.response.data.error.message;
    } else {
      // Network error or unknown shape — fall back gracefully.
      err.code = "NETWORK_ERROR";
      err.message = error.message || "An unexpected error occurred";
    }
    return Promise.reject(err);
  },
);

export default apiClient;

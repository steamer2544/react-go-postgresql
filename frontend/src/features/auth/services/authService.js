import apiClient from '@/services/apiClient';
import { AUTH_LOGIN, ME, ME_PROFILE, ME_SIGNATURE } from '@/constants/apiEndpoints';

export async function login(email, password) {
  return apiClient.post(AUTH_LOGIN, { email, password });
}

export async function getMe() {
  return apiClient.get(ME);
}

export async function updateProfile(payload) {
  return apiClient.put(ME_PROFILE, payload);
}

export async function uploadSignature(file) {
  const formData = new FormData();
  formData.append('signature', file);
  return apiClient.post(ME_SIGNATURE, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
}

export async function getSignatureUrl() {
  const response = await apiClient.get(ME_SIGNATURE, { responseType: 'arraybuffer' });
  const contentType = response.headers['content-type'] || 'image/png';
  const blob = new Blob([response.data], { type: contentType });
  return URL.createObjectURL(blob);
}

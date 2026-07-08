import apiClient from '@/services/apiClient';
import { QUOTATIONS } from '@/constants/apiEndpoints';

export async function list(params) {
  return apiClient.get(QUOTATIONS, { params });
}

export async function getById(id) {
  return apiClient.get(`${QUOTATIONS}/${id}`);
}

export async function create(payload) {
  return apiClient.post(QUOTATIONS, payload);
}

export async function update(id, payload) {
  return apiClient.put(`${QUOTATIONS}/${id}`, payload);
}

export async function remove(id) {
  return apiClient.delete(`${QUOTATIONS}/${id}`);
}

export async function submit(id) {
  return apiClient.post(`${QUOTATIONS}/${id}/submit`);
}

export async function approve(id) {
  return apiClient.post(`${QUOTATIONS}/${id}/approve`);
}

export async function reject(id) {
  return apiClient.post(`${QUOTATIONS}/${id}/reject`);
}

export async function getApprovalSignatureUrl(id) {
  const response = await apiClient.get(`${QUOTATIONS}/${id}/approval-signature`, {
    responseType: 'arraybuffer',
  });
  const contentType = response.headers['content-type'] || 'image/png';
  const blob = new Blob([response.data], { type: contentType });
  return URL.createObjectURL(blob);
}
